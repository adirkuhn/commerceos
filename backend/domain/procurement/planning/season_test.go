package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/procurement/planning"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestEasterSunday(t *testing.T) {
	for year, want := range map[int]time.Time{
		2024: day(2024, time.March, 31),
		2025: day(2025, time.April, 20),
		2026: day(2026, time.April, 5),
		2027: day(2027, time.March, 28),
		2038: day(2038, time.April, 25),
	} {
		assert.Equal(t, want, planning.EasterSunday(year), "%d", year)
	}
}

func TestSeasonEventAnchors(t *testing.T) {
	cases := map[planning.SeasonEventKey]map[int]time.Time{
		planning.SeasonChristmas:    {2026: day(2026, time.December, 25)},
		planning.SeasonEaster:       {2026: day(2026, time.April, 5), 2027: day(2027, time.March, 28)},
		planning.SeasonCarnival:     {2026: day(2026, time.February, 17), 2027: day(2027, time.February, 9)},
		planning.SeasonSinterklaas:  {2026: day(2026, time.December, 5)},
		planning.SeasonValentines:   {2026: day(2026, time.February, 14)},
		planning.SeasonMothersDay:   {2026: day(2026, time.May, 10), 2027: day(2027, time.May, 9)},
		planning.SeasonFathersDayBR: {2026: day(2026, time.August, 9)},
		planning.SeasonFestaJunina:  {2026: day(2026, time.June, 24)},
		planning.SeasonBlackFriday:  {2026: day(2026, time.November, 27), 2027: day(2027, time.November, 26)},
	}
	for key, years := range cases {
		event, ok := planning.FindSeasonEvent(key)
		require.True(t, ok, key)
		for year, want := range years {
			assert.Equal(t, want, event.Anchor(year), "%s %d", key, year)
		}
	}
}

func TestSeasonEventWindowFollowsTheMovingHoliday(t *testing.T) {
	easter, _ := planning.FindSeasonEvent(planning.SeasonEaster)

	w := easter.Window(2027)

	// 6 weeks before Easter Sunday up to and including Easter Monday.
	assert.Equal(t, day(2027, time.February, 14), w.Start)
	assert.Equal(t, day(2027, time.March, 30), w.End)
	assert.Equal(t, easter.Window(2026).Days(), w.Days())
}

func TestSeasonEventsAreAllListedWithUniqueKeys(t *testing.T) {
	seen := map[planning.SeasonEventKey]bool{}
	for _, e := range planning.SeasonEvents() {
		assert.False(t, seen[e.Key], e.Key)
		seen[e.Key] = true
		w := e.Window(2026)
		assert.True(t, w.End.After(w.Start), e.Key)
	}
	assert.Len(t, seen, 16)
	_, ok := planning.FindSeasonEvent("nope")
	assert.False(t, ok)
}

func TestSeasonWindowDays(t *testing.T) {
	assert.Equal(t, 31, planning.Window{Start: day(2026, time.December, 1), End: day(2027, time.January, 1)}.Days())
}

func TestPlanSeason_CoversPreSeasonAndSeasonDemand(t *testing.T) {
	r := planning.PlanSeason(planning.SeasonInput{
		AsOf:           day(2026, time.October, 6),
		Window:         planning.Window{Start: day(2026, time.November, 13), End: day(2026, time.December, 26)}, // 43 days
		PreSeasonUnits: 20, SeasonUnits: 860, SafetyDays: 5,
		OnHand: 30, OnOrder: 0, UnitsPerPurchasedItem: 6, LeadTimeDays: 21,
	})

	assert.InDelta(t, 20.0, r.DailyDemand, 0.0001) // 860 / 43
	assert.Equal(t, int64(100), r.SafetyStock)
	assert.Equal(t, int64(20+860+100-30), r.NeededUnits)
	assert.Equal(t, int64(159), r.PurchaseQuantity) // ceil(950 / 6)
	assert.Equal(t, int64(954), r.OrderUnits)
	// 10 left when the season starts, gone the same day: the order must
	// arrive by the season start.
	require.NotNil(t, r.StockoutDate)
	assert.Equal(t, day(2026, time.November, 13), *r.StockoutDate)
	require.NotNil(t, r.OrderByDate)
	assert.Equal(t, day(2026, time.October, 23), *r.OrderByDate) // Nov 13 - 21d
}

// Enough stock for the pre-season: it runs out within the season, and the
// order must still arrive before the season starts.
func TestPlanSeason_StockoutInsideSeason(t *testing.T) {
	r := planning.PlanSeason(planning.SeasonInput{
		AsOf:           day(2026, time.October, 6),
		Window:         planning.Window{Start: day(2026, time.November, 1), End: day(2026, time.November, 11)}, // 10 days
		PreSeasonUnits: 10, SeasonUnits: 100,
		OnHand: 60, UnitsPerPurchasedItem: 1, LeadTimeDays: 7,
	})
	require.NotNil(t, r.StockoutDate)
	assert.Equal(t, day(2026, time.November, 6), *r.StockoutDate) // 50 left at start, 10/day
	assert.Equal(t, day(2026, time.October, 25), *r.OrderByDate)  // bounded by start - 7
	assert.Equal(t, int64(50), r.NeededUnits)
}

// A stockout long before the season pulls the order-by date forward.
func TestPlanSeason_EarlyStockoutPullsOrderForward(t *testing.T) {
	r := planning.PlanSeason(planning.SeasonInput{
		AsOf:           day(2026, time.October, 6),
		Window:         planning.Window{Start: day(2026, time.December, 5), End: day(2026, time.December, 15)},
		PreSeasonUnits: 60, SeasonUnits: 100,
		OnHand: 10, UnitsPerPurchasedItem: 1, LeadTimeDays: 7,
	})
	// 60 over 60 pre-season days = 1/day -> 10 units last 10 days.
	assert.Equal(t, day(2026, time.October, 16), *r.StockoutDate)
	assert.Equal(t, day(2026, time.October, 9), *r.OrderByDate)
}

// Season already running: no pre-season, the window starts today.
func TestPlanSeason_SeasonAlreadyStarted(t *testing.T) {
	r := planning.PlanSeason(planning.SeasonInput{
		AsOf:        day(2026, time.December, 1),
		Window:      planning.Window{Start: day(2026, time.November, 13), End: day(2026, time.December, 11)},
		SeasonUnits: 100, OnHand: 0, UnitsPerPurchasedItem: 1, LeadTimeDays: 3,
	})
	assert.InDelta(t, 10.0, r.DailyDemand, 0.0001)
	assert.Equal(t, day(2026, time.November, 28), *r.OrderByDate) // already late
}

func TestPlanSeason_NoDemand(t *testing.T) {
	r := planning.PlanSeason(planning.SeasonInput{
		AsOf:   day(2026, time.October, 6),
		Window: planning.Window{Start: day(2026, time.November, 1), End: day(2026, time.November, 11)},
		OnHand: 5, UnitsPerPurchasedItem: 1, LeadTimeDays: 7,
	})
	assert.Equal(t, int64(0), r.PurchaseQuantity)
	assert.Nil(t, r.OrderByDate)
}

func TestSeasonalLift(t *testing.T) {
	// 300 in a 30-day season vs 56 in the 56 days before: 10x.
	lift := planning.SeasonalLift(300, 30, 56)
	require.NotNil(t, lift)
	assert.InDelta(t, 10.0, *lift, 0.0001)
	assert.True(t, planning.IsSeasonal(lift, 300))

	assert.Nil(t, planning.SeasonalLift(300, 30, 0), "nothing sold before: unbounded")
	assert.True(t, planning.IsSeasonal(nil, 300))
	assert.False(t, planning.IsSeasonal(nil, 0))

	flat := planning.SeasonalLift(30, 30, 56)
	assert.False(t, planning.IsSeasonal(flat, 30))
}
