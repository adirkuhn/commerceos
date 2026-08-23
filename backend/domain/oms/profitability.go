package oms

import (
	"context"
	"time"
)

// ProfitabilitySummary is the Phase 0 "Profitability Report" -- revenue,
// cost of goods sold, profit, and margin for [From, To). Profit/margin are
// computed only over the subset of revenue whose SKU has a known cost
// price (CostKnownRevenueMinorUnits) -- treating an unknown cost as zero
// would silently overstate profit, the same reasoning
// ProductPerformanceRow's cost/profit/margin fields being nil together
// already follows at the per-row level. RevenueMinorUnits is always the
// true total; CostKnownRevenueMinorUnits <= RevenueMinorUnits tells the
// caller how much of that total the profit figure actually covers.
type ProfitabilitySummary struct {
	RevenueMinorUnits          int64
	CostKnownRevenueMinorUnits int64
	CostMinorUnits             int64
	ProfitMinorUnits           int64
	// MarginBasisPoints is 0 when CostKnownRevenueMinorUnits is 0 (no
	// cost-priced sales at all in the period -- nothing to compute a
	// margin over).
	MarginBasisPoints int64
}

// ProfitabilityReportRepository backs the Profitability report -- same
// status/currency filtering as SalesReportRepository (see its doc).
type ProfitabilityReportRepository interface {
	ProfitabilityBetween(ctx context.Context, from, to time.Time, currencyCode string) (ProfitabilitySummary, error)
}
