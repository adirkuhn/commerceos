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

type CreateCartUseCase struct {
	db          *gorm.DB
	channelRepo channelsDomain.SalesChannelRepository
	cartRepo    storefrontDomain.CartRepository
}

func NewCreateCartUseCase(db *gorm.DB, channelRepo channelsDomain.SalesChannelRepository, cartRepo storefrontDomain.CartRepository) *CreateCartUseCase {
	return &CreateCartUseCase{db: db, channelRepo: channelRepo, cartRepo: cartRepo}
}

// Execute starts a new, empty cart on salesChannelID -- the channel a
// storefront's own API key resolved to (see apikeys.ResolvedAPIKey). Token
// is a fresh random UUID, same generator idgen.NewV4 uses for outbox event
// IDs -- unguessable, and the only thing that ever needs to address this
// cart again.
func (uc *CreateCartUseCase) Execute(ctx context.Context, salesChannelID string) (*storefrontDomain.Cart, error) {
	var cart storefrontDomain.Cart
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := findInternalChannel(ctx, uc.channelRepo, salesChannelID); err != nil {
			return err
		}
		now := time.Now()
		cart = storefrontDomain.Cart{
			TenantID: tenant.IDFromContext(ctx), SalesChannelID: salesChannelID,
			Token: idgen.NewV4(), Items: nil, CreatedAt: now, UpdatedAt: now,
		}
		return uc.cartRepo.Create(ctx, &cart)
	})
	if err != nil {
		return nil, fmt.Errorf("createCart: %w", err)
	}
	return &cart, nil
}
