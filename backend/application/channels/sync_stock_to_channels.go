package channels

import (
	"context"
	"fmt"
	"log"
	"maps"
	"strconv"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// SyncStockToChannelsUseCase re-pushes a SKU's current Syncable stock level
// to every channel its product is already linked to, in response to an
// inventory.stock.changed event. A SKU with no product listings is a
// cheap no-op — most SKUs aren't linked to any channel, and this runs on
// every stock movement, not just ones that happen to matter to a
// connected store.
type SyncStockToChannelsUseCase struct {
	db          *gorm.DB
	skuRepo     pmsDomain.SKURepository
	variantRepo pmsDomain.VariantRepository
	listingRepo channelsDomain.ProductChannelListingRepository
	syncLogRepo channelsDomain.SyncLogRepository
	connRepo    integrationsDomain.ConnectionRepository
	wooFactory  integrationsDomain.WooClientFactory
	stockLevel  stockLevelGetter
	encSecret   string
	bolPusher   bolProductPusher
}

func NewSyncStockToChannelsUseCase(
	db *gorm.DB,
	skuRepo pmsDomain.SKURepository,
	variantRepo pmsDomain.VariantRepository,
	listingRepo channelsDomain.ProductChannelListingRepository,
	syncLogRepo channelsDomain.SyncLogRepository,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	stockLevel stockLevelGetter,
	encryptionSecret string,
	bolPusher bolProductPusher,
) *SyncStockToChannelsUseCase {
	return &SyncStockToChannelsUseCase{
		db: db, skuRepo: skuRepo, variantRepo: variantRepo, listingRepo: listingRepo,
		syncLogRepo: syncLogRepo, connRepo: connRepo, wooFactory: wooFactory,
		stockLevel: stockLevel, encSecret: encryptionSecret, bolPusher: bolPusher,
	}
}

// Execute resolves the SKU's product, loads its listings, computes the
// current Syncable stock once, and fans it out to every linked channel.
// The returned error is reserved for infra-level failures (can't read
// from the DB) that warrant the caller nacking/retrying the whole event —
// a single channel being unreachable is recorded on that listing instead,
// never returned, so one flaky store can't block the whole event.
func (uc *SyncStockToChannelsUseCase) Execute(ctx context.Context, skuID string) error {
	return uc.run(ctx, skuID, false)
}

// ForceExecute is Execute's counterpart for a caller that wants the current
// stock level re-pushed regardless of what the last successful sync already
// reported (a manual "resync to channels" action, or the periodic bulk
// resync job) -- it skips syncOne's alreadySyncedTo short-circuit, which
// exists to avoid redundant Woo calls on the routine
// inventory.stock.changed path but would defeat the point of a deliberate
// force-resync.
func (uc *SyncStockToChannelsUseCase) ForceExecute(ctx context.Context, skuID string) error {
	return uc.run(ctx, skuID, true)
}

func (uc *SyncStockToChannelsUseCase) run(ctx context.Context, skuID string, force bool) error {
	var productID string
	var listings []channelsDomain.ProductChannelListing

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		variant, err := uc.variantRepo.FindByID(ctx, sku.VariantID)
		if err != nil {
			return err
		}
		productID = variant.ProductID

		listings, err = uc.listingRepo.ListByProduct(ctx, productID)
		return err
	})
	if err != nil {
		return fmt.Errorf("syncStockToChannels: %w", err)
	}
	if len(listings) == 0 {
		return nil
	}

	level, err := uc.stockLevel.Execute(ctx, skuID)
	if err != nil {
		return fmt.Errorf("syncStockToChannels: %w", err)
	}
	qty := strconv.FormatInt(level.Syncable, 10)
	fields := map[string]string{"stock_quantity": qty}
	metadata := toWooMetaData(level.SourceBatch)
	for _, md := range metadata {
		fields[md.Key] = md.Value
	}

	pushedBolConnections := map[string]bool{}
	for _, listing := range listings {
		uc.syncOne(ctx, productID, listing, parentExternalIDFor(listings, listing), qty, fields, metadata, force, pushedBolConnections)
	}
	return nil
}

// parentExternalIDFor resolves the Woo variation-update endpoint's required
// parent product id for a listing with IsVariation=true, by finding the
// sibling product-level row (VariantID nil) for the same connection --
// listings for one product+connection are always loaded together by
// Execute, so the sibling is already in hand. Returns "" for a
// non-variation listing, which UpdateProductStock takes as "this is a
// standalone product".
func parentExternalIDFor(listings []channelsDomain.ProductChannelListing, listing channelsDomain.ProductChannelListing) string {
	if !listing.IsVariation {
		return ""
	}
	for _, l := range listings {
		if l.ConnectionID == listing.ConnectionID && l.VariantID == nil {
			return l.ExternalID
		}
	}
	return ""
}

// syncOne dispatches a listing to its provider's own stock-sync logic --
// new provider support is one new case (and, for anything needing more than
// a fire-and-forget push, one new method), never a change to the providers
// already handled here.
func (uc *SyncStockToChannelsUseCase) syncOne(ctx context.Context, productID string, listing channelsDomain.ProductChannelListing, parentExternalID, qty string, fields map[string]string, metadata []integrationsDomain.WooMetaData, force bool, pushedBolConnections map[string]bool) {
	switch integrationsDomain.Provider(listing.Provider) {
	case integrationsDomain.ProviderWooCommerce:
		uc.syncWoo(ctx, listing, parentExternalID, qty, fields, metadata, force)
	case integrationsDomain.ProviderBol:
		uc.syncBol(ctx, productID, listing, pushedBolConnections)
	default:
		return // no other provider implemented yet
	}
}

// syncBol always re-pushes the product's full Bol offer for a connection
// the first time that connection is seen in this run (pushedBolConnections
// dedupes repeat calls for a multi-variant product's several listings under
// the same connection). Bol has no stock-only endpoint -- every push sends
// price+stock together, see RunScheduledBolExportUseCase's doc comment --
// so there is no cheaper single-field call available the way syncWoo has.
//
// Unlike syncWoo there is deliberately no "skip if unchanged" guard: the
// whole point of this use case is that every inventory change reaches every
// channel the product is enrolled in (a Bol listing existing here already
// means EnrollProductInChannelsUseCase decided this product belongs on Bol
// -- that's the "should this channel get it" decision, made once at
// enrollment time, not re-litigated per stock event). Letting Bol's push
// lag behind on the bol_stock schedule's interval is exactly the staleness
// this fixes. bolPusher is optional/nil -- same convention as
// SyncProductToChannelsUseCase.bolPusher -- for a caller that hasn't wired
// Bol support.
func (uc *SyncStockToChannelsUseCase) syncBol(ctx context.Context, productID string, listing channelsDomain.ProductChannelListing, pushedBolConnections map[string]bool) {
	if uc.bolPusher == nil || pushedBolConnections[listing.ConnectionID] {
		return
	}
	pushedBolConnections[listing.ConnectionID] = true
	if _, err := uc.bolPusher.Execute(ctx, PushProductToBolInput{ProductID: productID, ConnectionID: listing.ConnectionID}); err != nil {
		log.Printf("syncStockToChannels: bol product %s -> connection %s: %v", productID, listing.ConnectionID, err)
	}
}

func (uc *SyncStockToChannelsUseCase) syncWoo(ctx context.Context, listing channelsDomain.ProductChannelListing, parentExternalID, qty string, fields map[string]string, metadata []integrationsDomain.WooMetaData, force bool) {
	// Skip the actual Woo call (not just the log row — recordSyncLog
	// already dedupes those) when the last recorded push for this listing
	// already reported these exact field values. A stock movement usually
	// does change Syncable, but not always (e.g. an adjustment against an
	// already-expired batch, which Syncable excludes either way); the
	// nearest-expiry batch can also change lot with the quantity staying
	// the same, which fields (not just qty) catches. force (a deliberate
	// resync) always bypasses this -- the caller wants the push to happen
	// regardless of whether anything looks unchanged.
	if !force && uc.alreadySyncedTo(ctx, listing.ID, fields) {
		return
	}

	syncErr := uc.pushToWoo(ctx, listing, parentExternalID, qty, metadata)

	now := time.Now()
	listing.LastSyncedAt = &now
	listing.UpdatedAt = now
	status := channelsDomain.SyncLogStatusSuccess
	errMsg := ""
	if syncErr != nil {
		listing.SyncStatus = channelsDomain.SyncStatusError
		listing.LastSyncError = syncErr.Error()
		status = channelsDomain.SyncLogStatusError
		errMsg = syncErr.Error()
		log.Printf("syncStockToChannels: connection %s: %v", listing.ConnectionID, syncErr)
	} else {
		listing.SyncStatus = channelsDomain.SyncStatusSynced
		listing.LastSyncError = ""
	}

	recordSyncLog(ctx, uc.db, uc.syncLogRepo, listing, channelsDomain.SyncFlowPush, status, errMsg, fields)

	if uErr := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.listingRepo.Update(ctx, &listing)
	}); uErr != nil {
		log.Printf("syncStockToChannels: recording sync result for listing %s: %v", listing.ID, uErr)
	}
}

func (uc *SyncStockToChannelsUseCase) alreadySyncedTo(ctx context.Context, listingID string, fields map[string]string) bool {
	var prev []channelsDomain.SyncLogEntry
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		prev, err = uc.syncLogRepo.ListByListing(ctx, listingID, 1)
		return err
	})
	if err != nil || len(prev) == 0 {
		return false
	}
	return prev[0].Status == channelsDomain.SyncLogStatusSuccess && maps.Equal(prev[0].Fields, fields)
}

func (uc *SyncStockToChannelsUseCase) pushToWoo(ctx context.Context, listing channelsDomain.ProductChannelListing, parentExternalID, qty string, metadata []integrationsDomain.WooMetaData) error {
	var conn integrationsDomain.Connection
	var secret string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, listing.ConnectionID)
		if err != nil {
			return err
		}
		conn = *c
		secret, err = crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
		return err
	})
	if err != nil {
		return err
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return fmt.Errorf("%w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}

	qtyInt, err := strconv.Atoi(qty)
	if err != nil {
		return fmt.Errorf("syncStockToChannels: parse quantity: %w", err)
	}
	if err := client.UpdateProductStock(ctx, parentExternalID, listing.ExternalID, qtyInt, metadata); err != nil {
		return fmt.Errorf("%w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}
	return nil
}
