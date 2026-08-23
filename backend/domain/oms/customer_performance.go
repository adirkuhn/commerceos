package oms

import (
	"context"
	"time"
)

// CustomerPerformance is one customer's aggregate figures for
// CustomerReportRepository.CustomerPerformanceBetween -- the Phase 0
// "Customer Reports" roadmap item. CustomerName is read from the linked
// customers row (the customer's current name), not a per-order snapshot.
type CustomerPerformance struct {
	CustomerID                  string
	CustomerName                string
	OrderCount                  int64
	RevenueMinorUnits           int64
	AverageOrderValueMinorUnits int64
	FirstOrderAt                time.Time
	LastOrderAt                 time.Time
}

// CustomerReportRepository backs the Customer Performance report -- same
// status/currency filtering as SalesReportRepository (see its doc). Guest
// orders (SalesOrder.CustomerID nil) are excluded entirely -- there's no
// stable identity to group them under.
type CustomerReportRepository interface {
	// CustomerPerformanceBetween returns one row per customer with at
	// least one qualifying order in [from, to), ordered by revenue
	// descending, no limit.
	CustomerPerformanceBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]CustomerPerformance, error)
}
