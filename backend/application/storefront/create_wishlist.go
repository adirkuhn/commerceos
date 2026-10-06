package storefront

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type CreateWishlistUseCase struct {
	db           *gorm.DB
	channelRepo  channelsDomain.SalesChannelRepository
	wishlistRepo storefrontDomain.WishlistRepository
	outbox       eventrecord.Enqueuer
}

func NewCreateWishlistUseCase(db *gorm.DB, channelRepo channelsDomain.SalesChannelRepository, wishlistRepo storefrontDomain.WishlistRepository, outbox eventrecord.Enqueuer) *CreateWishlistUseCase {
	return &CreateWishlistUseCase{db: db, channelRepo: channelRepo, wishlistRepo: wishlistRepo, outbox: outbox}
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
		if err := uc.wishlistRepo.Create(ctx, &wishlist); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.WishlistCreated, wishlist.ID, eventrecord.Data{"sales_channel_id": wishlist.SalesChannelID})
	})
	if err != nil {
		return nil, fmt.Errorf("createWishlist: %w", err)
	}
	return &wishlist, nil
}
