package storefront_test

import (
	"context"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	"testing"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

func TestListProducts_allModeOnlyExcludesManualOverrides(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	overrideRepo := &fakeOverrideRepo{byChannel: map[string][]channelsDomain.SalesChannelProductOverride{
		"channel-1": {{ProductID: "product-excluded", Included: false}},
	}}
	catalog := &fakeListPOSProductCatalog{}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 2), channelRepo, overrideRepo, catalog, &fakeProductRepo{}, &fakeCategoryRepo{}, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, nil)

	_, err := uc.Execute(context.Background(), "channel-1", appstorefront.ListProductsInput{Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.Nil(t, catalog.gotIn.CategoryID)
	require.Nil(t, catalog.gotIn.Tag)
	require.Nil(t, catalog.gotIn.IncludeIDs)
	require.Equal(t, []string{"product-excluded"}, catalog.gotIn.ExcludeIDs)
	require.NotNil(t, catalog.gotIn.Status)
	require.Equal(t, pmsDomain.ProductStatusActive, *catalog.gotIn.Status)
}

func TestListProducts_categoryModeConstrainsToCategoryPlusManualIncludes(t *testing.T) {
	catChannel := internalChannel("channel-1")
	catChannel.EnrollmentMode = channelsDomain.EnrollmentModeCategory
	catID := "cat-1"
	catChannel.EnrollmentCategoryID = &catID
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": catChannel}}
	overrideRepo := &fakeOverrideRepo{byChannel: map[string][]channelsDomain.SalesChannelProductOverride{
		"channel-1": {{ProductID: "manually-included", Included: true}},
	}}
	catalog := &fakeListPOSProductCatalog{}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 2), channelRepo, overrideRepo, catalog, &fakeProductRepo{}, &fakeCategoryRepo{}, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, nil)

	_, err := uc.Execute(context.Background(), "channel-1", appstorefront.ListProductsInput{Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.NotNil(t, catalog.gotIn.CategoryID)
	require.Equal(t, "cat-1", *catalog.gotIn.CategoryID)
	require.NotNil(t, catalog.gotIn.IncludeIDs)
	require.Equal(t, []string{"manually-included"}, *catalog.gotIn.IncludeIDs)
}

func TestListProducts_manualModeWithNoIncludesStillQueries(t *testing.T) {
	manual := internalChannel("channel-1")
	manual.EnrollmentMode = channelsDomain.EnrollmentModeManual
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": manual}}
	catalog := &fakeListPOSProductCatalog{}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 2), channelRepo, &fakeOverrideRepo{}, catalog, &fakeProductRepo{}, &fakeCategoryRepo{}, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, nil)

	_, err := uc.Execute(context.Background(), "channel-1", appstorefront.ListProductsInput{Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.NotNil(t, catalog.gotIn.IncludeIDs)
	require.Empty(t, *catalog.gotIn.IncludeIDs)
}

func TestListProducts_browseCategoryFilterPassesThroughAsAndCategoryID(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	catalog := &fakeListPOSProductCatalog{}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 2), channelRepo, &fakeOverrideRepo{}, catalog, &fakeProductRepo{}, &fakeCategoryRepo{}, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, nil)
	browseCategory := "cat-shoes"

	_, err := uc.Execute(context.Background(), "channel-1", appstorefront.ListProductsInput{CategoryID: &browseCategory, Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.NotNil(t, catalog.gotIn.AndCategoryID)
	require.Equal(t, "cat-shoes", *catalog.gotIn.AndCategoryID)
}

func TestListProducts_manualProductIDsFilterPassesThroughAsAndIncludeIDs(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	catalog := &fakeListPOSProductCatalog{}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 2), channelRepo, &fakeOverrideRepo{}, catalog, &fakeProductRepo{}, &fakeCategoryRepo{}, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, nil)

	_, err := uc.Execute(context.Background(), "channel-1", appstorefront.ListProductsInput{ProductIDs: []string{"p1", "p2"}, Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.NotNil(t, catalog.gotIn.AndIncludeIDs)
	require.Equal(t, []string{"p1", "p2"}, *catalog.gotIn.AndIncludeIDs)
}

func TestListProducts_excludeProductIDsCombinesWithChannelOverrideExcludes(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	overrideRepo := &fakeOverrideRepo{byChannel: map[string][]channelsDomain.SalesChannelProductOverride{
		"channel-1": {{ProductID: "override-excluded", Included: false}},
	}}
	catalog := &fakeListPOSProductCatalog{}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 2), channelRepo, overrideRepo, catalog, &fakeProductRepo{}, &fakeCategoryRepo{}, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, nil)

	_, err := uc.Execute(context.Background(), "channel-1", appstorefront.ListProductsInput{ExcludeProductIDs: []string{"block-excluded"}, Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.ElementsMatch(t, []string{"override-excluded", "block-excluded"}, catalog.gotIn.ExcludeIDs)
}

func TestListProducts_nilProductIDsLeavesAndIncludeIDsUnconstrained(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	catalog := &fakeListPOSProductCatalog{}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 2), channelRepo, &fakeOverrideRepo{}, catalog, &fakeProductRepo{}, &fakeCategoryRepo{}, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, nil)

	_, err := uc.Execute(context.Background(), "channel-1", appstorefront.ListProductsInput{Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.Nil(t, catalog.gotIn.AndIncludeIDs)
}

func TestListProducts_mapsSKUsFirstSKUAsListingPrice(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	sale := int64(400)
	catalog := &fakeListPOSProductCatalog{out: &apppms.ListPOSProductCatalogResult{
		Total: 1,
		Products: []apppms.POSProductWithSKUs{{
			Product: pmsDomain.Product{ID: "product-1", Name: "Widget"},
			SKUs:    []apppms.POSProductSKU{{SKU: pmsDomain.SKU{PriceMinorUnits: 500, SalePriceMinorUnits: &sale, CurrencyCode: "EUR"}}},
		}},
	}}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 2), channelRepo, &fakeOverrideRepo{}, catalog, &fakeProductRepo{}, &fakeCategoryRepo{}, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, nil)

	result, err := uc.Execute(context.Background(), "channel-1", appstorefront.ListProductsInput{Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, result.Products, 1)
	require.NotNil(t, result.Products[0].PriceMinorUnits)
	require.Equal(t, int64(500), *result.Products[0].PriceMinorUnits)
	require.Equal(t, int64(400), *result.Products[0].SalePriceMinorUnits)
}

func TestListProducts_convertsListingPriceIntoTheChannelsCurrency(t *testing.T) {
	usdChannel := internalChannel("channel-usd")
	usdChannel.CurrencyCode = "USD"
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-usd": usdChannel}}
	sale := int64(400)
	catalog := &fakeListPOSProductCatalog{out: &apppms.ListPOSProductCatalogResult{
		Total: 1,
		Products: []apppms.POSProductWithSKUs{{
			Product: pmsDomain.Product{ID: "product-1", Name: "Widget"},
			SKUs:    []apppms.POSProductSKU{{SKU: pmsDomain.SKU{PriceMinorUnits: 1000, SalePriceMinorUnits: &sale, CurrencyCode: "EUR"}}},
		}},
	}}
	fxRepo := &fakeExchangeRateRepo{byCurrency: map[string]fxDomain.ExchangeRate{"USD": {CurrencyCode: "USD", RateBasisPoints: 10876}}}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 2), channelRepo, &fakeOverrideRepo{}, catalog, &fakeProductRepo{}, &fakeCategoryRepo{}, &fakeSKUCurrencyPriceRepo{}, fxRepo, nil)

	result, err := uc.Execute(context.Background(), "channel-usd", appstorefront.ListProductsInput{Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.Equal(t, "USD", *result.Products[0].CurrencyCode)
	require.Equal(t, int64(1088), *result.Products[0].PriceMinorUnits)    // 1000 * 1.0876, rounded
	require.Equal(t, int64(435), *result.Products[0].SalePriceMinorUnits) // 400 * 1.0876, rounded
}

func TestListProducts_usesAManualOverrideOverFxConversion(t *testing.T) {
	usdChannel := internalChannel("channel-usd")
	usdChannel.CurrencyCode = "USD"
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-usd": usdChannel}}
	catalog := &fakeListPOSProductCatalog{out: &apppms.ListPOSProductCatalogResult{
		Total: 1,
		Products: []apppms.POSProductWithSKUs{{
			Product: pmsDomain.Product{ID: "product-1", Name: "Widget"},
			SKUs:    []apppms.POSProductSKU{{SKU: pmsDomain.SKU{ID: "sku-1", PriceMinorUnits: 1000, CurrencyCode: "EUR"}}},
		}},
	}}
	overrideRepo := &fakeSKUCurrencyPriceRepo{byKey: map[string]pmsDomain.SKUCurrencyPrice{
		"sku-1|USD": {SKUID: "sku-1", CurrencyCode: "USD", PriceMinorUnits: 999},
	}}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 2), channelRepo, &fakeOverrideRepo{}, catalog, &fakeProductRepo{}, &fakeCategoryRepo{}, overrideRepo, &fakeExchangeRateRepo{}, nil)

	result, err := uc.Execute(context.Background(), "channel-usd", appstorefront.ListProductsInput{Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.Equal(t, int64(999), *result.Products[0].PriceMinorUnits)
}

func TestListProducts_appliesChannelPriceAdjustmentToListingPrice(t *testing.T) {
	discount := 2000 // 20% off
	b2bChannel := internalChannel("channel-b2b")
	b2bChannel.PriceAdjustmentBasisPoints = &discount
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-b2b": b2bChannel}}
	sale := int64(500)
	catalog := &fakeListPOSProductCatalog{out: &apppms.ListPOSProductCatalogResult{
		Total: 1,
		Products: []apppms.POSProductWithSKUs{{
			Product: pmsDomain.Product{ID: "product-1", Name: "Widget"},
			SKUs:    []apppms.POSProductSKU{{SKU: pmsDomain.SKU{PriceMinorUnits: 1000, SalePriceMinorUnits: &sale, CurrencyCode: "EUR"}}},
		}},
	}}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 2), channelRepo, &fakeOverrideRepo{}, catalog, &fakeProductRepo{}, &fakeCategoryRepo{}, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, nil)

	result, err := uc.Execute(context.Background(), "channel-b2b", appstorefront.ListProductsInput{Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.Equal(t, int64(800), *result.Products[0].PriceMinorUnits)
	require.Equal(t, int64(400), *result.Products[0].SalePriceMinorUnits)
}

func TestListProducts_returnsCategoryFacetsWithNamesAndNonZeroCountsOnly(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	productRepo := &fakeProductRepo{categoryCounts: map[string]int64{"cat-1": 3, "cat-2": 0}}
	categoryRepo := &fakeCategoryRepo{out: []pmsDomain.Category{
		{ID: "cat-1", Name: "Shoes"}, {ID: "cat-2", Name: "Empty Category"}, {ID: "cat-3", Name: "Uncounted"},
	}}
	catalog := &fakeListPOSProductCatalog{}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 2), channelRepo, &fakeOverrideRepo{}, catalog, productRepo, categoryRepo, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, nil)

	result, err := uc.Execute(context.Background(), "channel-1", appstorefront.ListProductsInput{Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.Equal(t, []appstorefront.CategoryFacet{{CategoryID: "cat-1", CategoryName: "Shoes", Count: 3}}, result.CategoryFacets)
}

func TestListProducts_rejectsUnknownChannel(t *testing.T) {
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 2), &fakeChannelRepo{}, &fakeOverrideRepo{}, &fakeListPOSProductCatalog{}, &fakeProductRepo{}, &fakeCategoryRepo{}, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, nil)

	_, err := uc.Execute(context.Background(), "missing-channel", appstorefront.ListProductsInput{})

	require.ErrorIs(t, err, storefrontDomain.ErrNoInternalChannel)
}

type stubStockLevels map[string]int64

func (s stubStockLevels) Execute(_ context.Context, ids []string) (map[string]appinventory.StockLevel, error) {
	out := map[string]appinventory.StockLevel{}
	for _, id := range ids {
		out[id] = appinventory.StockLevel{SKUID: id, Syncable: s[id]}
	}
	return out, nil
}

// A listing tile can add a single-SKU product straight to the cart and
// show whether it's in stock; several SKUs need the product page.
func TestListProducts_reportsStockAndTheSKUATileCanAdd(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	catalog := &fakeListPOSProductCatalog{out: &apppms.ListPOSProductCatalogResult{
		Total: 2,
		Products: []apppms.POSProductWithSKUs{
			{Product: pmsDomain.Product{ID: "single"}, SKUs: []apppms.POSProductSKU{{SKU: pmsDomain.SKU{ID: "s1", SKUCode: "FEIJAO-1KG", PriceMinorUnits: 500, CurrencyCode: "EUR"}}}},
			{Product: pmsDomain.Product{ID: "sizes"}, SKUs: []apppms.POSProductSKU{
				{SKU: pmsDomain.SKU{ID: "m", SKUCode: "TEE-M", PriceMinorUnits: 900, CurrencyCode: "EUR"}},
				{SKU: pmsDomain.SKU{ID: "l", SKUCode: "TEE-L", PriceMinorUnits: 900, CurrencyCode: "EUR"}},
			}},
		},
	}}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 2), channelRepo, &fakeOverrideRepo{}, catalog, &fakeProductRepo{}, &fakeCategoryRepo{}, &fakeSKUCurrencyPriceRepo{}, &fakeExchangeRateRepo{}, stubStockLevels{"s1": 4, "m": 0, "l": 2})

	result, err := uc.Execute(context.Background(), "channel-1", appstorefront.ListProductsInput{Page: 1, PerPage: 20})

	require.NoError(t, err)
	single, sizes := result.Products[0], result.Products[1]
	require.NotNil(t, single.SKUCode)
	require.Equal(t, "FEIJAO-1KG", *single.SKUCode)
	require.Equal(t, 1, single.SKUCount)
	require.True(t, single.InStock)
	require.Nil(t, sizes.SKUCode, "several SKUs: the shopper picks on the product page")
	require.Equal(t, 2, sizes.SKUCount)
	require.True(t, sizes.InStock, "in stock when any SKU is")
}
