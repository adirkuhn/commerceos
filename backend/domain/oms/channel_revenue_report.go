package oms

import (
	"context"
	"time"
)

// ChannelRevenuePoint is one bucket of the per-channel revenue time series
// -- e.g. one day's revenue for one source, depending on the queried
// granularity. Pairs with ChannelRevenue (the whole-period total per
// channel) on ChannelRevenueReportRepository.
type ChannelRevenuePoint struct {
	PeriodStart       time.Time
	Source            SalesOrderSource
	RevenueMinorUnits int64
}

// ChannelRevenueReportRepository backs the Reports "Revenue by Channel"
// report -- same status/currency filtering as SalesReportRepository (see
// its doc), but scoped to an arbitrary [from, to) instead of the
// dashboard's "since" window, and additionally bucketed by period for the
// report's daily/weekly/monthly view.
type ChannelRevenueReportRepository interface {
	RevenueByChannelBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]ChannelRevenue, error)
	RevenueByChannelTrendBetween(ctx context.Context, from, to time.Time, currencyCode string, granularity SalesTrendGranularity) ([]ChannelRevenuePoint, error)
}
