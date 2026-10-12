package dashboard_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/dashboard"
)

func amsterdam(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)
	return loc
}

func at(loc *time.Location, y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, loc)
}

func TestParseSalesPeriod(t *testing.T) {
	for _, s := range []string{"today", "this_week", "this_month"} {
		p, err := dashboard.ParseSalesPeriod(s)
		require.NoError(t, err)
		assert.Equal(t, dashboard.SalesPeriod(s), p)
	}
	_, err := dashboard.ParseSalesPeriod("this_year")
	assert.ErrorIs(t, err, dashboard.ErrInvalidSalesPeriod)
}

func TestSalesPeriod_TodayComparesWithAllOfYesterday(t *testing.T) {
	loc := amsterdam(t)
	now := at(loc, 2026, 10, 12, 14, 30)

	r := dashboard.SalesPeriodToday.Ranges(now, loc)

	assert.Equal(t, dashboard.DateRange{From: at(loc, 2026, 10, 12, 0, 0), To: now}, r.Current)
	assert.Equal(t, dashboard.DateRange{From: at(loc, 2026, 10, 11, 0, 0), To: at(loc, 2026, 10, 12, 0, 0)}, r.Previous)
}

func TestSalesPeriod_WeekStartsOnSunday(t *testing.T) {
	loc := amsterdam(t)
	now := at(loc, 2026, 10, 14, 9, 0) // Wednesday

	r := dashboard.SalesPeriodThisWeek.Ranges(now, loc)

	assert.Equal(t, at(loc, 2026, 10, 11, 0, 0), r.Current.From) // Sunday
	assert.Equal(t, now, r.Current.To)
	assert.Equal(t, dashboard.DateRange{From: at(loc, 2026, 10, 4, 0, 0), To: at(loc, 2026, 10, 11, 0, 0)}, r.Previous)
}

func TestSalesPeriod_OnSundayTheWeekStartsToday(t *testing.T) {
	loc := amsterdam(t)
	now := at(loc, 2026, 10, 11, 8, 0) // Sunday

	r := dashboard.SalesPeriodThisWeek.Ranges(now, loc)

	assert.Equal(t, at(loc, 2026, 10, 11, 0, 0), r.Current.From)
	assert.Equal(t, at(loc, 2026, 10, 4, 0, 0), r.Previous.From)
}

func TestSalesPeriod_MonthComparesWithTheWholePreviousMonth(t *testing.T) {
	loc := amsterdam(t)
	now := at(loc, 2026, 3, 31, 18, 0)

	r := dashboard.SalesPeriodThisMonth.Ranges(now, loc)

	assert.Equal(t, at(loc, 2026, 3, 1, 0, 0), r.Current.From)
	assert.Equal(t, dashboard.DateRange{From: at(loc, 2026, 2, 1, 0, 0), To: at(loc, 2026, 3, 1, 0, 0)}, r.Previous)
}

func TestSalesPeriod_CrossesTheYearBoundary(t *testing.T) {
	loc := amsterdam(t)

	month := dashboard.SalesPeriodThisMonth.Ranges(at(loc, 2027, 1, 5, 10, 0), loc)
	assert.Equal(t, at(loc, 2026, 12, 1, 0, 0), month.Previous.From)

	today := dashboard.SalesPeriodToday.Ranges(at(loc, 2027, 1, 1, 10, 0), loc)
	assert.Equal(t, at(loc, 2026, 12, 31, 0, 0), today.Previous.From)

	week := dashboard.SalesPeriodThisWeek.Ranges(at(loc, 2027, 1, 1, 10, 0), loc) // Friday
	assert.Equal(t, at(loc, 2026, 12, 27, 0, 0), week.Current.From)
	assert.Equal(t, at(loc, 2026, 12, 20, 0, 0), week.Previous.From)
}

func TestSalesPeriod_DaysAreLocalAcrossDaylightSavingChanges(t *testing.T) {
	loc := amsterdam(t)
	// Clocks went back at 03:00 on Sunday 25 October 2026: yesterday was 25 hours long.
	r := dashboard.SalesPeriodToday.Ranges(at(loc, 2026, 10, 26, 12, 0), loc)

	assert.Equal(t, at(loc, 2026, 10, 25, 0, 0), r.Previous.From)
	assert.Equal(t, 25*time.Hour, r.Previous.To.Sub(r.Previous.From))
}

func TestSalesPeriod_UsesTheTenantsCalendarNotUTC(t *testing.T) {
	loc := amsterdam(t)
	// 23:30 UTC on the 11th is already the 12th in Amsterdam.
	now := time.Date(2026, 10, 11, 23, 30, 0, 0, time.UTC)

	r := dashboard.SalesPeriodToday.Ranges(now, loc)

	assert.True(t, r.Current.From.Equal(at(loc, 2026, 10, 12, 0, 0)))
}

func TestChangePercent(t *testing.T) {
	p := dashboard.ChangePercent(1200, 1000)
	require.NotNil(t, p)
	assert.InDelta(t, 20.0, *p, 0.0001)

	p = dashboard.ChangePercent(500, 1000)
	require.NotNil(t, p)
	assert.InDelta(t, -50.0, *p, 0.0001)

	assert.Nil(t, dashboard.ChangePercent(500, 0), "no previous sales: no meaningful percentage")
}
