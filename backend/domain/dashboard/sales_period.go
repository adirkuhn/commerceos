// Package dashboard holds the rules behind the admin dashboard's figures
// that are not plain aggregates: which calendar period a figure covers and
// what it is compared with.
package dashboard

import (
	"time"

	"github.com/adirkuhn/commerceos/domain/errs"
)

// ErrInvalidSalesPeriod guards ParseSalesPeriod.
var ErrInvalidSalesPeriod = errs.New("SALES_PERIOD_INVALID", "sales period must be today, this_week or this_month")

// SalesPeriod is the calendar period the dashboard's sales card shows,
// always compared with the whole period before it.
type SalesPeriod string

const (
	SalesPeriodToday     SalesPeriod = "today"
	SalesPeriodThisWeek  SalesPeriod = "this_week"
	SalesPeriodThisMonth SalesPeriod = "this_month"
)

// ParseSalesPeriod validates s as a SalesPeriod.
func ParseSalesPeriod(s string) (SalesPeriod, error) {
	switch p := SalesPeriod(s); p {
	case SalesPeriodToday, SalesPeriodThisWeek, SalesPeriodThisMonth:
		return p, nil
	}
	return "", ErrInvalidSalesPeriod
}

// DateRange is [From, To): From inclusive, To exclusive.
type DateRange struct {
	From time.Time
	To   time.Time
}

// SalesPeriodRanges is the period so far and the whole period before it.
type SalesPeriodRanges struct {
	Current  DateRange
	Previous DateRange
}

// Ranges is the period so far (its start up to now) and the whole
// previous period, on loc's calendar: days start at local midnight, weeks
// on Sunday, months on the 1st. Local dates are built with time.Date, so a
// day with a daylight-saving change is 23 or 25 hours long, as it was.
func (p SalesPeriod) Ranges(now time.Time, loc *time.Location) SalesPeriodRanges {
	now = now.In(loc)
	y, m, d := now.Date()
	var start, prevStart time.Time
	switch p {
	case SalesPeriodThisWeek:
		start = time.Date(y, m, d-int(now.Weekday()), 0, 0, 0, 0, loc)
		prevStart = time.Date(y, m, d-int(now.Weekday())-7, 0, 0, 0, 0, loc)
	case SalesPeriodThisMonth:
		start = time.Date(y, m, 1, 0, 0, 0, 0, loc)
		prevStart = time.Date(y, m-1, 1, 0, 0, 0, 0, loc)
	default:
		start = time.Date(y, m, d, 0, 0, 0, 0, loc)
		prevStart = time.Date(y, m, d-1, 0, 0, 0, 0, loc)
	}
	return SalesPeriodRanges{
		Current:  DateRange{From: start, To: now},
		Previous: DateRange{From: prevStart, To: start},
	}
}

// ChangePercent is how much current differs from previous, in percent
// (20 = 20% more). Nil when previous is 0: there is nothing to compare with.
func ChangePercent(current, previous int64) *float64 {
	if previous == 0 {
		return nil
	}
	pct := float64(current-previous) / float64(previous) * 100
	return &pct
}
