package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

func TestListProducts_allModeOnlyExcludesManualOverrides(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	overrideRepo := &fakeOverrideRepo{byChannel: map[string][]channelsDomain.SalesChannelProductOverride{
		"channel-1": {{ProductID: "product-excluded", Included: false}},
	}}
	catalog := &fakeListPOSProductCatalog{}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 1), channelRepo, overrideRepo, catalog)

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
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 1), channelRepo, overrideRepo, catalog)

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
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 1), channelRepo, &fakeOverrideRepo{}, catalog)

	_, err := uc.Execute(context.Background(), "channel-1", appstorefront.ListProductsInput{Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.NotNil(t, catalog.gotIn.IncludeIDs)
	require.Empty(t, *catalog.gotIn.IncludeIDs)
}

func TestListProducts_browseCategoryFilterPassesThroughAsAndCategoryID(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": internalChannel("channel-1")}}
	catalog := &fakeListPOSProductCatalog{}
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 1), channelRepo, &fakeOverrideRepo{}, catalog)
	browseCategory := "cat-shoes"

	_, err := uc.Execute(context.Background(), "channel-1", appstorefront.ListProductsInput{CategoryID: &browseCategory, Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.NotNil(t, catalog.gotIn.AndCategoryID)
	require.Equal(t, "cat-shoes", *catalog.gotIn.AndCategoryID)
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
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 1), channelRepo, &fakeOverrideRepo{}, catalog)

	result, err := uc.Execute(context.Background(), "channel-1", appstorefront.ListProductsInput{Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, result.Products, 1)
	require.NotNil(t, result.Products[0].PriceMinorUnits)
	require.Equal(t, int64(500), *result.Products[0].PriceMinorUnits)
	require.Equal(t, int64(400), *result.Products[0].SalePriceMinorUnits)
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
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 1), channelRepo, &fakeOverrideRepo{}, catalog)

	result, err := uc.Execute(context.Background(), "channel-b2b", appstorefront.ListProductsInput{Page: 1, PerPage: 20})

	require.NoError(t, err)
	require.Equal(t, int64(800), *result.Products[0].PriceMinorUnits)
	require.Equal(t, int64(400), *result.Products[0].SalePriceMinorUnits)
}

func TestListProducts_rejectsUnknownChannel(t *testing.T) {
	uc := appstorefront.NewListProductsUseCase(newMockGormDB(t, 1), &fakeChannelRepo{}, &fakeOverrideRepo{}, &fakeListPOSProductCatalog{})

	_, err := uc.Execute(context.Background(), "missing-channel", appstorefront.ListProductsInput{})

	require.ErrorIs(t, err, storefrontDomain.ErrNoInternalChannel)
}
