package channels

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// Narrow ports for HandleMissingListingUseCase (ISP).
type listingStoreForMissing interface {
	ListByProductAndConnection(ctx context.Context, productID, connectionID string) ([]channelsDomain.ProductChannelListing, error)
	Delete(ctx context.Context, id string) error
}

type productFinderForMissing interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Product, error)
}

type salesChannelListerForMissing interface {
	ListByTenant(ctx context.Context) ([]channelsDomain.SalesChannel, error)
}

type channelOverrideFinderForMissing interface {
	FindByChannelAndProduct(ctx context.Context, channelID, productID string) (*channelsDomain.SalesChannelProductOverride, error)
}

// HandleMissingListingInput names the product whose listing on a
// connection points at a channel product that no longer exists
// (channels.product_listing.orphaned).
type HandleMissingListingInput struct {
	ProductID    string
	ConnectionID string
}

// HandleMissingListingUseCase decides what happens to a product deleted in
// WooCommerce (COM-18): when the sales channel still allows it -- the
// product is Active and enrolled in an active WooCommerce sales channel on
// that connection (its rule or a per-product override) -- the stale
// listing rows are removed and the product is created in the store again
// (PushProductToWooUseCase, which records the new listing). Otherwise the
// listing stays marked missing and nothing is pushed. Nothing missing any
// more is a no-op, so a repeat (or a later reconciliation) is harmless.
type HandleMissingListingUseCase struct {
	db        *gorm.DB
	listings  listingStoreForMissing
	products  productFinderForMissing
	channels  salesChannelListerForMissing
	overrides channelOverrideFinderForMissing
	pusher    productPusher
	outbox    eventrecord.Enqueuer
}

func NewHandleMissingListingUseCase(db *gorm.DB, listings listingStoreForMissing, products productFinderForMissing,
	channels salesChannelListerForMissing, overrides channelOverrideFinderForMissing, pusher productPusher,
	outbox eventrecord.Enqueuer) *HandleMissingListingUseCase {
	return &HandleMissingListingUseCase{db: db, listings: listings, products: products, channels: channels,
		overrides: overrides, pusher: pusher, outbox: outbox}
}

func (uc *HandleMissingListingUseCase) Execute(ctx context.Context, in HandleMissingListingInput) error {
	var recreate bool
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		rows, err := uc.listings.ListByProductAndConnection(ctx, in.ProductID, in.ConnectionID)
		if err != nil {
			return err
		}
		if !anyMissing(rows) {
			return nil
		}
		recreate, err = uc.channelAllows(ctx, in)
		if err != nil || !recreate {
			return err
		}
		for _, l := range rows {
			if err := uc.listings.Delete(ctx, l.ID); err != nil {
				return err
			}
			if err := recordListingDeleted(ctx, uc.outbox, l, ListingDeletedReasonMissingInChannel); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("handleMissingListing: %w", err)
	}
	if !recreate {
		return nil
	}
	// its own transactions -- tenant.WithTenant must never nest
	if _, err := uc.pusher.Execute(ctx, PushProductToWooInput{ProductID: in.ProductID, ConnectionID: in.ConnectionID}); err != nil {
		return fmt.Errorf("handleMissingListing: recreate product %s: %w", in.ProductID, err)
	}
	return nil
}

// channelAllows: the product is Active and enrolled in an active
// WooCommerce sales channel on this connection.
func (uc *HandleMissingListingUseCase) channelAllows(ctx context.Context, in HandleMissingListingInput) (bool, error) {
	product, err := uc.products.FindByID(ctx, in.ProductID)
	if err != nil {
		return false, err
	}
	if !product.IsSyncableToChannels() {
		return false, nil
	}
	channels, err := uc.channels.ListByTenant(ctx)
	if err != nil {
		return false, err
	}
	for _, ch := range channels {
		if !ch.Active || ch.Type != channelsDomain.ChannelTypeWooCommerce || ch.ConnectionID == nil || *ch.ConnectionID != in.ConnectionID {
			continue
		}
		override, err := uc.overrides.FindByChannelAndProduct(ctx, ch.ID, in.ProductID)
		if err != nil {
			return false, err
		}
		if ch.IsProductEnrolled(product.CategoryID, product.Tags, override) {
			return true, nil
		}
	}
	return false, nil
}

func anyMissing(listings []channelsDomain.ProductChannelListing) bool {
	for _, l := range listings {
		if l.SyncStatus == channelsDomain.SyncStatusMissing {
			return true
		}
	}
	return false
}
