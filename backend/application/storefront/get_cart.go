package storefront

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type GetCartUseCase struct {
	db       *gorm.DB
	cartRepo storefrontDomain.CartRepository
}

func NewGetCartUseCase(db *gorm.DB, cartRepo storefrontDomain.CartRepository) *GetCartUseCase {
	return &GetCartUseCase{db: db, cartRepo: cartRepo}
}

func (uc *GetCartUseCase) Execute(ctx context.Context, token string) (*storefrontDomain.Cart, error) {
	var cart *storefrontDomain.Cart
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		found, err := uc.cartRepo.FindByToken(ctx, token)
		if err != nil {
			return err
		}
		if found == nil {
			return storefrontDomain.ErrCartNotFound
		}
		cart = found
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getCart: %w", err)
	}
	return cart, nil
}
