package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type fakeCartRecommendationsGetter struct {
	out      *appreports.CartRecommendationsReport
	err      error
	gotLines []appreports.CartLine
}

func (f *fakeCartRecommendationsGetter) Execute(_ context.Context, lines []appreports.CartLine) (*appreports.CartRecommendationsReport, error) {
	f.gotLines = lines
	if f.err != nil {
		return nil, f.err
	}
	if f.out != nil {
		return f.out, nil
	}
	return &appreports.CartRecommendationsReport{}, nil
}

func seedRecommendationCatalog() (*channelsDomain.SalesChannel, *fakeChannelRepo, *fakeOverrideRepo, *fakeProductRepo, *fakeVariantRepo, *fakeSKURepo, *fakeMediaRepo) {
	channel := internalChannel("channel-1")
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": channel}}
	overrideRepo := &fakeOverrideRepo{}
	productRepo := &fakeProductRepo{byID: map[string]pmsDomain.Product{
		"product-1": {ID: "product-1", Name: "Widget", Status: pmsDomain.ProductStatusActive},
		"product-2": {ID: "product-2", Name: "Gadget", Status: pmsDomain.ProductStatusActive},
	}}
	variantRepo := &fakeVariantRepo{
		byID: map[string]pmsDomain.Variant{
			"variant-1": {ID: "variant-1", ProductID: "product-1"},
			"variant-2": {ID: "variant-2", ProductID: "product-2"},
		},
		byProduct: map[string][]pmsDomain.Variant{"product-1": {{ID: "variant-1", ProductID: "product-1"}}},
	}
	skuRepo := &fakeSKURepo{
		byCode: map[string]pmsDomain.SKU{
			"SKU-2": {ID: "sku-2", VariantID: "variant-2", SKUCode: "SKU-2", PriceMinorUnits: 1000, CurrencyCode: "EUR"},
		},
		byVariant: map[string][]pmsDomain.SKU{
			"variant-1": {{ID: "sku-1", VariantID: "variant-1", SKUCode: "SKU-1", PriceMinorUnits: 500, CurrencyCode: "EUR"}},
		},
	}
	mediaRepo := &fakeMediaRepo{}
	return &channel, channelRepo, overrideRepo, productRepo, variantRepo, skuRepo, mediaRepo
}

func TestGetProductRecommendations_seedsEngineWithTheProductsOwnFirstSKU(t *testing.T) {
	_, channelRepo, overrideRepo, productRepo, variantRepo, skuRepo, mediaRepo := seedRecommendationCatalog()
	getter := &fakeCartRecommendationsGetter{}
	uc := appstorefront.NewGetProductRecommendationsUseCase(newMockGormDB(t, 2), channelRepo, overrideRepo, productRepo, variantRepo, skuRepo, mediaRepo, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, getter)

	_, err := uc.Execute(context.Background(), "channel-1", "product-1")

	require.NoError(t, err)
	require.Equal(t, []appreports.CartLine{{SKUCode: "SKU-1", Quantity: 1}}, getter.gotLines)
}

func TestGetProductRecommendations_mapsAvailableRecommendedProducts(t *testing.T) {
	_, channelRepo, overrideRepo, productRepo, variantRepo, skuRepo, mediaRepo := seedRecommendationCatalog()
	getter := &fakeCartRecommendationsGetter{out: &appreports.CartRecommendationsReport{
		Recommendations: []appreports.CartRecommendation{{SKUCode: "SKU-2", ProductName: "Gadget", Confidence: 0.5}},
	}}
	uc := appstorefront.NewGetProductRecommendationsUseCase(newMockGormDB(t, 2), channelRepo, overrideRepo, productRepo, variantRepo, skuRepo, mediaRepo, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, getter)

	summaries, err := uc.Execute(context.Background(), "channel-1", "product-1")

	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.Equal(t, "product-2", summaries[0].ID)
	require.Equal(t, "Gadget", summaries[0].Name)
	require.Equal(t, int64(1000), *summaries[0].PriceMinorUnits)
}

type fakeFindOverrideRepo struct {
	channelsDomain.SalesChannelProductOverrideRepository
	byProduct map[string]channelsDomain.SalesChannelProductOverride
}

func (f *fakeFindOverrideRepo) FindByChannelAndProduct(_ context.Context, _, productID string) (*channelsDomain.SalesChannelProductOverride, error) {
	if o, ok := f.byProduct[productID]; ok {
		return &o, nil
	}
	return nil, nil
}

func TestGetProductRecommendations_skipsUnenrolledRecommendation(t *testing.T) {
	_, channelRepo, _, productRepo, variantRepo, skuRepo, mediaRepo := seedRecommendationCatalog()
	overrideRepo := &fakeFindOverrideRepo{byProduct: map[string]channelsDomain.SalesChannelProductOverride{
		"product-2": {ProductID: "product-2", Included: false},
	}}
	getter := &fakeCartRecommendationsGetter{out: &appreports.CartRecommendationsReport{
		Recommendations: []appreports.CartRecommendation{{SKUCode: "SKU-2", ProductName: "Gadget"}},
	}}
	uc := appstorefront.NewGetProductRecommendationsUseCase(newMockGormDB(t, 2), channelRepo, overrideRepo, productRepo, variantRepo, skuRepo, mediaRepo, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, getter)

	summaries, err := uc.Execute(context.Background(), "channel-1", "product-1")

	require.NoError(t, err)
	require.Empty(t, summaries)
}

func TestGetProductRecommendations_returnsNilWhenProductHasNoSKUYet(t *testing.T) {
	channel := internalChannel("channel-1")
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": channel}}
	variantRepo := &fakeVariantRepo{byProduct: map[string][]pmsDomain.Variant{}}
	getter := &fakeCartRecommendationsGetter{}
	uc := appstorefront.NewGetProductRecommendationsUseCase(newMockGormDB(t, 1), channelRepo, &fakeOverrideRepo{}, &fakeProductRepo{}, variantRepo, &fakeSKURepo{}, &fakeMediaRepo{}, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, getter)

	summaries, err := uc.Execute(context.Background(), "channel-1", "product-with-no-sku")

	require.NoError(t, err)
	require.Nil(t, summaries)
	require.Nil(t, getter.gotLines)
}
