package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	appreports "github.com/adirkuhn/commerceos/application/reports"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestGetInventorySalesReport_joinsStockAndSalesPerSKU(t *testing.T) {
	stockLevels := &stubStockLevelLister{out: &invapp.ListStockLevelsResult{Levels: []invapp.StockLevel{
		{SKUID: "sku-turning", Total: 40},
		{SKUID: "sku-soldout", Total: 0},
	}}}
	performance := &stubProductPerformanceReporter{out: &appreports.ProductPerformanceReport{
		Rows: []appreports.ProductPerformanceRow{
			{ProductPerformance: skuSoldRow("sku-turning", 10)},
			{ProductPerformance: skuSoldRow("sku-soldout", 6)},
		},
	}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{
		"sku-turning": {ID: "sku-turning", SKUCode: "TURN-1", VariantID: "variant-1"},
		"sku-soldout": {ID: "sku-soldout", SKUCode: "SOLD-1", VariantID: "variant-1"},
	}}
	variantLookup := &stubVariantLookupForSlowMoving{byID: map[string]*pmsDomain.Variant{
		"variant-1": {ID: "variant-1", ProductID: "product-1"},
	}}
	productLookup := &stubProductLookupForSlowMoving{byID: map[string]*pmsDomain.Product{
		"product-1": {ID: "product-1", Name: "Widget"},
	}}
	uc := appreports.NewGetInventorySalesReportUseCase(newMockGormDB(t), stockLevels, performance, skuLookup, variantLookup, productLookup)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Len(t, result.Rows, 2)

	byID := map[string]appreports.InventorySalesRow{}
	for _, row := range result.Rows {
		byID[row.SKUID] = row
	}

	turning := byID["sku-turning"]
	require.Equal(t, "TURN-1", turning.SKUCode)
	require.Equal(t, "Widget", turning.ProductName)
	require.Equal(t, int64(40), turning.StockOnHand)
	require.Equal(t, int64(10), turning.QuantitySoldInPeriod)
	require.NotNil(t, turning.TurnoverBasisPoints)
	require.Equal(t, int64(2500), *turning.TurnoverBasisPoints) // 10/40 = 25.00%

	soldOut := byID["sku-soldout"]
	require.Equal(t, int64(0), soldOut.StockOnHand)
	require.Equal(t, int64(6), soldOut.QuantitySoldInPeriod)
	require.Nil(t, soldOut.TurnoverBasisPoints) // no stock on hand -- can't compute a rate
}

func TestGetInventorySalesReport_propagatesStockLevelError(t *testing.T) {
	uc := appreports.NewGetInventorySalesReportUseCase(
		newMockGormDB(t), &stubStockLevelLister{err: context.DeadlineExceeded},
		&stubProductPerformanceReporter{out: &appreports.ProductPerformanceReport{}},
		&stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{},
	)

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.Error(t, err)
}

func TestGetInventorySalesReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetInventorySalesReportUseCase(
		newMockGormDB(t), &stubStockLevelLister{}, &stubProductPerformanceReporter{},
		&stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{},
	)

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour))

	require.Error(t, err)
}
