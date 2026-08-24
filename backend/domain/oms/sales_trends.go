package oms

import (
	"context"
	"time"
)

// SalesTrendGranularity is the bucket size the Phase 0 "Sales Trends &
// Seasonality" report groups revenue by.
type SalesTrendGranularity string

const (
	SalesTrendGranularityDay   SalesTrendGranularity = "day"
	SalesTrendGranularityWeek  SalesTrendGranularity = "week"
	SalesTrendGranularityMonth SalesTrendGranularity = "month"
)

func (g SalesTrendGranularity) Valid() bool {
	switch g {
	case SalesTrendGranularityDay, SalesTrendGranularityWeek, SalesTrendGranularityMonth:
		return true
	}
	return false
}

// SalesTrendPoint is one bucket of the time series -- e.g. one day, one
// week, or one month, depending on the queried granularity.
type SalesTrendPoint struct {
	PeriodStart       time.Time
	RevenueMinorUnits int64
	OrderCount        int64
	UnitsSold         int64
}

// WeekdaySales is revenue/orders summed across every occurrence of one
// weekday within the queried period -- the "seasonality" half of the
// report (which days of the week tend to sell more). Weekday uses the
// same numbering as time.Weekday (Sunday = 0).
type WeekdaySales struct {
	Weekday           time.Weekday
	RevenueMinorUnits int64
	OrderCount        int64
}

// SalesTrendRepository is the Phase 0 "Sales Trends & Seasonality" report's
// port.
type SalesTrendRepository interface {
	SalesTrendBetween(ctx context.Context, from, to time.Time, currencyCode string, granularity SalesTrendGranularity) ([]SalesTrendPoint, error)
	SalesByWeekdayBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]WeekdaySales, error)
}
