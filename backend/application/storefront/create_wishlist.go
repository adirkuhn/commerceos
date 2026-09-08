package storefront

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type CreateWishlistUseCase struct {
	db           *gorm.DB
	channelRepo  channelsDomain.SalesChannelRepository
	wishlistRepo storefrontDomain.WishlistRepository
}

func NewCreateWishlistUseCase(db *gorm.DB, channelRepo channelsDomain.SalesChannelRepository, wishlistRepo storefrontDomain.WishlistRepository) *CreateWishlistUseCase {
	return &CreateWishlistUseCase{db: db, channelRepo: channelRepo, wishlistRepo: wishlistRepo}
}

// Execute starts a new, empty wishlist on salesChannelID -- same
// fresh-random-token identity model as CreateCartUseCase (see its doc
// comment).
func (uc *CreateWishlistUseCase) Execute(ctx context.Context, salesChannelID string) (*storefrontDomain.Wishlist, error) {
	var wishlist storefrontDomain.Wishlist
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := findInternalChannel(ctx, uc.channelRepo, salesChannelID); err != nil {
			return err
		}
		now := time.Now()
		wishlist = storefrontDomain.Wishlist{
			TenantID: tenant.IDFromContext(ctx), SalesChannelID: salesChannelID,
			Token: idgen.NewV4(), ProductIDs: nil, CreatedAt: now, UpdatedAt: now,
		}
		return uc.wishlistRepo.Create(ctx, &wishlist)
	})
	if err != nil {
		return nil, fmt.Errorf("createWishlist: %w", err)
	}
	return &wishlist, nil
}
