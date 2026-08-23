package oms

import (
	"context"
	"time"
)

// CategoryPerformance is one category's aggregate sales figures for
// CategoryReportRepository.CategoryPerformanceBetween -- the Phase 0
// "Category Analysis" report. CategoryID/CategoryName are both nil/"" for
// line items whose SKU has no category, has since been deleted, or was
// never linked to a catalog SKU at all (adhoc line item) -- these all roll
// into one "Uncategorized" bucket rather than being dropped.
type CategoryPerformance struct {
	CategoryID        *string
	CategoryName      string
	QuantitySold      int64
	RevenueMinorUnits int64
	// OrderCount is the number of distinct orders this category appeared
	// in -- different from QuantitySold when an order buys more than one
	// unit, and a category can appear once per order even if it spans
	// multiple line items in that order.
	OrderCount int64
}

// CategoryReportRepository backs the Category Analysis report -- same
// status/currency filtering as SalesReportRepository (see its doc).
type CategoryReportRepository interface {
	// CategoryPerformanceBetween returns one row per category (plus one
	// "Uncategorized" row when applicable) for line items sold in
	// [from, to), ordered by revenue descending, no limit.
	CategoryPerformanceBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]CategoryPerformance, error)
}
