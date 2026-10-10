package insights_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appinsights "github.com/adirkuhn/commerceos/application/insights"
	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type stubSalesSummaryReporter struct {
	out *appreports.SalesSummaryReport
	err error
}

func (s *stubSalesSummaryReporter) Execute(_ context.Context, _, _ time.Time) (*appreports.SalesSummaryReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.SalesSummaryReport{}, nil
}

type stubProfitabilityReporter struct {
	out *appreports.ProfitabilityReport
	err error
}

func (s *stubProfitabilityReporter) Execute(_ context.Context, _, _ time.Time) (*appreports.ProfitabilityReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.ProfitabilityReport{}, nil
}

type stubProductPerformanceMatrixReporter struct {
	out *appreports.ProductPerformanceMatrix
	err error
}

func (s *stubProductPerformanceMatrixReporter) Execute(_ context.Context, _, _ time.Time) (*appreports.ProductPerformanceMatrix, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.ProductPerformanceMatrix{}, nil
}

type stubSlowMovingProductsReporter struct {
	out *appreports.SlowMovingProductsReport
	err error
}

func (s *stubSlowMovingProductsReporter) Execute(_ context.Context, _, _ time.Time) (*appreports.SlowMovingProductsReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.SlowMovingProductsReport{}, nil
}

type stubInventorySalesReporter struct {
	out *appreports.InventorySalesReport
	err error
}

func (s *stubInventorySalesReporter) Execute(_ context.Context, _, _ time.Time) (*appreports.InventorySalesReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.InventorySalesReport{}, nil
}

func noInsights() (*stubSalesSummaryReporter, *stubProfitabilityReporter, *stubProductPerformanceMatrixReporter, *stubSlowMovingProductsReporter, *stubInventorySalesReporter) {
	return &stubSalesSummaryReporter{}, &stubProfitabilityReporter{}, &stubProductPerformanceMatrixReporter{}, &stubSlowMovingProductsReporter{}, &stubInventorySalesReporter{}
}

func TestGetBusinessInsights_noSignalsGivesNoInsights(t *testing.T) {
	sales, profit, matrix, slow, inv := noInsights()
	uc := appinsights.NewGetBusinessInsightsUseCase(sales, profit, matrix, slow, inv)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Empty(t, result.Insights)
}

func TestGetBusinessInsights_flagsRevenueDecline(t *testing.T) {
	sales, profit, matrix, slow, inv := noInsights()
	change := -12.5
	sales.out = &appreports.SalesSummaryReport{RevenueChangePercent: &change}
	uc := appinsights.NewGetBusinessInsightsUseCase(sales, profit, matrix, slow, inv)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Len(t, result.Insights, 1)
	require.Equal(t, "revenue_declining", result.Insights[0].Code)
	require.Equal(t, appinsights.InsightSeverityWarning, result.Insights[0].Severity)
	require.Equal(t, map[string]any{"percent": 12.5}, result.Insights[0].Params)
}

func TestGetBusinessInsights_ignoresRevenueGrowth(t *testing.T) {
	sales, profit, matrix, slow, inv := noInsights()
	change := 12.5
	sales.out = &appreports.SalesSummaryReport{RevenueChangePercent: &change}
	uc := appinsights.NewGetBusinessInsightsUseCase(sales, profit, matrix, slow, inv)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Empty(t, result.Insights)
}

func TestGetBusinessInsights_flagsSlowMovingStock(t *testing.T) {
	sales, profit, matrix, slow, inv := noInsights()
	slow.out = &appreports.SlowMovingProductsReport{Rows: []appreports.SlowMovingProductRow{
		{SKUID: "sku-1"}, {SKUID: "sku-2"},
	}}
	uc := appinsights.NewGetBusinessInsightsUseCase(sales, profit, matrix, slow, inv)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Len(t, result.Insights, 1)
	require.Equal(t, "slow_moving_stock", result.Insights[0].Code)
	require.Contains(t, result.Insights[0].Message, "2")
	require.Equal(t, map[string]any{"count": 2}, result.Insights[0].Params)
}

func TestGetBusinessInsights_flagsIncompleteCostData(t *testing.T) {
	sales, profit, matrix, slow, inv := noInsights()
	profit.out = &appreports.ProfitabilityReport{Summary: omsDomain.ProfitabilitySummary{
		RevenueMinorUnits: 1000, CostKnownRevenueMinorUnits: 400,
	}}
	uc := appinsights.NewGetBusinessInsightsUseCase(sales, profit, matrix, slow, inv)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Len(t, result.Insights, 1)
	require.Equal(t, "incomplete_cost_data", result.Insights[0].Code)
	require.Equal(t, appinsights.InsightSeverityInfo, result.Insights[0].Severity)
	require.Equal(t, map[string]any{"percent": 40.0}, result.Insights[0].Params)
}

func TestGetBusinessInsights_doesNotFlagCompleteCostData(t *testing.T) {
	sales, profit, matrix, slow, inv := noInsights()
	profit.out = &appreports.ProfitabilityReport{Summary: omsDomain.ProfitabilitySummary{
		RevenueMinorUnits: 1000, CostKnownRevenueMinorUnits: 1000,
	}}
	uc := appinsights.NewGetBusinessInsightsUseCase(sales, profit, matrix, slow, inv)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Empty(t, result.Insights)
}

func TestGetBusinessInsights_flagsHiddenGemsAndProblems(t *testing.T) {
	sales, profit, matrix, slow, inv := noInsights()
	matrix.out = &appreports.ProductPerformanceMatrix{Rows: []appreports.ProductPerformanceMatrixRow{
		{Quadrant: appreports.QuadrantHiddenGems},
		{Quadrant: appreports.QuadrantProblems},
		{Quadrant: appreports.QuadrantProblems},
	}}
	uc := appinsights.NewGetBusinessInsightsUseCase(sales, profit, matrix, slow, inv)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Len(t, result.Insights, 2)
	codes := []string{result.Insights[0].Code, result.Insights[1].Code}
	require.Contains(t, codes, "hidden_gem_products")
	require.Contains(t, codes, "problem_products")
}

func TestGetBusinessInsights_flagsRestockRisk(t *testing.T) {
	sales, profit, matrix, slow, inv := noInsights()
	high := int64(12000)
	low := int64(500)
	inv.out = &appreports.InventorySalesReport{Rows: []appreports.InventorySalesRow{
		{SKUID: "sku-hot", TurnoverBasisPoints: &high},
		{SKUID: "sku-cold", TurnoverBasisPoints: &low},
		{SKUID: "sku-no-stock", TurnoverBasisPoints: nil},
	}}
	uc := appinsights.NewGetBusinessInsightsUseCase(sales, profit, matrix, slow, inv)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Len(t, result.Insights, 1)
	require.Equal(t, "restock_risk", result.Insights[0].Code)
	require.Contains(t, result.Insights[0].Message, "1")
	require.Equal(t, map[string]any{"count": 1}, result.Insights[0].Params)
}

func TestGetBusinessInsights_rejectsToBeforeFrom(t *testing.T) {
	sales, profit, matrix, slow, inv := noInsights()
	uc := appinsights.NewGetBusinessInsightsUseCase(sales, profit, matrix, slow, inv)

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour))

	require.Error(t, err)
}

func TestGetBusinessInsights_propagatesReportError(t *testing.T) {
	sales, profit, matrix, slow, inv := noInsights()
	sales.err = context.DeadlineExceeded
	uc := appinsights.NewGetBusinessInsightsUseCase(sales, profit, matrix, slow, inv)

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.Error(t, err)
}
