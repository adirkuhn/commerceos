package storefront

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// UpdateCartItemLineInput is one requested line -- SKUCode/Quantity only.
// Price/name/tax class are never taken from the caller: every line is
// re-resolved against the current catalog on every call (see
// lookupCatalogEntry), so a cart can never be primed with an arbitrary
// price.
type UpdateCartItemLineInput struct {
	SKUCode  string
	Quantity int64
}

type UpdateCartItemsUseCase struct {
	db           *gorm.DB
	cartRepo     storefrontDomain.CartRepository
	channelRepo  channelsDomain.SalesChannelRepository
	skuRepo      pmsDomain.SKURepository
	variantRepo  pmsDomain.VariantRepository
	productRepo  pmsDomain.ProductRepository
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository
	stockLevel   stockLevelGetter
}

func NewUpdateCartItemsUseCase(
	db *gorm.DB,
	cartRepo storefrontDomain.CartRepository,
	channelRepo channelsDomain.SalesChannelRepository,
	skuRepo pmsDomain.SKURepository,
	variantRepo pmsDomain.VariantRepository,
	productRepo pmsDomain.ProductRepository,
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository,
	stockLevel stockLevelGetter,
) *UpdateCartItemsUseCase {
	return &UpdateCartItemsUseCase{
		db: db, cartRepo: cartRepo, channelRepo: channelRepo, skuRepo: skuRepo,
		variantRepo: variantRepo, productRepo: productRepo, overrideRepo: overrideRepo, stockLevel: stockLevel,
	}
}

// Execute replaces token's cart's whole item set -- there is no
// incremental add/remove, same "rewrite wholesale" convention as
// application/promotions.UpdatePromotionUseCase. Each line's quantity must
// not exceed current stock; a request that would leave the cart over-stock
// on any line fails entirely (no partial apply) rather than silently
// clamping.
func (uc *UpdateCartItemsUseCase) Execute(ctx context.Context, token string, lines []UpdateCartItemLineInput) (*storefrontDomain.Cart, error) {
	var cart *storefrontDomain.Cart
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		found, err := uc.cartRepo.FindByToken(ctx, token)
		if err != nil {
			return err
		}
		if found == nil {
			return storefrontDomain.ErrCartNotFound
		}
		if found.IsCheckedOut() {
			return storefrontDomain.ErrCartAlreadyCheckedOut
		}
		channel, err := findInternalChannel(ctx, uc.channelRepo, found.SalesChannelID)
		if err != nil {
			return err
		}

		items := make([]storefrontDomain.CartItem, 0, len(lines))
		for _, line := range lines {
			entry, err := lookupCatalogEntry(ctx, uc.skuRepo, uc.variantRepo, uc.productRepo, uc.overrideRepo, uc.stockLevel, *channel, line.SKUCode)
			if err != nil {
				return err
			}
			if line.Quantity > entry.StockQuantity {
				return storefrontDomain.ErrInsufficientStock
			}
			items = append(items, storefrontDomain.CartItem{
				SKUID: entry.SKUID, SKUCode: line.SKUCode, ProductName: entry.ProductName,
				Quantity: line.Quantity, PriceMinorUnits: entry.EffectivePriceMinorUnits(),
				CurrencyCode: entry.CurrencyCode, TaxClassID: entry.TaxClassID,
			})
		}

		found.Items = items
		found.UpdatedAt = time.Now()
		if err := uc.cartRepo.Update(ctx, found); err != nil {
			return err
		}
		cart = found
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateCartItems: %w", err)
	}
	return cart, nil
}
