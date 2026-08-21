package scheduler

import (
	"context"
	"log"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// RunResyncProductsToChannelsUseCase force-pushes every linked product's
// current fields/stock to its channels, across every tenant -- catches
// drift the event-driven paths (SyncProductToChannelsUseCase,
// SyncStockToChannelsUseCase) can miss: a channel-side edit, or an event
// that failed and was dismissed rather than retried. Disabled by default
// (see migration 088) -- this is real per-product Woo API traffic, not a
// cheap sweep. A failure on one product is logged, not fatal to the rest.
type RunResyncProductsToChannelsUseCase struct {
	db          *gorm.DB
	listingRepo channelsDomain.ProductChannelListingRepository
	resyncUC    *appchannels.ResyncProductToChannelsUseCase
}

func NewRunResyncProductsToChannelsUseCase(db *gorm.DB, listingRepo channelsDomain.ProductChannelListingRepository, resyncUC *appchannels.ResyncProductToChannelsUseCase) *RunResyncProductsToChannelsUseCase {
	return &RunResyncProductsToChannelsUseCase{db: db, listingRepo: listingRepo, resyncUC: resyncUC}
}

func (uc *RunResyncProductsToChannelsUseCase) Run(ctx context.Context) error {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var listings []channelsDomain.ProductChannelListing
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		var err error
		listings, err = uc.listingRepo.ListAll(ctx)
		return err
	})
	if err != nil {
		return err
	}

	// Every product with at least one channel listing needs exactly one
	// resync, regardless of how many listings/connections it has --
	// ResyncProductToChannelsUseCase already fans out to all of them
	// internally.
	type productKey struct{ tenantID, productID string }
	seen := make(map[productKey]bool, len(listings))
	for _, l := range listings {
		key := productKey{tenantID: l.TenantID, productID: l.ProductID}
		if seen[key] {
			continue
		}
		seen[key] = true

		runCtx := tenant.WithTenantID(adminCtx, l.TenantID)
		if err := uc.resyncUC.Execute(runCtx, l.ProductID); err != nil {
			log.Printf("resync product %s to channels: %v", l.ProductID, err)
		}
	}
	return nil
}
