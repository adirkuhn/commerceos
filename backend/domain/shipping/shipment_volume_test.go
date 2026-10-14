package shipping_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/shipping"
)

func amsterdam(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)
	return loc
}

func TestVolumeGranularity_Valid(t *testing.T) {
	for _, g := range []shipping.VolumeGranularity{"week", "month", "quarter", "year"} {
		assert.True(t, g.Valid(), g)
	}
	for _, g := range []shipping.VolumeGranularity{"", "day", "fortnight"} {
		assert.False(t, g.Valid(), g)
	}
}

func TestVolumeBasis_Valid(t *testing.T) {
	assert.True(t, shipping.VolumeBasisLabelCreated.Valid())
	assert.True(t, shipping.VolumeBasisCarrierScan.Valid())
	assert.False(t, shipping.VolumeBasis("delivered").Valid())
	assert.False(t, shipping.VolumeBasis("").Valid())
}

// Weeks start on Sunday, like the dashboard's sales card.
func TestVolumePeriodStart_weekStartsOnSunday(t *testing.T) {
	loc := amsterdam(t)
	sunday := time.Date(2026, 10, 11, 0, 0, 0, 0, loc)

	for _, at := range []time.Time{
		sunday,
		time.Date(2026, 10, 14, 15, 30, 0, 0, loc), // Wednesday
		time.Date(2026, 10, 17, 23, 59, 0, 0, loc), // Saturday night
	} {
		assert.True(t, sunday.Equal(shipping.VolumePeriodStart(at, shipping.VolumeGranularityWeek, loc)), at)
	}
	nextSunday := time.Date(2026, 10, 18, 0, 0, 0, 0, loc)
	assert.True(t, nextSunday.Equal(shipping.VolumePeriodStart(nextSunday.Add(time.Minute), shipping.VolumeGranularityWeek, loc)))
}

// The bucket is the tenant's local calendar: 23:30 UTC on Saturday is
// already Sunday 01:30 in Amsterdam, so it opens the next week.
func TestVolumePeriodStart_usesTheTenantLocation(t *testing.T) {
	loc := amsterdam(t)
	at := time.Date(2026, 10, 17, 23, 30, 0, 0, time.UTC)

	got := shipping.VolumePeriodStart(at, shipping.VolumeGranularityWeek, loc)

	assert.True(t, time.Date(2026, 10, 18, 0, 0, 0, 0, loc).Equal(got), got)
}

func TestVolumePeriodStart_monthQuarterYear(t *testing.T) {
	loc := amsterdam(t)
	at := time.Date(2026, 8, 20, 12, 0, 0, 0, loc)

	assert.True(t, time.Date(2026, 8, 1, 0, 0, 0, 0, loc).Equal(shipping.VolumePeriodStart(at, shipping.VolumeGranularityMonth, loc)))
	assert.True(t, time.Date(2026, 7, 1, 0, 0, 0, 0, loc).Equal(shipping.VolumePeriodStart(at, shipping.VolumeGranularityQuarter, loc)))
	assert.True(t, time.Date(2026, 1, 1, 0, 0, 0, 0, loc).Equal(shipping.VolumePeriodStart(at, shipping.VolumeGranularityYear, loc)))
}

// A week crossing the DST switch (2026-10-25 in Amsterdam) still starts at
// local midnight on Sunday.
func TestVolumePeriodStart_weekAcrossDSTChange(t *testing.T) {
	loc := amsterdam(t)
	at := time.Date(2026, 10, 28, 9, 0, 0, 0, loc)

	got := shipping.VolumePeriodStart(at, shipping.VolumeGranularityWeek, loc)

	assert.True(t, time.Date(2026, 10, 25, 0, 0, 0, 0, loc).Equal(got), got)
}

// Every period touched by [from, to) is listed, so weeks without a single
// shipment still show up as zero.
func TestFillVolumePeriods_addsEmptyPeriodsInOrder(t *testing.T) {
	loc := amsterdam(t)
	from := time.Date(2026, 1, 15, 0, 0, 0, 0, loc)
	to := time.Date(2026, 4, 2, 0, 0, 0, 0, loc)
	feb := time.Date(2026, 2, 1, 0, 0, 0, 0, loc)

	got := shipping.FillVolumePeriods([]shipping.VolumePoint{{PeriodStart: feb.UTC(), Shipments: 4, Orders: 3}}, from, to, shipping.VolumeGranularityMonth, loc)

	require.Len(t, got, 4)
	assert.True(t, time.Date(2026, 1, 1, 0, 0, 0, 0, loc).Equal(got[0].PeriodStart))
	assert.Equal(t, int64(0), got[0].Shipments)
	assert.True(t, feb.Equal(got[1].PeriodStart))
	assert.Equal(t, int64(4), got[1].Shipments)
	assert.Equal(t, int64(3), got[1].Orders)
	assert.True(t, time.Date(2026, 4, 1, 0, 0, 0, 0, loc).Equal(got[3].PeriodStart))
	assert.Equal(t, loc, got[0].PeriodStart.Location())
}

func TestFillVolumePeriods_emptyRange(t *testing.T) {
	loc := amsterdam(t)
	at := time.Date(2026, 1, 15, 0, 0, 0, 0, loc)

	assert.Empty(t, shipping.FillVolumePeriods(nil, at, at, shipping.VolumeGranularityWeek, loc))
}
