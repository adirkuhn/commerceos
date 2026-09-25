package repositories

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// cohortCustomersCTE is shared by CohortSizesBetween and
// CohortRetentionBetween: it defines each customer's cohort as the
// calendar month of their all-time first order (not just their first
// order within [from, to)), then keeps only the customers whose cohort
// month falls inside the queried window -- a customer who already existed
// before the window isn't a new cohort for this report.
const cohortCustomersCTE = `
	WITH first_orders AS (
		SELECT customer_id, date_trunc('month', MIN(occurred_at)) AS cohort_month
		FROM sales_orders
		WHERE customer_id IS NOT NULL AND currency_code = ?
			AND status IN ?
		GROUP BY customer_id
	),
	cohort_customers AS (
		SELECT customer_id, cohort_month FROM first_orders
		WHERE cohort_month >= ? AND cohort_month < ?
	)
`

type cohortSizeRow struct {
	CohortMonth   time.Time
	CustomerCount int64
}

// CohortSizesBetween returns the number of customers newly acquired in
// each calendar month within [from, to) -- see cohortCustomersCTE's doc
// for what "newly acquired" means.
func (r *SalesOrderRepo) CohortSizesBetween(ctx context.Context, from, to time.Time, currencyCode string, timezone string) ([]omsDomain.CohortSize, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if err := useReportTimezone(tx, timezone); err != nil {
		return nil, fmt.Errorf("salesOrderRepo.CohortSizesBetween: %w", err)
	}
	var rows []cohortSizeRow
	err = tx.Raw(cohortCustomersCTE+`
		SELECT cohort_month, COUNT(DISTINCT customer_id) AS customer_count
		FROM cohort_customers
		GROUP BY cohort_month
		ORDER BY cohort_month ASC
	`, currencyCode, revenueCountingStatuses, from, to).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.CohortSizesBetween: %w", err)
	}
	out := make([]omsDomain.CohortSize, len(rows))
	for i, row := range rows {
		out[i] = omsDomain.CohortSize{CohortMonth: row.CohortMonth, CustomerCount: row.CustomerCount}
	}
	return out, nil
}

type cohortRetentionRow struct {
	CohortMonth       time.Time
	MonthsSinceCohort int64
	RetainedCustomers int64
	RevenueMinorUnits int64
}

// CohortRetentionBetween returns one row per (cohort month, months since
// acquisition) for every order placed by a cohort customer up to `to` --
// including orders after the acquisition month itself, which is how a
// cohort's retention across later months is measured. months_since_cohort
// 0 is the acquisition month.
func (r *SalesOrderRepo) CohortRetentionBetween(ctx context.Context, from, to time.Time, currencyCode string, timezone string) ([]omsDomain.CohortRetentionPoint, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if err := useReportTimezone(tx, timezone); err != nil {
		return nil, fmt.Errorf("salesOrderRepo.CohortRetentionBetween: %w", err)
	}
	var rows []cohortRetentionRow
	err = tx.Raw(cohortCustomersCTE+`
		SELECT
			cc.cohort_month AS cohort_month,
			(DATE_PART('year', date_trunc('month', so.occurred_at)) - DATE_PART('year', cc.cohort_month)) * 12
				+ (DATE_PART('month', date_trunc('month', so.occurred_at)) - DATE_PART('month', cc.cohort_month)) AS months_since_cohort,
			COUNT(DISTINCT so.customer_id) AS retained_customers,
			COALESCE(SUM(`+netOrderRevenueSQL("so")+`), 0) AS revenue_minor_units
		FROM cohort_customers cc
		JOIN sales_orders so ON so.customer_id = cc.customer_id
		WHERE so.occurred_at < ? AND so.currency_code = ?
			AND so.status IN ?
		GROUP BY cc.cohort_month, months_since_cohort
		ORDER BY cc.cohort_month ASC, months_since_cohort ASC
	`, currencyCode, revenueCountingStatuses, from, to, to, currencyCode, revenueCountingStatuses).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.CohortRetentionBetween: %w", err)
	}
	out := make([]omsDomain.CohortRetentionPoint, len(rows))
	for i, row := range rows {
		out[i] = omsDomain.CohortRetentionPoint{
			CohortMonth: row.CohortMonth, MonthsSinceCohort: row.MonthsSinceCohort,
			RetainedCustomers: row.RetainedCustomers, RevenueMinorUnits: row.RevenueMinorUnits,
		}
	}
	return out, nil
}

type skuSetSalesRow struct {
	RevenueMinorUnits int64
	UnitsSold         int64
	OrderCount        int64
}

// SKUSetSalesBetween aggregates revenue/units/distinct-order-count for an
// arbitrary set of SKUs within [from, to) -- the Phase 0 "Promotion
// Analytics" report's before/during building block. An empty skuIDs
// returns a zero-value summary without querying (an empty SQL IN () is
// invalid).
func (r *SalesOrderRepo) SKUSetSalesBetween(ctx context.Context, skuIDs []string, from, to time.Time, currencyCode string) (omsDomain.SKUSetSalesSummary, error) {
	if len(skuIDs) == 0 {
		return omsDomain.SKUSetSalesSummary{}, nil
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return omsDomain.SKUSetSalesSummary{}, err
	}
	var row skuSetSalesRow
	err = tx.Raw(`
		SELECT
			COALESCE(SUM(`+netLineRevenueSQL+`), 0) AS revenue_minor_units,
			COALESCE(SUM(`+netLineQuantitySQL+`), 0) AS units_sold,
			COUNT(DISTINCT li.sales_order_id) AS order_count
		FROM sales_order_line_items li
		JOIN sales_orders so ON so.id = li.sales_order_id AND `+lineNotFullyRefundedSQL+`
		WHERE li.sku_id IN ? AND so.occurred_at >= ? AND so.occurred_at < ? AND so.currency_code = ?
			AND so.status IN ?
	`, skuIDs, from, to, currencyCode, revenueCountingStatuses).Scan(&row).Error
	if err != nil {
		return omsDomain.SKUSetSalesSummary{}, fmt.Errorf("salesOrderRepo.SKUSetSalesBetween: %w", err)
	}
	return omsDomain.SKUSetSalesSummary{
		RevenueMinorUnits: row.RevenueMinorUnits, UnitsSold: row.UnitsSold, OrderCount: row.OrderCount,
	}, nil
}

type orderSKURow struct {
	OrderID string `gorm:"column:order_id"`
	SKUID   string `gorm:"column:sku_id"`
}

// ListOrderSKUsBetween returns one (order_id, sku_id) row per distinct SKU
// in each qualifying order -- deliberately flat rather than a Postgres
// array_agg, since scanning a Postgres array column into a Go []string
// isn't supported by GORM's plain database/sql Scan without an extra
// array-type dependency; grouping into omsDomain.OrderSKUs happens here in
// Go instead, order preserved via a parallel slice.
func (r *SalesOrderRepo) ListOrderSKUsBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]omsDomain.OrderSKUs, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []orderSKURow
	err = tx.Raw(`
		SELECT DISTINCT so.id AS order_id, li.sku_id AS sku_id
		FROM sales_order_line_items li
		JOIN sales_orders so ON so.id = li.sales_order_id AND `+lineNotFullyRefundedSQL+`
		WHERE li.sku_id IS NOT NULL
			AND so.occurred_at >= ? AND so.occurred_at < ? AND so.currency_code = ?
			AND so.status IN ?
		ORDER BY so.id
	`, from, to, currencyCode, revenueCountingStatuses).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.ListOrderSKUsBetween: %w", err)
	}

	var orders []omsDomain.OrderSKUs
	var current *omsDomain.OrderSKUs
	for _, row := range rows {
		if current == nil || current.OrderID != row.OrderID {
			orders = append(orders, omsDomain.OrderSKUs{OrderID: row.OrderID})
			current = &orders[len(orders)-1]
		}
		current.SKUIDs = append(current.SKUIDs, row.SKUID)
	}
	return orders, nil
}
