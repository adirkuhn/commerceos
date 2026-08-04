package channels

import (
	"context"
	"fmt"
	"log"
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
) *SyncStockToChannelsUseCase {
	return &SyncStockToChannelsUseCase{
		db: db, skuRepo: skuRepo, variantRepo: variantRepo, listingRepo: listingRepo,
		syncLogRepo: syncLogRepo, connRepo: connRepo, wooFactory: wooFactory,
		stockLevel: stockLevel, encSecret: encryptionSecret,
	}
}

// Execute resolves the SKU's product, loads its listings, computes the
// current Syncable stock once, and fans it out to every linked channel.
// The returned error is reserved for infra-level failures (can't read
// from the DB) that warrant the caller nacking/retrying the whole event —
// a single channel being unreachable is recorded on that listing instead,
// never returned, so one flaky store can't block the whole event.
func (uc *SyncStockToChannelsUseCase) Execute(ctx context.Context, skuID string) error {
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

	for _, listing := range listings {
		uc.syncOne(ctx, listing, qty)
	}
	return nil
}

func (uc *SyncStockToChannelsUseCase) syncOne(ctx context.Context, listing channelsDomain.ProductChannelListing, qty string) {
	if listing.Provider != string(integrationsDomain.ProviderWooCommerce) {
		return // no other provider implemented yet
	}
	fields := map[string]string{"stock_quantity": qty}

	// Skip the actual Woo call (not just the log row — recordSyncLog
	// already dedupes those) when the last recorded push for this listing
	// already reported this exact quantity. A stock movement usually does
	// change Syncable, but not always (e.g. an adjustment against an
	// already-expired batch, which Syncable excludes either way).
	if uc.alreadySyncedTo(ctx, listing.ID, qty) {
		return
	}

	syncErr := uc.pushToWoo(ctx, listing, qty)

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

func (uc *SyncStockToChannelsUseCase) alreadySyncedTo(ctx context.Context, listingID, qty string) bool {
	var prev []channelsDomain.SyncLogEntry
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		prev, err = uc.syncLogRepo.ListByListing(ctx, listingID, 1)
		return err
	})
	if err != nil || len(prev) == 0 {
		return false
	}
	return prev[0].Status == channelsDomain.SyncLogStatusSuccess && prev[0].Fields["stock_quantity"] == qty
}

func (uc *SyncStockToChannelsUseCase) pushToWoo(ctx context.Context, listing channelsDomain.ProductChannelListing, qty string) error {
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
	if err := client.UpdateProductStock(ctx, listing.ExternalID, qtyInt); err != nil {
		return fmt.Errorf("%w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}
	return nil
}
