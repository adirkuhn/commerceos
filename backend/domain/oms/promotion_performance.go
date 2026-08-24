package oms

import (
	"context"
	"time"
)

// SKUSetSalesSummary is revenue/units/orders for an arbitrary set of SKUs
// within a period -- the building block for the Phase 0 "Promotion
// Analytics" report's before/during comparison.
type SKUSetSalesSummary struct {
	RevenueMinorUnits int64
	UnitsSold         int64
	// OrderCount is the number of distinct orders any of the given SKUs
	// appeared in -- not comparable to a single-SKU report's order count
	// when the set has more than one SKU (one order can count once here
	// despite containing several of the given SKUs).
	OrderCount int64
}

// PromotionPerformanceRepository backs the Phase 0 "Promotion Analytics"
// report -- same status/currency filtering as SalesReportRepository.
type PromotionPerformanceRepository interface {
	SKUSetSalesBetween(ctx context.Context, skuIDs []string, from, to time.Time, currencyCode string) (SKUSetSalesSummary, error)
}
