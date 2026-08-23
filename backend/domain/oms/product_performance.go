package oms

import (
	"context"
	"time"
)

// ProductPerformance is one SKU's aggregate sales figures for
// ProductPerformanceRepository.ProductPerformanceBetween -- the Phase 0
// "Product Performance" report. SKUID/SKUCode/ProductName are read from the
// line item's own snapshot (same convention as ProductSales), not joined
// from pms, so a since-renamed or deleted product still reports correctly.
// SKUID is nil for an adhoc/unmatched line item.
type ProductPerformance struct {
	SKUID             *string
	SKUCode           string
	ProductName       string
	QuantitySold      int64
	RevenueMinorUnits int64
	// OrderCount is the number of distinct orders this SKU appeared in --
	// different from QuantitySold when an order buys more than one unit.
	OrderCount  int64
	FirstSaleAt time.Time
	LastSaleAt  time.Time
}

// ProductPerformanceRepository backs the Product Performance report --
// same status/currency filtering as SalesReportRepository (see its doc).
type ProductPerformanceRepository interface {
	// ProductPerformanceBetween returns one row per SKU/adhoc-line-item
	// grouping sold in [from, to), ordered by revenue descending, no limit.
	ProductPerformanceBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]ProductPerformance, error)
}
