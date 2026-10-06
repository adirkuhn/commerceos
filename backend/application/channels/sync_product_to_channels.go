package channels

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"log"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// bolProductPusher is the one method this use case needs from
// PushProductToBolUseCase (ISP), same reasoning as productPusher above.
// Optional: nil when the caller hasn't wired Bol support (e.g. an older
// call site not yet updated) -- syncBolConnection no-ops rather than
// panicking, since Bol support is additive and every existing call site
// predates it.
type bolProductPusher interface {
	Execute(ctx context.Context, in PushProductToBolInput) ([]channelsDomain.ProductChannelListing, error)
}

// bolOfferGater is the one method this use case needs from
// PushProductToBolUseCase to gate/ungate a Bol listing (ISP) -- a minimal
// on-hold toggle, never a full offer push. Optional: nil when the caller
// hasn't wired Bol support, same convention as bolPusher.
type bolOfferGater interface {
	SetOnHold(ctx context.Context, connectionID, externalID string, onHold bool) error
}

// SyncProductToChannelsUseCase re-pushes a product's current state to every
// channel it's already linked to, in response to a pms.product.changed
// event. A product with no listings is a cheap no-op — most products
// aren't linked to any channel, and this runs on every core product/SKU
// edit, not just ones that happen to matter to a connected store.
//
// A product/variant whose SKU count isn't exactly 1 is silently skipped
// rather than erroring — same invariant PushProductToWooUseCase enforces
// on the initial push.
//
// When the actual shape of a connection's listings no longer matches what
// the product's HasVariants/SyncVariantsAsWooVariations flags call for
// (the flag toggled since the last push, or the product grew/lost
// variants), this deletes the old external representation and every old
// listing row for that connection, then re-pushes via PushProductToWooUseCase
// to recreate it in the new shape from scratch.
//
// Stock quantity is deliberately not pushed here — that's the sync
// schedule's job (ExportWooStockLevelsUseCase); duplicating it here would
// just mean two code paths racing to set the same field.
type SyncProductToChannelsUseCase struct {
	db           *gorm.DB
	productRepo  pmsDomain.ProductRepository
	variantRepo  pmsDomain.VariantRepository
	skuRepo      pmsDomain.SKURepository
	listingRepo  channelsDomain.ProductChannelListingRepository
	syncLogRepo  channelsDomain.SyncLogRepository
	connRepo     integrationsDomain.ConnectionRepository
	wooFactory   integrationsDomain.WooClientFactory
	encSecret    string
	pusher       productPusher
	bolPusher    bolProductPusher
	bolGater     bolOfferGater
	stockCatchUp productStockForceSyncer
	mediaRepo    productMediaLister
	categoryLink categoryLinkResolver
	outbox       eventrecord.Enqueuer
}

func NewSyncProductToChannelsUseCase(
	db *gorm.DB,
	productRepo pmsDomain.ProductRepository,
	variantRepo pmsDomain.VariantRepository,
	skuRepo pmsDomain.SKURepository,
	listingRepo channelsDomain.ProductChannelListingRepository,
	syncLogRepo channelsDomain.SyncLogRepository,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
	pusher productPusher,
	mediaRepo productMediaLister,
	categoryLink categoryLinkResolver,
	bolPusher bolProductPusher,
	bolGater bolOfferGater,
	stockCatchUp productStockForceSyncer,
	outbox eventrecord.Enqueuer,
) *SyncProductToChannelsUseCase {
	return &SyncProductToChannelsUseCase{
		db: db, productRepo: productRepo, variantRepo: variantRepo, skuRepo: skuRepo,
		listingRepo: listingRepo, syncLogRepo: syncLogRepo,
		connRepo: connRepo, wooFactory: wooFactory, encSecret: encryptionSecret, pusher: pusher,
		mediaRepo: mediaRepo, categoryLink: categoryLink, bolPusher: bolPusher,
		bolGater: bolGater, stockCatchUp: stockCatchUp,
		outbox: outbox,
	}
}

// Execute loads the product's fresh state and fans it out to every linked
// channel. The returned error is reserved for infra-level failures (can't
// read from the DB) that warrant the caller nacking/retrying the whole
// event — a single channel being unreachable is recorded on that listing's
// SyncStatus instead, never returned, so one flaky store can't block
// (or endlessly retry) the whole event.
func (uc *SyncProductToChannelsUseCase) Execute(ctx context.Context, productID string) error {
	var product pmsDomain.Product
	var variants []pmsDomain.Variant
	skuByVariant := map[string]pmsDomain.SKU{}
	var listings []channelsDomain.ProductChannelListing
	var media []pmsDomain.Media

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		listings, err = uc.listingRepo.ListByProduct(ctx, productID)
		if err != nil {
			return err
		}
		if len(listings) == 0 {
			return nil
		}

		p, err := uc.productRepo.FindByID(ctx, productID)
		if err != nil {
			return err
		}
		product = *p

		media, err = uc.mediaRepo.ListByProduct(ctx, productID)
		if err != nil {
			return err
		}

		variants, err = uc.variantRepo.ListByProduct(ctx, productID)
		if err != nil {
			return err
		}
		for _, v := range variants {
			skus, err := uc.skuRepo.ListByVariant(ctx, v.ID)
			if err != nil {
				return err
			}
			// A bulk pool (UnitType=kilo) SKU is never itself sellable --
			// excluded here so it's treated the same as a "removed"
			// variant everywhere below: never (re)created on the channel,
			// and any stray listing for it gets cleaned up.
			if len(skus) == 1 && skus[0].UnitType != pmsDomain.SKUUnitTypeKilo {
				skuByVariant[v.ID] = skus[0]
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("syncProductToChannels: %w", err)
	}
	if len(listings) == 0 {
		return nil
	}

	byConnection := map[string][]channelsDomain.ProductChannelListing{}
	var order []string
	for _, l := range listings {
		if _, ok := byConnection[l.ConnectionID]; !ok {
			order = append(order, l.ConnectionID)
		}
		byConnection[l.ConnectionID] = append(byConnection[l.ConnectionID], l)
	}

	images := wooImagesFromMedia(media, product.Name)
	for _, connectionID := range order {
		uc.syncConnection(ctx, product, variants, skuByVariant, connectionID, byConnection[connectionID], images)
	}
	return nil
}

// syncConnection is gated by the product's own Active/Draft/Archived status
// before any of the existing shape logic runs -- see
// pms.Product.IsSyncableToChannels and ProductChannelListing.GatedAt's doc
// comments for the full contract this enforces:
//
//   - Active, never gated: falls straight through to the shape logic below,
//     unchanged from before gating existed.
//   - Just left Active (GatedAt nil, product not Active): gateConnection
//     pushes every existing row's current fields one last time (Woo: full
//     field payload including the new draft/archived status; Bol: an
//     onHoldByRetailer=true toggle), creates/deletes nothing, then marks
//     every row gated.
//   - Still not Active, already gated: return immediately, zero API calls --
//     "avoid create or update ... if the product is in draft or archived
//     status."
//   - Returned to Active from gated: ungateConnection clears the gate (Woo:
//     nothing extra needed, the shape logic below already re-pushes full
//     fields including the new Active status; Bol: onHoldByRetailer=false),
//     forces a stock catch-up for every SKU, then falls through to the
//     normal shape logic so anything that changed while gated (including a
//     variant added or removed) is reconciled immediately.
func (uc *SyncProductToChannelsUseCase) syncConnection(ctx context.Context, product pmsDomain.Product, variants []pmsDomain.Variant, skuByVariant map[string]pmsDomain.SKU, connectionID string, listings []channelsDomain.ProductChannelListing, images []integrationsDomain.WooProductImage) {
	active := product.IsSyncableToChannels()
	wasGated := listings[0].IsGated()

	if !active {
		if !wasGated {
			uc.gateConnection(ctx, product, variants, skuByVariant, connectionID, listings, images)
		}
		return
	}

	if wasGated {
		uc.ungateConnection(ctx, connectionID, listings)
		// Clear the in-memory copies too -- the shape logic below reads and
		// re-persists these same listing structs (recordSyncResult etc.),
		// and without this it would write the stale non-nil GatedAt right
		// back over the clear ungateConnection just persisted.
		for i := range listings {
			listings[i].GatedAt = nil
		}
		uc.forceStockCatchUp(ctx, skuByVariant)
	}

	switch integrationsDomain.Provider(listings[0].Provider) {
	case integrationsDomain.ProviderWooCommerce:
		// falls through to the existing shape-based logic below
	case integrationsDomain.ProviderBol:
		uc.syncBolConnection(ctx, product, connectionID)
		return
	default:
		return // no other provider implemented yet
	}

	desired := desiredShape(product)
	actual := actualShape(listings)
	if actual != desired {
		uc.migrateShape(ctx, product, connectionID, listings)
		return
	}

	switch desired {
	case shapeSingleSKU:
		sku, ok := skuByVariant[variants[0].ID]
		if !ok {
			return
		}
		uc.syncOne(ctx, product, sku, listings[0], images)
	case shapeSeparateProducts:
		uc.syncSeparateProducts(ctx, product, variants, skuByVariant, connectionID, listings, images)
	case shapeVariations:
		uc.syncVariations(ctx, product, variants, skuByVariant, connectionID, listings, images)
	}
}

// syncBolConnection just re-runs PushProductToBolUseCase -- unlike Woo's
// three shapes, Bol offers have no "variation" mode to migrate between, and
// PushProductToBolUseCase is already idempotent (it looks up each
// variant's existing listing and PATCHes instead of re-creating), so
// there's no separate update-in-place logic to duplicate here.
func (uc *SyncProductToChannelsUseCase) syncBolConnection(ctx context.Context, product pmsDomain.Product, connectionID string) {
	if uc.bolPusher == nil {
		return
	}
	if _, err := uc.bolPusher.Execute(ctx, PushProductToBolInput{ProductID: product.ID, ConnectionID: connectionID}); err != nil {
		log.Printf("syncProductToChannels: bol product %s -> connection %s: %v", product.ID, connectionID, err)
	}
}

func (uc *SyncProductToChannelsUseCase) recordSyncResult(ctx context.Context, listing channelsDomain.ProductChannelListing, syncErr error, fields map[string]string) {
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
		log.Printf("syncProductToChannels: listing %s: %v", listing.ID, syncErr)
	} else {
		listing.SyncStatus = channelsDomain.SyncStatusSynced
		listing.LastSyncError = ""
	}

	recordSyncLog(ctx, uc.db, uc.syncLogRepo, uc.outbox, listing, channelsDomain.SyncFlowPush, status, errMsg, fields)

	if uErr := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.listingRepo.Update(ctx, &listing)
	}); uErr != nil {
		log.Printf("syncProductToChannels: recording sync result for listing %s: %v", listing.ID, uErr)
	}
}
