package storefront

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type GetWishlistUseCase struct {
	db           *gorm.DB
	wishlistRepo storefrontDomain.WishlistRepository
}

func NewGetWishlistUseCase(db *gorm.DB, wishlistRepo storefrontDomain.WishlistRepository) *GetWishlistUseCase {
	return &GetWishlistUseCase{db: db, wishlistRepo: wishlistRepo}
}

func (uc *GetWishlistUseCase) Execute(ctx context.Context, token string) (*storefrontDomain.Wishlist, error) {
	var wishlist *storefrontDomain.Wishlist
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		found, err := uc.wishlistRepo.FindByToken(ctx, token)
		if err != nil {
			return err
		}
		if found == nil {
			return storefrontDomain.ErrWishlistNotFound
		}
		wishlist = found
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getWishlist: %w", err)
	}
	return wishlist, nil
}
