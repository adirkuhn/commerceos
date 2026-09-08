package storefront

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ToggleWishlistItemUseCase struct {
	db           *gorm.DB
	wishlistRepo storefrontDomain.WishlistRepository
	channelRepo  channelsDomain.SalesChannelRepository
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository
	productRepo  pmsDomain.ProductRepository
}

func NewToggleWishlistItemUseCase(
	db *gorm.DB,
	wishlistRepo storefrontDomain.WishlistRepository,
	channelRepo channelsDomain.SalesChannelRepository,
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository,
	productRepo pmsDomain.ProductRepository,
) *ToggleWishlistItemUseCase {
	return &ToggleWishlistItemUseCase{db: db, wishlistRepo: wishlistRepo, channelRepo: channelRepo, overrideRepo: overrideRepo, productRepo: productRepo}
}

// Execute adds productID to the wishlist if absent, or removes it if
// already present -- one call per heart-icon click, no separate add/remove
// endpoints needed. Availability (enrolled + active, see
// SubmitReviewUseCase's identical check) is only enforced on the adding
// path -- a shopper can always remove something already saved, even if the
// product has since been deactivated or unenrolled.
func (uc *ToggleWishlistItemUseCase) Execute(ctx context.Context, token, productID string) (*storefrontDomain.Wishlist, error) {
	var wishlist *storefrontDomain.Wishlist
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		found, err := uc.wishlistRepo.FindByToken(ctx, token)
		if err != nil {
			return err
		}
		if found == nil {
			return storefrontDomain.ErrWishlistNotFound
		}

		if found.Contains(productID) {
			filtered := make([]string, 0, len(found.ProductIDs))
			for _, id := range found.ProductIDs {
				if id != productID {
					filtered = append(filtered, id)
				}
			}
			found.ProductIDs = filtered
		} else {
			channel, err := findInternalChannel(ctx, uc.channelRepo, found.SalesChannelID)
			if err != nil {
				return err
			}
			product, err := uc.productRepo.FindByID(ctx, productID)
			if err != nil {
				if errors.Is(err, pmsDomain.ErrProductNotFound) {
					return storefrontDomain.ErrProductNotAvailable
				}
				return err
			}
			if product.Status != pmsDomain.ProductStatusActive {
				return storefrontDomain.ErrProductNotAvailable
			}
			override, err := uc.overrideRepo.FindByChannelAndProduct(ctx, found.SalesChannelID, productID)
			if err != nil {
				return err
			}
			if !channel.IsProductEnrolled(product.CategoryID, product.Tags, override) {
				return storefrontDomain.ErrProductNotAvailable
			}
			found.ProductIDs = append(found.ProductIDs, productID)
		}
		found.UpdatedAt = time.Now()
		if err := uc.wishlistRepo.Update(ctx, found); err != nil {
			return err
		}
		wishlist = found
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("toggleWishlistItem: %w", err)
	}
	return wishlist, nil
}
