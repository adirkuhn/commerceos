package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

func newUpdateCartItemsUseCase(t *testing.T, n int, cartRepo *fakeCartRepo, channelRepo *fakeChannelRepo, skuRepo *fakeSKURepo, variantRepo *fakeVariantRepo, productRepo *fakeProductRepo, overrideRepo *fakeOverrideRepo, stockLevel *fakeStockLevel) *appstorefront.UpdateCartItemsUseCase {
	return appstorefront.NewUpdateCartItemsUseCase(
		newMockGormDB(t, n), cartRepo, channelRepo, skuRepo, variantRepo, productRepo, overrideRepo,
		&fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, stockLevel, nil, &fakeCouponCurrencyAmountRepo{},
	)
}

func TestUpdateCartItems_replacesTheWholeLineSetWithFreshCatalogSnapshot(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{
		"token-1": {ID: "cart-1", Token: "token-1", SalesChannelID: "channel-1"},
	}}
	sku, variant, product, override, stockLevel := seedCatalogOneSKU("SKU-A", "sku-1", 500, 10)
	uc := newUpdateCartItemsUseCase(t, 2, cartRepo, channelRepo, sku, variant, product, override, stockLevel)

	cart, err := uc.Execute(context.Background(), "token-1", []appstorefront.UpdateCartItemLineInput{{SKUCode: "SKU-A", Quantity: 3}})

	require.NoError(t, err)
	require.Len(t, cart.Cart.Items, 1)
	require.Equal(t, "sku-1", cart.Cart.Items[0].SKUID)
	require.Equal(t, int64(500), cart.Cart.Items[0].PriceMinorUnits)
	require.Equal(t, int64(3), cart.Cart.Items[0].Quantity)
	require.Equal(t, cartRepo.updated, cart.Cart)
}

func TestUpdateCartItems_appliesChannelPriceAdjustment(t *testing.T) {
	discount := 2000 // 20% off
	b2bChannel := internalChannel("channel-b2b")
	b2bChannel.PriceAdjustmentBasisPoints = &discount
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-b2b": b2bChannel}}
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{
		"token-1": {ID: "cart-1", Token: "token-1", SalesChannelID: "channel-b2b"},
	}}
	sku, variant, product, override, stockLevel := seedCatalogOneSKU("SKU-A", "sku-1", 1000, 10)
	uc := newUpdateCartItemsUseCase(t, 2, cartRepo, channelRepo, sku, variant, product, override, stockLevel)

	cart, err := uc.Execute(context.Background(), "token-1", []appstorefront.UpdateCartItemLineInput{{SKUCode: "SKU-A", Quantity: 1}})

	require.NoError(t, err)
	require.Equal(t, int64(800), cart.Cart.Items[0].PriceMinorUnits)
}

func TestUpdateCartItems_convertsPriceIntoTheChannelsCurrencyBeforeSnapshotting(t *testing.T) {
	usdChannel := internalChannel("channel-usd")
	usdChannel.CurrencyCode = "USD"
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-usd": usdChannel}}
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{
		"token-1": {ID: "cart-1", Token: "token-1", SalesChannelID: "channel-usd"},
	}}
	sku, variant, product, override, stockLevel := seedCatalogOneSKU("SKU-A", "sku-1", 1000, 10)
	fxRepo := &fakeExchangeRateRepo{byCurrency: map[string]fxDomain.ExchangeRate{"USD": {CurrencyCode: "USD", RateBasisPoints: 10876}}}
	uc := appstorefront.NewUpdateCartItemsUseCase(
		newMockGormDB(t, 2), cartRepo, channelRepo, sku, variant, product, override,
		&fakeSKUCurrencyPriceRepo{}, fxRepo, stockLevel, nil, &fakeCouponCurrencyAmountRepo{},
	)

	cart, err := uc.Execute(context.Background(), "token-1", []appstorefront.UpdateCartItemLineInput{{SKUCode: "SKU-A", Quantity: 1}})

	require.NoError(t, err)
	require.Equal(t, "USD", cart.Cart.Items[0].CurrencyCode)
	require.Equal(t, int64(1088), cart.Cart.Items[0].PriceMinorUnits)
}

func TestUpdateCartItems_appliesQuantityBreakWhenLineQuantityQualifies(t *testing.T) {
	b2bChannel := internalChannel("channel-b2b")
	b2bChannel.QuantityBreaks = []channelsDomain.QuantityBreak{{MinQuantity: 10, DiscountBasisPoints: 3000}}
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-b2b": b2bChannel}}
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{
		"token-1": {ID: "cart-1", Token: "token-1", SalesChannelID: "channel-b2b"},
	}}
	sku, variant, product, override, stockLevel := seedCatalogOneSKU("SKU-A", "sku-1", 1000, 50)
	uc := newUpdateCartItemsUseCase(t, 2, cartRepo, channelRepo, sku, variant, product, override, stockLevel)

	cart, err := uc.Execute(context.Background(), "token-1", []appstorefront.UpdateCartItemLineInput{{SKUCode: "SKU-A", Quantity: 10}})

	require.NoError(t, err)
	require.Equal(t, int64(700), cart.Cart.Items[0].PriceMinorUnits) // 30% off, not the flat channel rate
}

func TestUpdateCartItems_usesFlatDiscountWhenQuantityBreakNotMet(t *testing.T) {
	discount := 1000 // 10% off flat
	b2bChannel := internalChannel("channel-b2b")
	b2bChannel.PriceAdjustmentBasisPoints = &discount
	b2bChannel.QuantityBreaks = []channelsDomain.QuantityBreak{{MinQuantity: 10, DiscountBasisPoints: 3000}}
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-b2b": b2bChannel}}
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{
		"token-1": {ID: "cart-1", Token: "token-1", SalesChannelID: "channel-b2b"},
	}}
	sku, variant, product, override, stockLevel := seedCatalogOneSKU("SKU-A", "sku-1", 1000, 50)
	uc := newUpdateCartItemsUseCase(t, 2, cartRepo, channelRepo, sku, variant, product, override, stockLevel)

	cart, err := uc.Execute(context.Background(), "token-1", []appstorefront.UpdateCartItemLineInput{{SKUCode: "SKU-A", Quantity: 5}})

	require.NoError(t, err)
	require.Equal(t, int64(900), cart.Cart.Items[0].PriceMinorUnits) // flat 10% off, break not met
}

func TestUpdateCartItems_usesSalePriceWhenActive(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": {ID: "cart-1", SalesChannelID: "channel-1"}}}
	sku, variant, product, override, stockLevel := seedCatalogOneSKU("SKU-A", "sku-1", 500, 10)
	sale := int64(400)
	entry := sku.byCode["SKU-A"]
	entry.SalePriceMinorUnits = &sale
	sku.byCode["SKU-A"] = entry
	uc := newUpdateCartItemsUseCase(t, 2, cartRepo, channelRepo, sku, variant, product, override, stockLevel)

	cart, err := uc.Execute(context.Background(), "token-1", []appstorefront.UpdateCartItemLineInput{{SKUCode: "SKU-A", Quantity: 1}})

	require.NoError(t, err)
	require.Equal(t, int64(400), cart.Cart.Items[0].PriceMinorUnits)
}

func TestUpdateCartItems_rejectsQuantityOverStock(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": {ID: "cart-1", SalesChannelID: "channel-1"}}}
	sku, variant, product, override, stockLevel := seedCatalogOneSKU("SKU-A", "sku-1", 500, 2)
	uc := newUpdateCartItemsUseCase(t, 1, cartRepo, channelRepo, sku, variant, product, override, stockLevel)

	_, err := uc.Execute(context.Background(), "token-1", []appstorefront.UpdateCartItemLineInput{{SKUCode: "SKU-A", Quantity: 3}})

	require.ErrorIs(t, err, storefrontDomain.ErrInsufficientStock)
}

func TestUpdateCartItems_rejectsUnenrolledSKU(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": {ID: "cart-1", SalesChannelID: "channel-1"}}}
	sku, variant, product, override, stockLevel := seedCatalogOneSKU("SKU-A", "sku-1", 500, 10)
	uc := newUpdateCartItemsUseCase(t, 1, cartRepo, channelRepo, sku, variant, product, override, stockLevel)

	_, err := uc.Execute(context.Background(), "token-1", []appstorefront.UpdateCartItemLineInput{{SKUCode: "SKU-UNKNOWN", Quantity: 1}})

	require.ErrorIs(t, err, storefrontDomain.ErrProductNotAvailable)
}

func TestUpdateCartItems_rejectsAlreadyCheckedOutCart(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	checkedOut := storefrontDomain.Cart{ID: "cart-1", SalesChannelID: "channel-1", CheckedOutAt: timePtr()}
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": checkedOut}}
	sku, variant, product, override, stockLevel := seedCatalogOneSKU("SKU-A", "sku-1", 500, 10)
	uc := newUpdateCartItemsUseCase(t, 1, cartRepo, channelRepo, sku, variant, product, override, stockLevel)

	_, err := uc.Execute(context.Background(), "token-1", []appstorefront.UpdateCartItemLineInput{{SKUCode: "SKU-A", Quantity: 1}})

	require.ErrorIs(t, err, storefrontDomain.ErrCartAlreadyCheckedOut)
}

func TestUpdateCartItems_rejectsUnknownCart(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	uc := newUpdateCartItemsUseCase(t, 1, &fakeCartRepo{}, channelRepo, &fakeSKURepo{}, &fakeVariantRepo{}, &fakeProductRepo{}, &fakeOverrideRepo{}, &fakeStockLevel{})

	_, err := uc.Execute(context.Background(), "unknown-token", nil)

	require.ErrorIs(t, err, storefrontDomain.ErrCartNotFound)
}
