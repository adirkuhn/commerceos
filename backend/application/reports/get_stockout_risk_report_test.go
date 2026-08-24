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

func TestGetStockoutRiskReport_flagsSKUsThatWillRunOutBeforeLeadTime(t *testing.T) {
	// 30-day period, sold 60 units => velocity 2/day. Stock on hand 20 =>
	// 10 days remaining. Lead time 14 days => at risk (10 < 14).
	sku := &pmsDomain.SKU{ID: "sku-a", SKUCode: "SKU-A", VariantID: "variant-1"}
	stockLevels := &stubStockLevelLister{out: &invapp.ListStockLevelsResult{
		Levels: []invapp.StockLevel{{SKUID: "sku-a", Total: 20}},
	}}
	skuID := "sku-a"
	performance := &stubProductPerformanceReporter{out: &appreports.ProductPerformanceReport{
		Rows: []appreports.ProductPerformanceRow{{ProductPerformance: omsDomain.ProductPerformance{SKUID: &skuID, QuantitySold: 60}}},
	}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{"sku-a": sku}}
	variantLookup := &stubVariantLookupForSlowMoving{byID: map[string]*pmsDomain.Variant{
		"variant-1": {ID: "variant-1", ProductID: "product-1"},
	}}
	productLookup := &stubProductLookupForSlowMoving{byID: map[string]*pmsDomain.Product{
		"product-1": {ID: "product-1", Name: "Widget"},
	}}
	uc := appreports.NewGetStockoutRiskReportUseCase(newMockGormDB(t), stockLevels, performance, skuLookup, variantLookup, productLookup)

	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 30)
	report, err := uc.Execute(context.Background(), from, to, 14)

	require.NoError(t, err)
	require.Len(t, report.Rows, 1)
	row := report.Rows[0]
	require.Equal(t, "SKU-A", row.SKUCode)
	require.Equal(t, "Widget", row.ProductName)
	require.InDelta(t, 2.0, row.DailyVelocity, 0.001)
	require.NotNil(t, row.DaysOfStockRemaining)
	require.InDelta(t, 10.0, *row.DaysOfStockRemaining, 0.001)
	require.Equal(t, int64(8), row.SuggestedReorderQuantity) // 2*14 - 20 = 8
	require.True(t, row.AtRisk)
}

func TestGetStockoutRiskReport_notAtRiskWhenStockOutlastsLeadTime(t *testing.T) {
	stockLevels := &stubStockLevelLister{out: &invapp.ListStockLevelsResult{
		Levels: []invapp.StockLevel{{SKUID: "sku-a", Total: 1000}},
	}}
	skuID := "sku-a"
	performance := &stubProductPerformanceReporter{out: &appreports.ProductPerformanceReport{
		Rows: []appreports.ProductPerformanceRow{{ProductPerformance: omsDomain.ProductPerformance{SKUID: &skuID, QuantitySold: 30}}},
	}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{"sku-a": {ID: "sku-a", SKUCode: "SKU-A"}}}
	uc := appreports.NewGetStockoutRiskReportUseCase(newMockGormDB(t), stockLevels, performance, skuLookup, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 30)
	report, err := uc.Execute(context.Background(), from, to, 14)

	require.NoError(t, err)
	require.Len(t, report.Rows, 1)
	require.False(t, report.Rows[0].AtRisk)
	require.Equal(t, int64(0), report.Rows[0].SuggestedReorderQuantity)
}

func TestGetStockoutRiskReport_noSalesGivesNilDaysRemainingAndNotAtRisk(t *testing.T) {
	stockLevels := &stubStockLevelLister{out: &invapp.ListStockLevelsResult{
		Levels: []invapp.StockLevel{{SKUID: "sku-a", Total: 5}},
	}}
	performance := &stubProductPerformanceReporter{out: &appreports.ProductPerformanceReport{}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{"sku-a": {ID: "sku-a", SKUCode: "SKU-A"}}}
	uc := appreports.NewGetStockoutRiskReportUseCase(newMockGormDB(t), stockLevels, performance, skuLookup, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 30)
	report, err := uc.Execute(context.Background(), from, to, 14)

	require.NoError(t, err)
	require.Len(t, report.Rows, 1)
	require.Nil(t, report.Rows[0].DaysOfStockRemaining)
	require.False(t, report.Rows[0].AtRisk)
	require.Equal(t, int64(0), report.Rows[0].SuggestedReorderQuantity)
}

func TestGetStockoutRiskReport_outOfStockWithOngoingDemandIsAtRisk(t *testing.T) {
	stockLevels := &stubStockLevelLister{out: &invapp.ListStockLevelsResult{
		Levels: []invapp.StockLevel{{SKUID: "sku-a", Total: 0}},
	}}
	skuID := "sku-a"
	performance := &stubProductPerformanceReporter{out: &appreports.ProductPerformanceReport{
		Rows: []appreports.ProductPerformanceRow{{ProductPerformance: omsDomain.ProductPerformance{SKUID: &skuID, QuantitySold: 30}}},
	}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{"sku-a": {ID: "sku-a", SKUCode: "SKU-A"}}}
	uc := appreports.NewGetStockoutRiskReportUseCase(newMockGormDB(t), stockLevels, performance, skuLookup, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 30)
	report, err := uc.Execute(context.Background(), from, to, 14)

	require.NoError(t, err)
	require.Len(t, report.Rows, 1)
	require.NotNil(t, report.Rows[0].DaysOfStockRemaining)
	require.InDelta(t, 0.0, *report.Rows[0].DaysOfStockRemaining, 0.001)
	require.True(t, report.Rows[0].AtRisk)
	require.Equal(t, int64(14), report.Rows[0].SuggestedReorderQuantity) // velocity 1/day * 14 - 0
}

func TestGetStockoutRiskReport_defaultsLeadTimeWhenNotPositive(t *testing.T) {
	stockLevels := &stubStockLevelLister{out: &invapp.ListStockLevelsResult{
		Levels: []invapp.StockLevel{{SKUID: "sku-a", Total: 5}},
	}}
	skuID := "sku-a"
	performance := &stubProductPerformanceReporter{out: &appreports.ProductPerformanceReport{
		Rows: []appreports.ProductPerformanceRow{{ProductPerformance: omsDomain.ProductPerformance{SKUID: &skuID, QuantitySold: 30}}},
	}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{"sku-a": {ID: "sku-a", SKUCode: "SKU-A"}}}
	uc := appreports.NewGetStockoutRiskReportUseCase(newMockGormDB(t), stockLevels, performance, skuLookup, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 30)
	// leadTimeDays <= 0 means "use the default" (14), not "no lead time".
	report, err := uc.Execute(context.Background(), from, to, 0)

	require.NoError(t, err)
	require.Len(t, report.Rows, 1)
	// velocity 1/day, stock 5 => 5 days remaining, default lead time 14 => at risk.
	require.True(t, report.Rows[0].AtRisk)
}

func TestGetStockoutRiskReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetStockoutRiskReportUseCase(
		newMockGormDB(t), &stubStockLevelLister{}, &stubProductPerformanceReporter{},
		&stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{},
	)

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour), 14)

	require.Error(t, err)
}

func TestGetStockoutRiskReport_propagatesStockLevelsError(t *testing.T) {
	uc := appreports.NewGetStockoutRiskReportUseCase(
		newMockGormDB(t), &stubStockLevelLister{err: context.DeadlineExceeded}, &stubProductPerformanceReporter{},
		&stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{},
	)

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), 14)

	require.Error(t, err)
}

func TestGetStockoutRiskReport_propagatesProductPerformanceError(t *testing.T) {
	uc := appreports.NewGetStockoutRiskReportUseCase(
		newMockGormDB(t), &stubStockLevelLister{out: &invapp.ListStockLevelsResult{}}, &stubProductPerformanceReporter{err: context.DeadlineExceeded},
		&stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{},
	)

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), 14)

	require.Error(t, err)
}
