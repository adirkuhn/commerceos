package oms

import (
	"context"
	"time"
)

// CohortSize is the number of customers whose very first order (all-time,
// not just within the queried period) fell inside CohortMonth. Only
// cohorts whose first order landed within the queried [From, To) are ever
// returned -- a customer whose first order predates the window isn't a new
// cohort for this report.
type CohortSize struct {
	CohortMonth   time.Time
	CustomerCount int64
}

// CohortRetentionPoint is one cell of the cohort retention grid: how many
// customers from CohortMonth's acquisition cohort placed an order
// MonthsSinceCohort months later (0 = the acquisition month itself), and
// how much revenue that represents.
type CohortRetentionPoint struct {
	CohortMonth       time.Time
	MonthsSinceCohort int64
	RetainedCustomers int64
	RevenueMinorUnits int64
}

// CustomerCohortRepository backs the Phase 0 "Customer Cohorts" report.
// Both methods define a cohort by each customer's all-time first order,
// not their first order within [from, to) -- so a customer count only
// changes as later reports include more calendar time. Cohort months are
// timezone (an IANA zone, settings.TenantSettings.ReportingTimezone) is the
// zone days/weeks/months/weekdays are bucketed in -- the tenant's local
// calendar, not UTC.
type CustomerCohortRepository interface {
	CohortSizesBetween(ctx context.Context, from, to time.Time, currencyCode string, timezone string) ([]CohortSize, error)
	CohortRetentionBetween(ctx context.Context, from, to time.Time, currencyCode string, timezone string) ([]CohortRetentionPoint, error)
}
