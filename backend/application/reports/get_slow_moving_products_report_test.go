package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubStockLevelLister struct {
	out *invapp.ListStockLevelsResult
	err error
}

func (s *stubStockLevelLister) Execute(_ context.Context, _ invapp.ListStockLevelsInput) (*invapp.ListStockLevelsResult, error) {
	return s.out, s.err
}

type stubVariantLookupForSlowMoving struct {
	byID map[string]*pmsDomain.Variant
}

func (s *stubVariantLookupForSlowMoving) FindByID(_ context.Context, id string) (*pmsDomain.Variant, error) {
	if v, ok := s.byID[id]; ok {
		return v, nil
	}
	return nil, pmsDomain.ErrVariantNotFound
}

type stubProductLookupForSlowMoving struct {
	byID map[string]*pmsDomain.Product
}

func (s *stubProductLookupForSlowMoving) FindByID(_ context.Context, id string) (*pmsDomain.Product, error) {
	if p, ok := s.byID[id]; ok {
		return p, nil
	}
	return nil, pmsDomain.ErrProductNotFound
}

func TestGetSlowMovingProductsReport_flagsStockedSKUsWithZeroSalesInPeriod(t *testing.T) {
	stockLevels := &stubStockLevelLister{out: &invapp.ListStockLevelsResult{Levels: []invapp.StockLevel{
		{SKUID: "sku-slow", Total: 42},
		{SKUID: "sku-selling", Total: 10},
		{SKUID: "sku-out-of-stock", Total: 0},
	}}}
	performance := &stubProductPerformanceReporter{out: &appreports.ProductPerformanceReport{
		CurrencyCode: "EUR",
		Rows: []appreports.ProductPerformanceRow{
			{ProductPerformance: skuSoldRow("sku-selling", 5)},
		},
	}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{
		"sku-slow": {ID: "sku-slow", SKUCode: "SLOW-1", VariantID: "variant-1"},
	}}
	variantLookup := &stubVariantLookupForSlowMoving{byID: map[string]*pmsDomain.Variant{
		"variant-1": {ID: "variant-1", ProductID: "product-1"},
	}}
	productLookup := &stubProductLookupForSlowMoving{byID: map[string]*pmsDomain.Product{
		"product-1": {ID: "product-1", Name: "Slow Widget"},
	}}
	uc := appreports.NewGetSlowMovingProductsReportUseCase(newMockGormDB(t), stockLevels, performance, skuLookup, variantLookup, productLookup)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Len(t, result.Rows, 1)
	row := result.Rows[0]
	require.Equal(t, "sku-slow", row.SKUID)
	require.Equal(t, "SLOW-1", row.SKUCode)
	require.Equal(t, "Slow Widget", row.ProductName)
	require.Equal(t, int64(42), row.StockOnHand)
	require.Equal(t, int64(0), row.QuantitySoldInPeriod)
}

func TestGetSlowMovingProductsReport_excludesZeroStockSKUs(t *testing.T) {
	stockLevels := &stubStockLevelLister{out: &invapp.ListStockLevelsResult{Levels: []invapp.StockLevel{
		{SKUID: "sku-out-of-stock", Total: 0},
	}}}
	performance := &stubProductPerformanceReporter{out: &appreports.ProductPerformanceReport{}}
	uc := appreports.NewGetSlowMovingProductsReportUseCase(newMockGormDB(t), stockLevels, performance, &stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Empty(t, result.Rows)
}

func TestGetSlowMovingProductsReport_propagatesStockLevelError(t *testing.T) {
	uc := appreports.NewGetSlowMovingProductsReportUseCase(
		newMockGormDB(t), &stubStockLevelLister{err: context.DeadlineExceeded},
		&stubProductPerformanceReporter{out: &appreports.ProductPerformanceReport{}},
		&stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{},
	)

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.Error(t, err)
}

func TestGetSlowMovingProductsReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetSlowMovingProductsReportUseCase(
		newMockGormDB(t), &stubStockLevelLister{}, &stubProductPerformanceReporter{},
		&stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{},
	)

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour))

	require.Error(t, err)
}

func skuSoldRow(skuID string, qty int64) omsDomain.ProductPerformance {
	return omsDomain.ProductPerformance{SKUID: &skuID, QuantitySold: qty}
}
