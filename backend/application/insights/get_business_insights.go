package insights

import (
	"context"
	"fmt"
	"math"
	"time"

	appreports "github.com/adirkuhn/commerceos/application/reports"
)

// InsightSeverity ranks how urgently an Insight deserves attention.
type InsightSeverity string

const (
	InsightSeverityInfo    InsightSeverity = "info"
	InsightSeverityWarning InsightSeverity = "warning"
)

// Insight is one rule-based flag surfaced by the Phase 0 "Business
// Insights Dashboard" -- deliberately distinct from application/dashboard
// (a fixed "this month" snapshot screen, not a rules engine). Code is a
// stable machine-readable identifier for the rule that fired, letting the
// frontend key/style/dismiss by rule without parsing Message. Params are
// the rule's numbers (count, percent) the admin fills its translated text
// with (reports.json insights.<code>); Message is the English fallback.
type Insight struct {
	Code     string
	Severity InsightSeverity
	Params   map[string]any
	Message  string
}

// BusinessInsightsReport is every Insight that fired for [From, To).
type BusinessInsightsReport struct {
	Insights []Insight
}

// The following interfaces narrow each report's own use case to the one
// method this rule engine needs (ISP), and avoid depending on their
// concrete types -- this use case is pure composition, it runs no query
// of its own.
type salesSummaryReporter interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.SalesSummaryReport, error)
}
type profitabilityReporter interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.ProfitabilityReport, error)
}
type productPerformanceMatrixReporter interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.ProductPerformanceMatrix, error)
}
type slowMovingProductsReporter interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.SlowMovingProductsReport, error)
}
type inventorySalesReporter interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.InventorySalesReport, error)
}

type GetBusinessInsightsUseCase struct {
	salesSummary             salesSummaryReporter
	profitability            profitabilityReporter
	productPerformanceMatrix productPerformanceMatrixReporter
	slowMovingProducts       slowMovingProductsReporter
	inventorySales           inventorySalesReporter
}

func NewGetBusinessInsightsUseCase(
	salesSummary salesSummaryReporter,
	profitability profitabilityReporter,
	productPerformanceMatrix productPerformanceMatrixReporter,
	slowMovingProducts slowMovingProductsReporter,
	inventorySales inventorySalesReporter,
) *GetBusinessInsightsUseCase {
	return &GetBusinessInsightsUseCase{
		salesSummary: salesSummary, profitability: profitability, productPerformanceMatrix: productPerformanceMatrix,
		slowMovingProducts: slowMovingProducts, inventorySales: inventorySales,
	}
}

func (uc *GetBusinessInsightsUseCase) Execute(ctx context.Context, from, to time.Time) (*BusinessInsightsReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getBusinessInsights: %w", appreports.ErrToBeforeFrom)
	}

	// Each of these opens and closes its own transaction -- sequential,
	// not nested, since neither is in progress while the others run.
	sales, err := uc.salesSummary.Execute(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("getBusinessInsights: %w", err)
	}
	profit, err := uc.profitability.Execute(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("getBusinessInsights: %w", err)
	}
	matrix, err := uc.productPerformanceMatrix.Execute(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("getBusinessInsights: %w", err)
	}
	slowMoving, err := uc.slowMovingProducts.Execute(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("getBusinessInsights: %w", err)
	}
	inventorySales, err := uc.inventorySales.Execute(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("getBusinessInsights: %w", err)
	}

	var report BusinessInsightsReport
	for _, rule := range []func() *Insight{
		func() *Insight { return revenueDecliningInsight(sales) },
		func() *Insight { return slowMovingStockInsight(slowMoving) },
		func() *Insight { return incompleteCostDataInsight(profit) },
		func() *Insight { return hiddenGemProductsInsight(matrix) },
		func() *Insight { return problemProductsInsight(matrix) },
		func() *Insight { return restockRiskInsight(inventorySales) },
	} {
		if insight := rule(); insight != nil {
			report.Insights = append(report.Insights, *insight)
		}
	}
	return &report, nil
}

func revenueDecliningInsight(sales *appreports.SalesSummaryReport) *Insight {
	if sales.RevenueChangePercent == nil || *sales.RevenueChangePercent >= 0 {
		return nil
	}
	return &Insight{
		Code: "revenue_declining", Severity: InsightSeverityWarning, Params: map[string]any{"percent": oneDecimal(-*sales.RevenueChangePercent)},
		Message: fmt.Sprintf("Revenue is down %.1f%% vs the previous period.", -*sales.RevenueChangePercent),
	}
}

func slowMovingStockInsight(slowMoving *appreports.SlowMovingProductsReport) *Insight {
	n := len(slowMoving.Rows)
	if n == 0 {
		return nil
	}
	return &Insight{
		Code: "slow_moving_stock", Severity: InsightSeverityWarning, Params: map[string]any{"count": n},
		Message: fmt.Sprintf("%d product(s) have stock on hand but zero sales this period.", n),
	}
}

func incompleteCostDataInsight(profit *appreports.ProfitabilityReport) *Insight {
	s := profit.Summary
	if s.RevenueMinorUnits == 0 || s.CostKnownRevenueMinorUnits >= s.RevenueMinorUnits {
		return nil
	}
	coveragePercent := float64(s.CostKnownRevenueMinorUnits) * 100 / float64(s.RevenueMinorUnits)
	return &Insight{
		Code: "incomplete_cost_data", Severity: InsightSeverityInfo, Params: map[string]any{"percent": oneDecimal(coveragePercent)},
		Message: fmt.Sprintf("Only %.1f%% of revenue has a known cost price; profit/margin figures cover only that subset.", coveragePercent),
	}
}

func hiddenGemProductsInsight(matrix *appreports.ProductPerformanceMatrix) *Insight {
	n := countQuadrant(matrix, appreports.QuadrantHiddenGems)
	if n == 0 {
		return nil
	}
	return &Insight{
		Code: "hidden_gem_products", Severity: InsightSeverityInfo, Params: map[string]any{"count": n},
		Message: fmt.Sprintf("%d hidden-gem product(s) (high margin, low revenue) -- consider promoting.", n),
	}
}

func problemProductsInsight(matrix *appreports.ProductPerformanceMatrix) *Insight {
	n := countQuadrant(matrix, appreports.QuadrantProblems)
	if n == 0 {
		return nil
	}
	return &Insight{
		Code: "problem_products", Severity: InsightSeverityWarning, Params: map[string]any{"count": n},
		Message: fmt.Sprintf("%d problem product(s) (low revenue, low margin) -- consider discontinuing.", n),
	}
}

func countQuadrant(matrix *appreports.ProductPerformanceMatrix, quadrant appreports.ProductPerformanceQuadrant) int {
	n := 0
	for _, row := range matrix.Rows {
		if row.Quadrant == quadrant {
			n++
		}
	}
	return n
}

const restockRiskTurnoverBasisPoints = 10000

func restockRiskInsight(inventorySales *appreports.InventorySalesReport) *Insight {
	n := 0
	for _, row := range inventorySales.Rows {
		if row.TurnoverBasisPoints != nil && *row.TurnoverBasisPoints >= restockRiskTurnoverBasisPoints {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	return &Insight{
		Code: "restock_risk", Severity: InsightSeverityWarning, Params: map[string]any{"count": n},
		Message: fmt.Sprintf("%d product(s) sold through 100%% or more of their stock on hand this period -- restock risk.", n),
	}
}

// oneDecimal rounds a percentage the way the message shows it.
func oneDecimal(f float64) float64 { return math.Round(f*10) / 10 }
