package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

func newGetProductUseCase(t *testing.T, n int, channelRepo *fakeChannelRepo, overrideRepo *fakeOverrideRepo, productRepo *fakeProductRepo, variantRepo *fakeVariantRepo, skuRepo *fakeSKURepo, mediaRepo *fakeMediaRepo, stockLevel *fakeStockLevel) *appstorefront.GetProductUseCase {
	return appstorefront.NewGetProductUseCase(newMockGormDB(t, n), channelRepo, overrideRepo, productRepo, variantRepo, skuRepo, mediaRepo, stockLevel)
}

func TestGetProduct_returnsFullVariantSKUAndImageDetail(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	productRepo := &fakeProductRepo{byID: map[string]pmsDomain.Product{
		"product-1": {ID: "product-1", Name: "Widget", Status: pmsDomain.ProductStatusActive},
	}}
	variantRepo := &fakeVariantRepo{byProduct: map[string][]pmsDomain.Variant{
		"product-1": {{ID: "variant-1", ProductID: "product-1", Name: "Default"}},
	}}
	skuRepo := &fakeSKURepo{byVariant: map[string][]pmsDomain.SKU{
		"variant-1": {{ID: "sku-1", VariantID: "variant-1", SKUCode: "SKU-A", PriceMinorUnits: 500, CurrencyCode: "EUR"}},
	}}
	mediaRepo := &fakeMediaRepo{byProduct: map[string][]pmsDomain.Media{"product-1": {{URL: "https://example.com/a.jpg"}}}}
	stockLevel := &fakeStockLevel{byID: map[string]int64{"sku-1": 7}}

	uc := newGetProductUseCase(t, 1, channelRepo, &fakeOverrideRepo{}, productRepo, variantRepo, skuRepo, mediaRepo, stockLevel)

	detail, err := uc.Execute(context.Background(), "channel-1", "product-1")

	require.NoError(t, err)
	require.Equal(t, "Widget", detail.Name)
	require.Equal(t, []string{"https://example.com/a.jpg"}, detail.Images)
	require.Len(t, detail.Variants, 1)
	require.Len(t, detail.Variants[0].SKUs, 1)
	require.Equal(t, "SKU-A", detail.Variants[0].SKUs[0].SKUCode)
	require.Equal(t, int64(7), detail.Variants[0].SKUs[0].StockQuantity)
}

func TestGetProduct_appliesChannelPriceAdjustment(t *testing.T) {
	discount := 2000 // 20% off
	b2bChannel := internalChannel("channel-b2b")
	b2bChannel.PriceAdjustmentBasisPoints = &discount
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-b2b": b2bChannel}}
	productRepo := &fakeProductRepo{byID: map[string]pmsDomain.Product{
		"product-1": {ID: "product-1", Name: "Widget", Status: pmsDomain.ProductStatusActive},
	}}
	variantRepo := &fakeVariantRepo{byProduct: map[string][]pmsDomain.Variant{
		"product-1": {{ID: "variant-1", ProductID: "product-1", Name: "Default"}},
	}}
	skuRepo := &fakeSKURepo{byVariant: map[string][]pmsDomain.SKU{
		"variant-1": {{ID: "sku-1", VariantID: "variant-1", SKUCode: "SKU-A", PriceMinorUnits: 1000, CurrencyCode: "EUR"}},
	}}
	uc := newGetProductUseCase(t, 1, channelRepo, &fakeOverrideRepo{}, productRepo, variantRepo, skuRepo, &fakeMediaRepo{}, &fakeStockLevel{})

	detail, err := uc.Execute(context.Background(), "channel-b2b", "product-1")

	require.NoError(t, err)
	require.Equal(t, int64(800), detail.Variants[0].SKUs[0].PriceMinorUnits)
}

func TestGetProduct_rejectsDraftProduct(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	productRepo := &fakeProductRepo{byID: map[string]pmsDomain.Product{
		"product-1": {ID: "product-1", Name: "Widget", Status: pmsDomain.ProductStatusDraft},
	}}
	uc := newGetProductUseCase(t, 1, channelRepo, &fakeOverrideRepo{}, productRepo, &fakeVariantRepo{}, &fakeSKURepo{}, &fakeMediaRepo{}, &fakeStockLevel{})

	_, err := uc.Execute(context.Background(), "channel-1", "product-1")

	require.ErrorIs(t, err, storefrontDomain.ErrProductNotAvailable)
}

func TestGetProduct_rejectsUnknownProduct(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	uc := newGetProductUseCase(t, 1, channelRepo, &fakeOverrideRepo{}, &fakeProductRepo{}, &fakeVariantRepo{}, &fakeSKURepo{}, &fakeMediaRepo{}, &fakeStockLevel{})

	_, err := uc.Execute(context.Background(), "channel-1", "unknown-product")

	require.ErrorIs(t, err, storefrontDomain.ErrProductNotAvailable)
}

func TestGetProduct_rejectsUnenrolledProductInManualMode(t *testing.T) {
	manual := internalChannel("channel-1")
	manual.EnrollmentMode = channelsDomain.EnrollmentModeManual
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": manual}}
	productRepo := &fakeProductRepo{byID: map[string]pmsDomain.Product{
		"product-1": {ID: "product-1", Name: "Widget", Status: pmsDomain.ProductStatusActive},
	}}
	uc := newGetProductUseCase(t, 1, channelRepo, &fakeOverrideRepo{}, productRepo, &fakeVariantRepo{}, &fakeSKURepo{}, &fakeMediaRepo{}, &fakeStockLevel{})

	_, err := uc.Execute(context.Background(), "channel-1", "product-1")

	require.ErrorIs(t, err, storefrontDomain.ErrProductNotAvailable)
}
