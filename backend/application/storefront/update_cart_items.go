package storefront

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
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
	db                   *gorm.DB
	cartRepo             storefrontDomain.CartRepository
	channelRepo          channelsDomain.SalesChannelRepository
	skuRepo              pmsDomain.SKURepository
	variantRepo          pmsDomain.VariantRepository
	productRepo          pmsDomain.ProductRepository
	overrideRepo         channelsDomain.SalesChannelProductOverrideRepository
	skuCurrencyPriceRepo pmsDomain.SKUCurrencyPriceRepository
	fxRepo               fxDomain.ExchangeRateRepository
	stockLevel           StockLevelGetter
	couponRepo           couponsDomain.Repository
	couponCurrencyRepo   couponsDomain.CouponCurrencyAmountRepository
}

func NewUpdateCartItemsUseCase(
	db *gorm.DB,
	cartRepo storefrontDomain.CartRepository,
	channelRepo channelsDomain.SalesChannelRepository,
	skuRepo pmsDomain.SKURepository,
	variantRepo pmsDomain.VariantRepository,
	productRepo pmsDomain.ProductRepository,
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository,
	skuCurrencyPriceRepo pmsDomain.SKUCurrencyPriceRepository,
	fxRepo fxDomain.ExchangeRateRepository,
	stockLevel StockLevelGetter,
	couponRepo couponsDomain.Repository,
	couponCurrencyRepo couponsDomain.CouponCurrencyAmountRepository,
) *UpdateCartItemsUseCase {
	return &UpdateCartItemsUseCase{
		db: db, cartRepo: cartRepo, channelRepo: channelRepo, skuRepo: skuRepo,
		variantRepo: variantRepo, productRepo: productRepo, overrideRepo: overrideRepo,
		skuCurrencyPriceRepo: skuCurrencyPriceRepo, fxRepo: fxRepo, stockLevel: stockLevel,
		couponRepo: couponRepo, couponCurrencyRepo: couponCurrencyRepo,
	}
}

type resolvedLine struct {
	line  UpdateCartItemLineInput
	entry catalogEntry
}

// Execute replaces token's cart's whole item set -- there is no
// incremental add/remove, same "rewrite wholesale" convention as
// application/promotions.UpdatePromotionUseCase. Each line's quantity must
// not exceed current stock; a request that would leave the cart over-stock
// on any line fails entirely (no partial apply) rather than silently
// clamping. Stock is checked in a separate step after the catalog lookup's
// own transaction closes -- StockLevelGetter opens its own (see its doc
// comment), and this codebase never nests tenant.WithTenant calls.
func (uc *UpdateCartItemsUseCase) Execute(ctx context.Context, token string, lines []UpdateCartItemLineInput) (*CartResult, error) {
	var cart *storefrontDomain.Cart
	var resolved []resolvedLine
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

		resolved = make([]resolvedLine, 0, len(lines))
		for _, line := range lines {
			entry, err := lookupCatalogEntry(ctx, uc.skuRepo, uc.variantRepo, uc.productRepo, uc.overrideRepo, uc.skuCurrencyPriceRepo, uc.fxRepo, *channel, line.SKUCode, line.Quantity)
			if err != nil {
				return err
			}
			resolved = append(resolved, resolvedLine{line: line, entry: *entry})
		}
		cart = found
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateCartItems: %w", err)
	}

	items := make([]storefrontDomain.CartItem, len(resolved))
	for i, r := range resolved {
		level, err := uc.stockLevel.Execute(ctx, r.entry.SKUID)
		if err != nil {
			return nil, fmt.Errorf("updateCartItems: %w", err)
		}
		if r.line.Quantity > level.Syncable {
			return nil, fmt.Errorf("updateCartItems: %w", storefrontDomain.ErrInsufficientStock)
		}
		items[i] = storefrontDomain.CartItem{
			SKUID: r.entry.SKUID, SKUCode: r.line.SKUCode, ProductName: r.entry.ProductName,
			Quantity: r.line.Quantity, PriceMinorUnits: r.entry.EffectivePriceMinorUnits(),
			CurrencyCode: r.entry.CurrencyCode, TaxClassID: r.entry.TaxClassID,
		}
	}

	cart.Items = items
	cart.UpdatedAt = time.Now()
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.cartRepo.Update(ctx, cart)
	})
	if err != nil {
		return nil, fmt.Errorf("updateCartItems: %w", err)
	}
	return &CartResult{Cart: cart, DiscountMinorUnits: resolveDisplayDiscount(ctx, uc.db, uc.couponRepo, uc.couponCurrencyRepo, uc.fxRepo, cart)}, nil
}
