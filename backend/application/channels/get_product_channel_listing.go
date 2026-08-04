package channels

import (
	"context"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// GetProductChannelListingUseCase powers the product edit page's
// per-channel tab (e.g. "WooCommerce") — it needs to know whether this
// product is linked to that connection at all, and if so its sync state,
// without the caller having to know the ProductChannelListing repository
// exists.
type GetProductChannelListingUseCase struct {
	db          *gorm.DB
	listingRepo channelsDomain.ProductChannelListingRepository
}

func NewGetProductChannelListingUseCase(db *gorm.DB, listingRepo channelsDomain.ProductChannelListingRepository) *GetProductChannelListingUseCase {
	return &GetProductChannelListingUseCase{db: db, listingRepo: listingRepo}
}

func (uc *GetProductChannelListingUseCase) Execute(ctx context.Context, productID, connectionID string) (*channelsDomain.ProductChannelListing, error) {
	var listing channelsDomain.ProductChannelListing
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.listingRepo.FindByProductAndConnection(ctx, productID, connectionID)
		if err != nil {
			return err
		}
		listing = *existing
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getProductChannelListing: %w", err)
	}
	return &listing, nil
}
