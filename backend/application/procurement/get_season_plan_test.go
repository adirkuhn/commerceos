package procurement_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/domain/procurement/planning"
)

func utcDate(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func newSeasonUC(t *testing.T, catalog *stubCatalogReader, demand *stubDemandReader, supply *stubSupplyReader) *appprocurement.GetSeasonPlanUseCase {
	return appprocurement.NewGetSeasonPlanUseCase(newMockGormDB(t), catalog, demand, supply, func() time.Time { return planningNow })
}

func christmas() *planning.SeasonEventKey {
	k := planning.SeasonChristmas
	return &k
}

// Today = 2026-10-06. Christmas 2026 season: Nov 13 - Dec 27 (44 days),
// last year's: Nov 13 2025 - Dec 27 2025. Pre-season: Oct 6 - Nov 13 (38 days).
var (
	xmasSeason         = window{utcDate(2026, time.November, 13), utcDate(2026, time.December, 27)}
	xmasLastYear       = window{utcDate(2025, time.November, 13), utcDate(2025, time.December, 27)}
	xmasLastYearPre    = window{utcDate(2025, time.October, 6), utcDate(2025, time.November, 13)}
	xmasLastYearBefore = window{utcDate(2025, time.September, 18), utcDate(2025, time.November, 13)}
)

func panettoneDemand() *stubDemandReader {
	return &stubDemandReader{
		sold: map[window]map[string]int64{
			recentWindow:         {"panettone": 118, "rice": 560},
			lastYearRecentWindow: {"panettone": 100, "rice": 560},
			xmasLastYear:         {"panettone": 820, "rice": 440},
			xmasLastYearPre:      {"panettone": 40, "rice": 380},
			xmasLastYearBefore:   {"panettone": 56, "rice": 560},
		},
		firstSale: map[string]time.Time{"panettone": utcDate(2023, 11, 1), "rice": utcDate(2023, 1, 1)},
	}
}

func TestGetSeasonPlan_ChristmasOrdersSeasonalProductsBeforeTheSeason(t *testing.T) {
	catalog := &stubCatalogReader{skus: []planning.PlanningSKU{
		{SKUID: "panettone", SKUCode: "PAN-1", ProductName: "Panettone", Active: true, OnHand: 60},
		{SKUID: "rice", SKUCode: "RICE", ProductName: "Rice", Active: true},
	}}
	supply := &stubSupplyReader{sourcing: map[string]planning.Sourcing{"panettone": {
		SupplierID: "sup-1", SupplierName: "Bauducco NL", LeadTimeDays: intPtr(21), UnitsPerPurchasedItem: 6,
		UnitCostMinorUnits: 450, CurrencyCode: "EUR",
	}}}

	out, err := newSeasonUC(t, catalog, panettoneDemand(), supply).Execute(context.Background(), appprocurement.SeasonPlanInput{Event: christmas(), SafetyDays: 0})

	require.NoError(t, err)
	assert.Equal(t, planning.SeasonChristmas, *out.Event)
	assert.Equal(t, 2026, out.Year)
	assert.Equal(t, xmasSeason.from, out.Window.Start)
	assert.Equal(t, xmasSeason.to, out.Window.End)
	assert.Equal(t, xmasLastYear.from, out.LastYearWindow.Start)

	// Rice sells the same all year: not part of the Christmas plan.
	require.Len(t, out.Suppliers, 1)
	require.Len(t, out.Suppliers[0].Lines, 1)
	line := out.Suppliers[0].Lines[0]
	assert.Equal(t, "panettone", line.SKUID)
	assert.Equal(t, planning.ForecastMethodSeasonal, line.Forecast.Method)
	assert.Equal(t, int64(968), line.Forecast.Units) // ceil(820 * 1.18)
	assert.Equal(t, 44, line.ForecastDays)
	assert.Equal(t, int64(820), line.LastYearUnits)
	require.NotNil(t, line.PreSeason)
	assert.Equal(t, int64(48), line.PreSeason.Units) // ceil(40 * 1.18)
	require.NotNil(t, line.SeasonalLift)
	assert.InDelta(t, (820.0/44)/(56.0/56), *line.SeasonalLift, 0.0001)
	// 48 + 968 - 60 = 956 -> 160 boxes of 6.
	assert.Equal(t, int64(956), line.Reorder.NeededUnits)
	assert.Equal(t, int64(160), line.Reorder.PurchaseQuantity)
	assert.Equal(t, utcDate(2026, time.October, 23), *line.Reorder.OrderByDate) // Nov 13 - 21d
}

func TestGetSeasonPlan_AllProductsIncludesNonSeasonal(t *testing.T) {
	catalog := &stubCatalogReader{skus: []planning.PlanningSKU{
		{SKUID: "panettone", Active: true}, {SKUID: "rice", Active: true},
	}}

	out, err := newSeasonUC(t, catalog, panettoneDemand(), &stubSupplyReader{}).Execute(context.Background(), appprocurement.SeasonPlanInput{Event: christmas(), AllProducts: true})

	require.NoError(t, err)
	require.Len(t, out.Suppliers, 1)
	assert.Len(t, out.Suppliers[0].Lines, 2)
}

// Easter 2027 is Mar 28, Easter 2026 was Apr 5: last year's season is
// aligned on Easter itself, not on the calendar dates.
func TestGetSeasonPlan_EasterComparesWithLastYearsEaster(t *testing.T) {
	easter := planning.SeasonEaster
	demand := &stubDemandReader{}

	out, err := newSeasonUC(t, &stubCatalogReader{}, demand, &stubSupplyReader{}).Execute(context.Background(), appprocurement.SeasonPlanInput{Event: &easter, Year: 2027})

	require.NoError(t, err)
	assert.Equal(t, utcDate(2027, time.February, 14), out.Window.Start)
	assert.Equal(t, utcDate(2026, time.February, 22), out.LastYearWindow.Start) // Apr 5 2026 - 42d
	assert.Contains(t, demand.asked, window{utcDate(2026, time.February, 22), utcDate(2026, time.April, 7)})
}

// Easter 2026 (Apr 5) is over by October: the next one is planned.
func TestGetSeasonPlan_DefaultsToTheNextOccurrence(t *testing.T) {
	easter := planning.SeasonEaster

	out, err := newSeasonUC(t, &stubCatalogReader{}, &stubDemandReader{}, &stubSupplyReader{}).Execute(context.Background(), appprocurement.SeasonPlanInput{Event: &easter})

	require.NoError(t, err)
	assert.Equal(t, 2027, out.Year)
}

func TestGetSeasonPlan_CustomRangeComparesWithTheSameDatesLastYear(t *testing.T) {
	custom := planning.Window{Start: utcDate(2026, time.November, 1), End: utcDate(2026, time.November, 11)}
	demand := &stubDemandReader{}

	out, err := newSeasonUC(t, &stubCatalogReader{}, demand, &stubSupplyReader{}).Execute(context.Background(), appprocurement.SeasonPlanInput{Custom: &custom})

	require.NoError(t, err)
	assert.Nil(t, out.Event)
	assert.Equal(t, custom, out.Window)
	assert.Equal(t, custom.ShiftYears(-1), out.LastYearWindow)
}

// The season already started: only what's left of it is planned, and
// compared with the same remaining part of last year's.
func TestGetSeasonPlan_RunningSeasonPlansTheRemainder(t *testing.T) {
	custom := planning.Window{Start: utcDate(2026, time.October, 1), End: utcDate(2026, time.October, 16)}
	catalog := &stubCatalogReader{skus: []planning.PlanningSKU{{SKUID: "a", Active: true}}}
	demand := &stubDemandReader{
		sold:      map[window]map[string]int64{{utcDate(2025, time.October, 6), utcDate(2025, time.October, 16)}: {"a": 100}},
		firstSale: map[string]time.Time{"a": utcDate(2024, 1, 1)},
	}

	out, err := newSeasonUC(t, catalog, demand, &stubSupplyReader{}).Execute(context.Background(), appprocurement.SeasonPlanInput{Custom: &custom})

	require.NoError(t, err)
	line := out.Suppliers[0].Lines[0]
	assert.Equal(t, 10, line.ForecastDays)
	assert.Equal(t, int64(100), line.Forecast.Units)
	assert.Equal(t, int64(0), line.PreSeason.Units)
}

func TestGetSeasonPlan_RejectsInvalidSeasons(t *testing.T) {
	unknown := planning.SeasonEventKey("halloween")
	over := planning.Window{Start: utcDate(2026, time.September, 1), End: utcDate(2026, time.October, 1)}
	backwards := planning.Window{Start: utcDate(2026, time.December, 1), End: utcDate(2026, time.November, 1)}
	tooLong := planning.Window{Start: utcDate(2026, time.November, 1), End: utcDate(2027, time.November, 3)}
	valid := planning.Window{Start: utcDate(2026, time.November, 1), End: utcDate(2026, time.November, 3)}
	cases := map[string]struct {
		in   appprocurement.SeasonPlanInput
		want error
	}{
		"neither":       {appprocurement.SeasonPlanInput{}, appprocurement.ErrInvalidSeason},
		"both":          {appprocurement.SeasonPlanInput{Event: christmas(), Custom: &valid}, appprocurement.ErrInvalidSeason},
		"unknown event": {appprocurement.SeasonPlanInput{Event: &unknown}, appprocurement.ErrUnknownSeasonEvent},
		"past year":     {appprocurement.SeasonPlanInput{Event: christmas(), Year: 2025}, appprocurement.ErrSeasonOver},
		"far year":      {appprocurement.SeasonPlanInput{Event: christmas(), Year: 2028}, appprocurement.ErrInvalidSeason},
		"custom over":   {appprocurement.SeasonPlanInput{Custom: &over}, appprocurement.ErrSeasonOver},
		"backwards":     {appprocurement.SeasonPlanInput{Custom: &backwards}, appprocurement.ErrInvalidSeason},
		"too long":      {appprocurement.SeasonPlanInput{Custom: &tooLong}, appprocurement.ErrInvalidSeason},
		"safety":        {appprocurement.SeasonPlanInput{Event: christmas(), SafetyDays: 91}, appprocurement.ErrInvalidPlanningWindow},
	}
	for name, c := range cases {
		_, err := newSeasonUC(t, &stubCatalogReader{}, &stubDemandReader{}, &stubSupplyReader{}).Execute(context.Background(), c.in)
		assert.ErrorIs(t, err, c.want, name)
	}
}

func TestGetSeasonPlan_PropagatesReaderErrors(t *testing.T) {
	boom := errors.New("boom")
	_, err := newSeasonUC(t, &stubCatalogReader{}, &stubDemandReader{err: boom}, &stubSupplyReader{}).Execute(context.Background(), appprocurement.SeasonPlanInput{Event: christmas()})
	assert.ErrorIs(t, err, boom)
}

func TestListSeasonEvents_NextOccurrenceOfEach(t *testing.T) {
	uc := appprocurement.NewListSeasonEventsUseCase(func() time.Time { return planningNow })

	events := uc.Execute(0)

	require.Len(t, events, len(planning.SeasonEvents()))
	byKey := map[planning.SeasonEventKey]appprocurement.SeasonEventOccurrence{}
	for _, e := range events {
		byKey[e.Key] = e
	}
	assert.Equal(t, 2026, byKey[planning.SeasonChristmas].Year)
	assert.Equal(t, utcDate(2026, time.December, 25), byKey[planning.SeasonChristmas].Date)
	assert.Equal(t, 2027, byKey[planning.SeasonEaster].Year)
	assert.Equal(t, utcDate(2027, time.March, 28), byKey[planning.SeasonEaster].Date)
	// Soonest first.
	assert.Equal(t, planning.SeasonBlackFriday, events[0].Key)
}

func TestListSeasonEvents_ForAYear(t *testing.T) {
	events := appprocurement.NewListSeasonEventsUseCase(func() time.Time { return planningNow }).Execute(2027)
	for _, e := range events {
		assert.Equal(t, 2027, e.Year, e.Key)
	}
}
