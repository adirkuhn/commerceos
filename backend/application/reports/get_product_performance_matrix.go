package reports

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// ProductPerformanceQuadrant is one bucket of the Phase 0 "Product
// Performance Matrix" -- classifying a SKU as High/Low sales crossed with
// High/Low margin, split around the tenant's own median for each axis
// (over the queried period's rows with a known margin), not a fixed
// threshold -- a matrix is only meaningful relative to the tenant's own
// catalog.
type ProductPerformanceQuadrant string

const (
	// QuadrantWinners is high revenue + high margin -- push harder.
	QuadrantWinners ProductPerformanceQuadrant = "winners"
	// QuadrantStars is high revenue + low margin -- optimize pricing.
	QuadrantStars ProductPerformanceQuadrant = "stars"
	// QuadrantHiddenGems is low revenue + high margin -- promote.
	QuadrantHiddenGems ProductPerformanceQuadrant = "hidden_gems"
	// QuadrantProblems is low revenue + low margin -- consider discontinuing.
	QuadrantProblems ProductPerformanceQuadrant = "problems"
)

// ProductPerformanceMatrixRow is one SKU's performance row plus its
// quadrant classification.
type ProductPerformanceMatrixRow struct {
	ProductPerformanceRow
	Quadrant ProductPerformanceQuadrant
}

// ProductPerformanceMatrix is the Phase 0 "Product Performance Matrix"
// report -- built entirely on top of ProductPerformanceReport's own rows,
// classified by revenue/margin relative to the tenant's own medians for
// the period. Rows with no known margin (see ProductPerformanceRow's doc)
// are excluded -- they can't be placed on a margin axis at all.
type ProductPerformanceMatrix struct {
	CurrencyCode            string
	MedianRevenueMinorUnits int64
	MedianMarginBasisPoints int64
	Rows                    []ProductPerformanceMatrixRow
}

// productPerformanceReporter narrows GetProductPerformanceReportUseCase to
// the one method this use case needs (ISP) -- lets tests stub it directly
// instead of constructing a real use case with real repos.
type productPerformanceReporter interface {
	Execute(ctx context.Context, from, to time.Time) (*ProductPerformanceReport, error)
}

type GetProductPerformanceMatrixUseCase struct {
	reporter productPerformanceReporter
}

func NewGetProductPerformanceMatrixUseCase(reporter productPerformanceReporter) *GetProductPerformanceMatrixUseCase {
	return &GetProductPerformanceMatrixUseCase{reporter: reporter}
}

func (uc *GetProductPerformanceMatrixUseCase) Execute(ctx context.Context, from, to time.Time) (*ProductPerformanceMatrix, error) {
	perf, err := uc.reporter.Execute(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("getProductPerformanceMatrix: %w", err)
	}

	classifiable := make([]ProductPerformanceRow, 0, len(perf.Rows))
	for _, row := range perf.Rows {
		if row.MarginBasisPoints != nil {
			classifiable = append(classifiable, row)
		}
	}

	matrix := ProductPerformanceMatrix{CurrencyCode: perf.CurrencyCode}
	if len(classifiable) == 0 {
		return &matrix, nil
	}

	revenues := make([]int64, len(classifiable))
	margins := make([]int64, len(classifiable))
	for i, row := range classifiable {
		revenues[i] = row.RevenueMinorUnits
		margins[i] = *row.MarginBasisPoints
	}
	medianRevenue := median(revenues)
	medianMargin := median(margins)
	matrix.MedianRevenueMinorUnits = medianRevenue
	matrix.MedianMarginBasisPoints = medianMargin

	matrix.Rows = make([]ProductPerformanceMatrixRow, len(classifiable))
	for i, row := range classifiable {
		highRevenue := row.RevenueMinorUnits >= medianRevenue
		highMargin := *row.MarginBasisPoints >= medianMargin
		var quadrant ProductPerformanceQuadrant
		switch {
		case highRevenue && highMargin:
			quadrant = QuadrantWinners
		case highRevenue && !highMargin:
			quadrant = QuadrantStars
		case !highRevenue && highMargin:
			quadrant = QuadrantHiddenGems
		default:
			quadrant = QuadrantProblems
		}
		matrix.Rows[i] = ProductPerformanceMatrixRow{ProductPerformanceRow: row, Quadrant: quadrant}
	}
	return &matrix, nil
}

// median sorts a copy of values and returns the middle element (or the
// average of the two middle elements, integer-divided, for an even count)
// -- a threshold for High/Low classification, not a monetary output field,
// so integer-division rounding is an acceptable simplification.
func median(values []int64) int64 {
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}
