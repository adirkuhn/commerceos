package oms

import (
	"context"
	"time"
)

// SalesSummary is one period's aggregate figures for
// SalesReportRepository.SalesSummaryBetween -- the Phase 0 "Sales & Revenue
// Report" roadmap item's first slice. AverageOrderValueMinorUnits is 0 when
// OrderCount is 0 (never divides by zero).
type SalesSummary struct {
	RevenueMinorUnits           int64
	OrderCount                  int64
	UnitsSold                   int64
	AverageOrderValueMinorUnits int64
}

// SalesReportRepository backs ad-hoc, arbitrary-date-range sales reports --
// distinct from DashboardSalesReportRepository (which is scoped to the
// fixed dashboard screen's "since X to now" queries). Same status/currency
// filtering rules as DashboardSalesReportRepository (see its doc): only
// orders whose status actually reduced stock count, and figures are scoped
// to a single currency.
type SalesReportRepository interface {
	// SalesSummaryBetween aggregates orders with occurred_at in
	// [from, to).
	SalesSummaryBetween(ctx context.Context, from, to time.Time, currencyCode string) (SalesSummary, error)
}
