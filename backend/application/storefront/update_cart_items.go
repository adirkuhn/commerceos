package storefront

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	"github.com/adirkuhn/commerceos/domain/events"
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
	outbox               eventrecord.Enqueuer
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
	outbox eventrecord.Enqueuer,
) *UpdateCartItemsUseCase {
	return &UpdateCartItemsUseCase{
		db: db, cartRepo: cartRepo, channelRepo: channelRepo, skuRepo: skuRepo,
		variantRepo: variantRepo, productRepo: productRepo, overrideRepo: overrideRepo,
		skuCurrencyPriceRepo: skuCurrencyPriceRepo, fxRepo: fxRepo, stockLevel: stockLevel,
		couponRepo: couponRepo, couponCurrencyRepo: couponCurrencyRepo, outbox: outbox,
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

	before := cart.Items
	cart.Items = items
	cart.UpdatedAt = time.Now()
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.cartRepo.Update(ctx, cart); err != nil {
			return err
		}
		return recordCartItemsUpdated(ctx, uc.outbox, *cart, before)
	})
	if err != nil {
		return nil, fmt.Errorf("updateCartItems: %w", err)
	}
	return &CartResult{Cart: cart, DiscountMinorUnits: resolveDisplayDiscount(ctx, uc.db, uc.couponRepo, uc.couponCurrencyRepo, uc.fxRepo, cart)}, nil
}

type cartItemEventLine struct {
	SKUID    string `json:"sku_id"`
	SKUCode  string `json:"sku_code"`
	Quantity int64  `json:"quantity"`
}

// recordCartItemsUpdated raises storefront.cart.items_updated: the new
// lines, plus the change as sku_code -> quantity before and after.
func recordCartItemsUpdated(ctx context.Context, outbox eventrecord.Enqueuer, cart storefrontDomain.Cart, before []storefrontDomain.CartItem) error {
	lines := make([]cartItemEventLine, len(cart.Items))
	for i, it := range cart.Items {
		lines[i] = cartItemEventLine{SKUID: it.SKUID, SKUCode: it.SKUCode, Quantity: it.Quantity}
	}
	return eventrecord.Record(ctx, outbox, events.CartItemsUpdated, cart.ID,
		eventrecord.Data{"sales_channel_id": cart.SalesChannelID, "line_count": len(cart.Items), "items": lines},
		eventrecord.NewChanges().Set("items", quantitiesBySKUCode(before), quantitiesBySKUCode(cart.Items)))
}

func quantitiesBySKUCode(items []storefrontDomain.CartItem) map[string]int64 {
	out := make(map[string]int64, len(items))
	for _, it := range items {
		out[it.SKUCode] += it.Quantity
	}
	return out
}
