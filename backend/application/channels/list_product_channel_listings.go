package channels

import (
	"context"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ListProductChannelListingsUseCase powers the product edit page's
// per-channel tab for a multi-variant product -- GetProductChannelListingUseCase
// only ever returns one representative row (the product-level row when one
// exists, else an arbitrary variant-level row), which collapses a
// separate-products or variations push (one row per variant, plus a
// parent in variations mode) down to a single, potentially misleading
// summary. This returns every row for the (product, connection) pair, so
// the UI can show one entry per variant.
type ListProductChannelListingsUseCase struct {
	db          *gorm.DB
	listingRepo channelsDomain.ProductChannelListingRepository
}

func NewListProductChannelListingsUseCase(db *gorm.DB, listingRepo channelsDomain.ProductChannelListingRepository) *ListProductChannelListingsUseCase {
	return &ListProductChannelListingsUseCase{db: db, listingRepo: listingRepo}
}

func (uc *ListProductChannelListingsUseCase) Execute(ctx context.Context, productID, connectionID string) ([]channelsDomain.ProductChannelListing, error) {
	var listings []channelsDomain.ProductChannelListing
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		listings, err = uc.listingRepo.ListByProductAndConnection(ctx, productID, connectionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listProductChannelListings: %w", err)
	}
	return listings, nil
}
