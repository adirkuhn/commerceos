package repositories

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

func (r *SalesOrderRepo) RevenueSince(ctx context.Context, since time.Time, currencyCode string) (int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return 0, err
	}
	var total int64
	err = tx.Model(&salesOrderModel{}).
		Where("occurred_at >= ? AND currency_code = ? AND status IN ?", since, currencyCode, revenueCountingStatuses).
		Select("COALESCE(SUM(total_minor_units), 0)").Scan(&total).Error
	if err != nil {
		return 0, fmt.Errorf("salesOrderRepo.RevenueSince: %w", err)
	}
	return total, nil
}

type channelRevenueRow struct {
	Source          string
	TotalMinorUnits int64
}

func (r *SalesOrderRepo) RevenueByChannelSince(ctx context.Context, since time.Time, currencyCode string) ([]omsDomain.ChannelRevenue, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []channelRevenueRow
	err = tx.Model(&salesOrderModel{}).
		Where("occurred_at >= ? AND currency_code = ? AND status IN ?", since, currencyCode, revenueCountingStatuses).
		Select("source, COALESCE(SUM(total_minor_units), 0) AS total_minor_units").
		Group("source").Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.RevenueByChannelSince: %w", err)
	}
	out := make([]omsDomain.ChannelRevenue, len(rows))
	for i, row := range rows {
		out[i] = omsDomain.ChannelRevenue{Source: omsDomain.SalesOrderSource(row.Source), TotalMinorUnits: row.TotalMinorUnits}
	}
	return out, nil
}

type salesSummaryRow struct {
	OrderCount        int64
	RevenueMinorUnits int64
}

// SalesSummaryBetween aggregates order count/revenue from sales_orders and
// units sold from sales_order_line_items in two queries (a line-item join
// would multiply-count orders with several lines against the order-level
// total) -- same status/currency filtering as RevenueSince.
func (r *SalesOrderRepo) SalesSummaryBetween(ctx context.Context, from, to time.Time, currencyCode string) (omsDomain.SalesSummary, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return omsDomain.SalesSummary{}, err
	}
	var summary salesSummaryRow
	err = tx.Model(&salesOrderModel{}).
		Where("occurred_at >= ? AND occurred_at < ? AND currency_code = ? AND status IN ?", from, to, currencyCode, revenueCountingStatuses).
		Select("COUNT(*) AS order_count, COALESCE(SUM(total_minor_units), 0) AS revenue_minor_units").
		Scan(&summary).Error
	if err != nil {
		return omsDomain.SalesSummary{}, fmt.Errorf("salesOrderRepo.SalesSummaryBetween: %w", err)
	}

	var unitsSold int64
	err = tx.Raw(`
		SELECT COALESCE(SUM(li.quantity), 0)
		FROM sales_order_line_items li
		JOIN sales_orders so ON so.id = li.sales_order_id
		WHERE so.occurred_at >= ? AND so.occurred_at < ? AND so.currency_code = ?
			-- keep in sync with revenueCountingStatuses above
			AND so.status IN ('on-hold', 'processing', 'completed')
	`, from, to, currencyCode).Scan(&unitsSold).Error
	if err != nil {
		return omsDomain.SalesSummary{}, fmt.Errorf("salesOrderRepo.SalesSummaryBetween: %w", err)
	}

	result := omsDomain.SalesSummary{
		RevenueMinorUnits: summary.RevenueMinorUnits, OrderCount: summary.OrderCount, UnitsSold: unitsSold,
	}
	if result.OrderCount > 0 {
		result.AverageOrderValueMinorUnits = result.RevenueMinorUnits / result.OrderCount
	}
	return result, nil
}

type salesTrendBucketRow struct {
	PeriodStart       time.Time
	RevenueMinorUnits int64
	OrderCount        int64
}

type salesTrendUnitsRow struct {
	PeriodStart time.Time
	UnitsSold   int64
}

// SalesTrendBetween buckets revenue/order count (from sales_orders) and
// units sold (from sales_order_line_items) by the given granularity --
// two queries for the same multiply-counting reason as SalesSummaryBetween,
// merged in Go by truncated timestamp. Granularity is validated by the
// caller (SalesTrendGranularity.Valid()) before it ever reaches this
// parameterized date_trunc call.
func (r *SalesOrderRepo) SalesTrendBetween(ctx context.Context, from, to time.Time, currencyCode string, granularity omsDomain.SalesTrendGranularity) ([]omsDomain.SalesTrendPoint, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var buckets []salesTrendBucketRow
	err = tx.Raw(`
		SELECT date_trunc(?, occurred_at) AS period_start,
			COALESCE(SUM(total_minor_units), 0) AS revenue_minor_units,
			COUNT(*) AS order_count
		FROM sales_orders
		WHERE occurred_at >= ? AND occurred_at < ? AND currency_code = ?
			-- keep in sync with revenueCountingStatuses above
			AND status IN ('on-hold', 'processing', 'completed')
		GROUP BY period_start
		ORDER BY period_start ASC
	`, string(granularity), from, to, currencyCode).Scan(&buckets).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.SalesTrendBetween: %w", err)
	}

	var unitsRows []salesTrendUnitsRow
	err = tx.Raw(`
		SELECT date_trunc(?, so.occurred_at) AS period_start,
			COALESCE(SUM(li.quantity), 0) AS units_sold
		FROM sales_order_line_items li
		JOIN sales_orders so ON so.id = li.sales_order_id
		WHERE so.occurred_at >= ? AND so.occurred_at < ? AND so.currency_code = ?
			-- keep in sync with revenueCountingStatuses above
			AND so.status IN ('on-hold', 'processing', 'completed')
		GROUP BY period_start
	`, string(granularity), from, to, currencyCode).Scan(&unitsRows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.SalesTrendBetween: %w", err)
	}
	unitsByPeriod := make(map[int64]int64, len(unitsRows))
	for _, u := range unitsRows {
		unitsByPeriod[u.PeriodStart.UTC().Unix()] = u.UnitsSold
	}

	points := make([]omsDomain.SalesTrendPoint, len(buckets))
	for i, b := range buckets {
		points[i] = omsDomain.SalesTrendPoint{
			PeriodStart: b.PeriodStart, RevenueMinorUnits: b.RevenueMinorUnits, OrderCount: b.OrderCount,
			UnitsSold: unitsByPeriod[b.PeriodStart.UTC().Unix()],
		}
	}
	return points, nil
}

type weekdaySalesRow struct {
	Weekday           int
	RevenueMinorUnits int64
	OrderCount        int64
}

// SalesByWeekdayBetween is the "seasonality" half of the Sales Trends
// report -- sums revenue/orders across every occurrence of each weekday
// within [from, to). Postgres's EXTRACT(DOW) numbers Sunday=0..Saturday=6,
// same as time.Weekday, so no remapping is needed.
func (r *SalesOrderRepo) SalesByWeekdayBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]omsDomain.WeekdaySales, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []weekdaySalesRow
	err = tx.Raw(`
		SELECT EXTRACT(DOW FROM occurred_at)::int AS weekday,
			COALESCE(SUM(total_minor_units), 0) AS revenue_minor_units,
			COUNT(*) AS order_count
		FROM sales_orders
		WHERE occurred_at >= ? AND occurred_at < ? AND currency_code = ?
			-- keep in sync with revenueCountingStatuses above
			AND status IN ('on-hold', 'processing', 'completed')
		GROUP BY weekday
		ORDER BY weekday ASC
	`, from, to, currencyCode).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.SalesByWeekdayBetween: %w", err)
	}
	result := make([]omsDomain.WeekdaySales, len(rows))
	for i, row := range rows {
		result[i] = omsDomain.WeekdaySales{
			Weekday: time.Weekday(row.Weekday), RevenueMinorUnits: row.RevenueMinorUnits, OrderCount: row.OrderCount,
		}
	}
	return result, nil
}

// RevenueByChannelBetween is RevenueByChannelSince's [from, to) counterpart
// for the Reports "Revenue by Channel" report -- same grouping, bounded
// window instead of an open-ended "since".
func (r *SalesOrderRepo) RevenueByChannelBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]omsDomain.ChannelRevenue, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []channelRevenueRow
	err = tx.Model(&salesOrderModel{}).
		Where("occurred_at >= ? AND occurred_at < ? AND currency_code = ? AND status IN ?", from, to, currencyCode, revenueCountingStatuses).
		Select("source, COALESCE(SUM(total_minor_units), 0) AS total_minor_units").
		Group("source").Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.RevenueByChannelBetween: %w", err)
	}
	out := make([]omsDomain.ChannelRevenue, len(rows))
	for i, row := range rows {
		out[i] = omsDomain.ChannelRevenue{Source: omsDomain.SalesOrderSource(row.Source), TotalMinorUnits: row.TotalMinorUnits}
	}
	return out, nil
}

type channelRevenueTrendRow struct {
	PeriodStart       time.Time
	Source            string
	RevenueMinorUnits int64
}

// RevenueByChannelTrendBetween is the daily/weekly/monthly view of the
// Reports "Revenue by Channel" report -- same date_trunc bucketing as
// SalesTrendBetween, additionally grouped by source.
func (r *SalesOrderRepo) RevenueByChannelTrendBetween(ctx context.Context, from, to time.Time, currencyCode string, granularity omsDomain.SalesTrendGranularity) ([]omsDomain.ChannelRevenuePoint, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []channelRevenueTrendRow
	err = tx.Raw(`
		SELECT date_trunc(?, occurred_at) AS period_start,
			source,
			COALESCE(SUM(total_minor_units), 0) AS revenue_minor_units
		FROM sales_orders
		WHERE occurred_at >= ? AND occurred_at < ? AND currency_code = ?
			-- keep in sync with revenueCountingStatuses above
			AND status IN ('on-hold', 'processing', 'completed')
		GROUP BY period_start, source
		ORDER BY period_start ASC
	`, string(granularity), from, to, currencyCode).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.RevenueByChannelTrendBetween: %w", err)
	}
	points := make([]omsDomain.ChannelRevenuePoint, len(rows))
	for i, row := range rows {
		points[i] = omsDomain.ChannelRevenuePoint{
			PeriodStart: row.PeriodStart, Source: omsDomain.SalesOrderSource(row.Source), RevenueMinorUnits: row.RevenueMinorUnits,
		}
	}
	return points, nil
}

// revenueCountingStatuses mirrors omsDomain.SalesOrderStatus.ReducesStock's
// set exactly -- a status that never reduced stock (pending, cancelled,
// refunded, failed) was never real revenue either. Kept as a package-level
// slice (rather than calling ReducesStock per-row) since these three
// queries filter in SQL, not in Go.
var revenueCountingStatuses = []string{
	string(omsDomain.SalesOrderStatusOnHold),
	string(omsDomain.SalesOrderStatusProcessing),
	string(omsDomain.SalesOrderStatusCompleted),
}
