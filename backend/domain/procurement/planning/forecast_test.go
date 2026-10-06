package planning_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/procurement/planning"
)

func TestForecastDemand_SeasonalUsesLastYearsWindowScaledByGrowth(t *testing.T) {
	// Panettone: 820 sold in last year's window; the last 8 weeks sold 118
	// vs 100 in the same 8 weeks a year ago -> 18% growth.
	f := planning.ForecastDemand(planning.DemandHistory{
		RecentUnits: 118, RecentDays: 56, LastYearRecentUnits: 100,
		LastYearTargetUnits: 820, HasLastYearHistory: true,
	}, 42)

	assert.Equal(t, planning.ForecastMethodSeasonal, f.Method)
	assert.InDelta(t, 1.18, f.GrowthFactor, 0.0001)
	assert.Equal(t, int64(968), f.Units) // ceil(820 * 1.18) = 967.6 -> 968
}

func TestForecastDemand_SeasonalGrowthIsClamped(t *testing.T) {
	high := planning.ForecastDemand(planning.DemandHistory{
		RecentUnits: 1000, RecentDays: 56, LastYearRecentUnits: 10,
		LastYearTargetUnits: 100, HasLastYearHistory: true,
	}, 30)
	assert.Equal(t, planning.MaxGrowthFactor, high.GrowthFactor)
	assert.Equal(t, int64(200), high.Units)

	low := planning.ForecastDemand(planning.DemandHistory{
		RecentUnits: 1, RecentDays: 56, LastYearRecentUnits: 100,
		LastYearTargetUnits: 100, HasLastYearHistory: true,
	}, 30)
	assert.Equal(t, planning.MinGrowthFactor, low.GrowthFactor)
	assert.Equal(t, int64(50), low.Units)
}

// No sales in one of the two comparison windows says nothing about growth
// (e.g. product was out of stock all of last autumn) -- keep last year as-is.
func TestForecastDemand_SeasonalWithoutComparableRecentWindowsHasNoGrowth(t *testing.T) {
	f := planning.ForecastDemand(planning.DemandHistory{
		RecentUnits: 0, RecentDays: 56, LastYearRecentUnits: 40,
		LastYearTargetUnits: 300, HasLastYearHistory: true,
	}, 30)
	assert.Equal(t, planning.ForecastMethodSeasonal, f.Method)
	assert.Equal(t, 1.0, f.GrowthFactor)
	assert.Equal(t, int64(300), f.Units)
}

func TestForecastDemand_FallsBackToRecentVelocityWithoutLastYearHistory(t *testing.T) {
	f := planning.ForecastDemand(planning.DemandHistory{
		RecentUnits: 56, RecentDays: 56, LastYearTargetUnits: 500, HasLastYearHistory: false,
	}, 30)
	assert.Equal(t, planning.ForecastMethodRecentVelocity, f.Method)
	assert.Equal(t, 1.0, f.GrowthFactor)
	assert.Equal(t, int64(30), f.Units)
}

// Selling now although nothing sold in this window last year (newly
// popular, or last year's stock ran out) -- trust what sells today.
func TestForecastDemand_FallsBackToRecentVelocityWhenLastYearWindowSoldNothing(t *testing.T) {
	f := planning.ForecastDemand(planning.DemandHistory{
		RecentUnits: 28, RecentDays: 56, LastYearRecentUnits: 0,
		LastYearTargetUnits: 0, HasLastYearHistory: true,
	}, 10)
	assert.Equal(t, planning.ForecastMethodRecentVelocity, f.Method)
	assert.Equal(t, int64(5), f.Units)
}

func TestForecastDemand_NoDemand(t *testing.T) {
	f := planning.ForecastDemand(planning.DemandHistory{RecentDays: 56, HasLastYearHistory: true}, 30)
	assert.Equal(t, planning.ForecastMethodNoDemand, f.Method)
	assert.Equal(t, int64(0), f.Units)
}

func TestForecastDemand_ZeroDaysIsNoDemand(t *testing.T) {
	f := planning.ForecastDemand(planning.DemandHistory{RecentUnits: 10, RecentDays: 56}, 0)
	assert.Equal(t, planning.ForecastMethodNoDemand, f.Method)
	assert.Equal(t, int64(0), f.Units)
}
