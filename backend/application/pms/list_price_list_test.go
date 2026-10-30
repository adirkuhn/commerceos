package pms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

type stubPriceListCatalog struct {
	out *apppms.ListPOSProductCatalogResult
	err error
	got apppms.ListProductsInput
}

func (s *stubPriceListCatalog) Execute(_ context.Context, in apppms.ListProductsInput) (*apppms.ListPOSProductCatalogResult, error) {
	s.got = in
	return s.out, s.err
}

type stubPriceListAvailability struct {
	out    map[string]invapp.SKUAvailability
	err    error
	gotIDs []string
}

func (s *stubPriceListAvailability) Execute(_ context.Context, skuIDs []string) (map[string]invapp.SKUAvailability, error) {
	s.gotIDs = skuIDs
	return s.out, s.err
}

type stubPriceListCategories struct{ out []pmsDomain.Category }

func (s *stubPriceListCategories) List(_ context.Context) ([]pmsDomain.Category, error) {
	return s.out, nil
}

type stubPriceListTaxRates struct {
	byClass map[string]int64
	calls   int
}

func (s *stubPriceListTaxRates) Rate(_ context.Context, taxClassID *string, _ string) (int64, bool, error) {
	s.calls++
	key := ""
	if taxClassID != nil {
		key = *taxClassID
	}
	rate, configured := s.byClass[key]
	return rate, configured, nil
}

type priceListFixture struct {
	catalog      *stubPriceListCatalog
	availability *stubPriceListAvailability
	settings     *stubSettingsRepoForPMS
	categories   *stubPriceListCategories
	taxRates     *stubPriceListTaxRates
}

func newPriceListFixture() *priceListFixture {
	return &priceListFixture{
		catalog:      &stubPriceListCatalog{out: &apppms.ListPOSProductCatalogResult{}},
		availability: &stubPriceListAvailability{},
		settings:     &stubSettingsRepoForPMS{settings: &settingsDomain.TenantSettings{TenantID: "t-1"}},
		categories:   &stubPriceListCategories{},
		taxRates:     &stubPriceListTaxRates{byClass: map[string]int64{}},
	}
}

func (f *priceListFixture) useCase(t *testing.T) *apppms.ListPriceListUseCase {
	return apppms.NewListPriceListUseCase(newMockGormDB(t), f.catalog, f.availability, f.settings, f.categories, f.taxRates)
}

func TestListPriceList_oneRowPerActiveSKUWithStockCostTaxAndTarget(t *testing.T) {
	f := newPriceListFixture()
	categoryID, reduced := "cat-1", "reduced"
	markup := 4000
	sale := int64(899)
	ended := time.Now().AddDate(0, 0, -1)
	cost := int64(500)
	expiry := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	f.catalog.out = &apppms.ListPOSProductCatalogResult{Total: 7, Products: []apppms.POSProductWithSKUs{
		{Product: pmsDomain.Product{ID: "p-1", Name: "Feijão", CategoryID: &categoryID}, SKUs: []apppms.POSProductSKU{
			{VariantName: "1kg", SKU: pmsDomain.SKU{ID: "sku-1", SKUCode: "FEI-1", PriceMinorUnits: 999, SalePriceMinorUnits: &sale,
				CurrencyCode: "EUR", CostPriceMinorUnits: &cost, TaxClassID: &reduced, Status: pmsDomain.SKUStatusActive, UnitType: pmsDomain.SKUUnitTypeUnit}},
			{VariantName: "old", SKU: pmsDomain.SKU{ID: "sku-off", Status: pmsDomain.SKUStatusInactive}},
		}},
		{Product: pmsDomain.Product{ID: "p-2", Name: "Farofa"}, SKUs: []apppms.POSProductSKU{
			{SKU: pmsDomain.SKU{ID: "sku-2", SKUCode: "FAR-1", PriceMinorUnits: 450, SalePriceMinorUnits: &sale, SaleEndDate: &ended,
				CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive}},
		}},
	}}
	f.availability.out = map[string]invapp.SKUAvailability{"sku-1": {Available: 12, EarliestExpiry: &expiry}}
	f.settings.settings.DefaultTargetMarkupBasisPoints = 2500
	f.settings.settings.PricesIncludeTax = true
	f.settings.settings.DefaultTaxCountry = "NL"
	f.categories.out = []pmsDomain.Category{{ID: categoryID, TargetMarkupBasisPoints: &markup}}
	f.taxRates.byClass = map[string]int64{"reduced": 900, "": 2100}

	got, err := f.useCase(t).Execute(context.Background(), apppms.ListPriceListInput{Page: 2, PerPage: 25, Search: "fe"})

	require.NoError(t, err)
	assert.Equal(t, int64(7), got.Total)
	assert.True(t, got.PricesIncludeTax)
	assert.Equal(t, apppms.ListProductsInput{Page: 2, PerPage: 25, Search: "fe", Status: ptrProductStatus(pmsDomain.ProductStatusActive)}, f.catalog.got)
	assert.Equal(t, []string{"sku-1", "sku-2"}, f.availability.gotIDs, "inactive SKUs are not priced")
	require.Len(t, got.Rows, 2)
	assert.Equal(t, apppms.PriceListRow{
		ProductID: "p-1", ProductName: "Feijão", VariantName: "1kg", SKUID: "sku-1", SKUCode: "FEI-1",
		UnitType: pmsDomain.SKUUnitTypeUnit, CurrencyCode: "EUR", PriceMinorUnits: 999, SalePriceMinorUnits: &sale,
		CostPriceMinorUnits: &cost, AvailableQuantity: 12, EarliestExpiry: &expiry,
		TaxRateBasisPoints: 900, TaxRateConfigured: true, TargetMarkupBasisPoints: 4000,
	}, got.Rows[0])
	assert.Nil(t, got.Rows[1].SalePriceMinorUnits, "an ended sale is not offered")
	assert.Nil(t, got.Rows[1].CostPriceMinorUnits)
	assert.Equal(t, int64(2100), got.Rows[1].TaxRateBasisPoints, "nil tax class uses the default class")
	assert.Equal(t, 2500, got.Rows[1].TargetMarkupBasisPoints, "no category falls back to the tenant default")
}

// No default VAT class (or no rate for the tax country): the row says so,
// so the order form and quote can warn instead of silently using 0%.
func TestListPriceList_unconfiguredVATIsFlagged(t *testing.T) {
	f := newPriceListFixture()
	f.catalog.out = &apppms.ListPOSProductCatalogResult{Total: 1, Products: []apppms.POSProductWithSKUs{
		{Product: pmsDomain.Product{ID: "p-1", Name: "Flocao"}, SKUs: []apppms.POSProductSKU{
			{SKU: pmsDomain.SKU{ID: "sku-1", CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive}},
		}},
	}}
	f.taxRates.byClass = map[string]int64{}

	got, err := f.useCase(t).Execute(context.Background(), apppms.ListPriceListInput{})

	require.NoError(t, err)
	require.Len(t, got.Rows, 1)
	assert.False(t, got.Rows[0].TaxRateConfigured)
	assert.Zero(t, got.Rows[0].TaxRateBasisPoints)
}

func TestListPriceList_productIDsRestrictTheCatalog(t *testing.T) {
	f := newPriceListFixture()

	_, err := f.useCase(t).Execute(context.Background(), apppms.ListPriceListInput{ProductIDs: []string{"p-1", "p-2"}})

	require.NoError(t, err)
	require.NotNil(t, f.catalog.got.IncludeIDs)
	assert.Equal(t, []string{"p-1", "p-2"}, *f.catalog.got.IncludeIDs)
}

func TestListPriceList_taxRateResolvedOncePerClass(t *testing.T) {
	f := newPriceListFixture()
	f.catalog.out = &apppms.ListPOSProductCatalogResult{Products: []apppms.POSProductWithSKUs{
		{Product: pmsDomain.Product{ID: "p-1"}, SKUs: []apppms.POSProductSKU{
			{SKU: pmsDomain.SKU{ID: "a", Status: pmsDomain.SKUStatusActive}},
			{SKU: pmsDomain.SKU{ID: "b", Status: pmsDomain.SKUStatusActive}},
		}},
	}}

	_, err := f.useCase(t).Execute(context.Background(), apppms.ListPriceListInput{})

	require.NoError(t, err)
	assert.Equal(t, 1, f.taxRates.calls)
}

// Never-saved settings read as zero values, but the tenant default is
// tax-inclusive prices (see settings.TenantSettings.PricesIncludeTax).
func TestListPriceList_neverSavedSettingsMeanPricesIncludeTax(t *testing.T) {
	f := newPriceListFixture()
	f.settings.settings = &settingsDomain.TenantSettings{}

	got, err := f.useCase(t).Execute(context.Background(), apppms.ListPriceListInput{})

	require.NoError(t, err)
	assert.True(t, got.PricesIncludeTax)
}

func TestListPriceList_wrapsCatalogError(t *testing.T) {
	f := newPriceListFixture()
	f.catalog.err = errors.New("boom")

	_, err := apppms.NewListPriceListUseCase(nil, f.catalog, f.availability, f.settings, f.categories, f.taxRates).
		Execute(context.Background(), apppms.ListPriceListInput{})

	assert.ErrorContains(t, err, "listPriceList")
}

func TestListPriceList_wrapsAvailabilityError(t *testing.T) {
	f := newPriceListFixture()
	f.catalog.out = &apppms.ListPOSProductCatalogResult{Products: []apppms.POSProductWithSKUs{
		{SKUs: []apppms.POSProductSKU{{SKU: pmsDomain.SKU{ID: "a", Status: pmsDomain.SKUStatusActive}}}},
	}}
	f.availability.err = errors.New("boom")

	_, err := apppms.NewListPriceListUseCase(nil, f.catalog, f.availability, f.settings, f.categories, f.taxRates).
		Execute(context.Background(), apppms.ListPriceListInput{})

	assert.ErrorContains(t, err, "listPriceList")
}

func ptrProductStatus(s pmsDomain.ProductStatus) *pmsDomain.ProductStatus { return &s }
