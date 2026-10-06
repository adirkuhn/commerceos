// Package planning is the Purchase Planner's pure domain logic: forecast
// how much of a SKU will sell in a window, then how much to buy and by
// when. Rule-based and explainable (no ML) -- every number traces back to
// sales history the merchant can see.
package planning

import "math"

type ForecastMethod string

const (
	// ForecastMethodSeasonal: units sold in the same window last year,
	// scaled by GrowthFactor -- what makes Panettone show up before
	// Christmas although it barely sells in October.
	ForecastMethodSeasonal ForecastMethod = "seasonal"
	// ForecastMethodRecentVelocity: the last RecentWindowDays' average
	// daily sales, projected over the window.
	ForecastMethodRecentVelocity ForecastMethod = "recent_velocity"
	ForecastMethodNoDemand       ForecastMethod = "no_demand"
)

const (
	// RecentWindowDays is the "how is it selling lately" window, both for
	// velocity and for comparing this year against last year.
	RecentWindowDays = 56
	// MinGrowthFactor/MaxGrowthFactor bound year-over-year growth: one odd
	// month (a stockout, a one-off big order) must not halve or triple a
	// whole season's order.
	MinGrowthFactor = 0.5
	MaxGrowthFactor = 2.0
)

// DemandHistory is one SKU's sales in the windows a forecast compares.
// Units are stock units (grams for a weight pool).
type DemandHistory struct {
	// RecentUnits sold in the last RecentDays.
	RecentUnits int64
	RecentDays  int
	// LastYearRecentUnits sold in the same RecentDays one year earlier.
	LastYearRecentUnits int64
	// LastYearTargetUnits sold in the forecast window one year earlier.
	LastYearTargetUnits int64
	// HasLastYearHistory is true when the SKU was already selling before
	// last year's forecast window started -- otherwise "sold 0 last year"
	// means "didn't exist yet", not "doesn't sell then".
	HasLastYearHistory bool
}

type Forecast struct {
	Units  int64
	Method ForecastMethod
	// GrowthFactor is 1 unless Method is seasonal and both recent windows
	// sold something.
	GrowthFactor float64
}

// ForecastDemand forecasts units sold over the next days. Seasonal when
// last year's same window sold something, recent velocity otherwise.
func ForecastDemand(h DemandHistory, days int) Forecast {
	if days <= 0 {
		return Forecast{Method: ForecastMethodNoDemand, GrowthFactor: 1}
	}
	if h.HasLastYearHistory && h.LastYearTargetUnits > 0 {
		growth := growthFactor(h.RecentUnits, h.LastYearRecentUnits)
		return Forecast{
			Units:        ceilUnits(float64(h.LastYearTargetUnits) * growth),
			Method:       ForecastMethodSeasonal,
			GrowthFactor: growth,
		}
	}
	if h.RecentUnits > 0 && h.RecentDays > 0 {
		return Forecast{
			Units:        ceilUnits(float64(h.RecentUnits) / float64(h.RecentDays) * float64(days)),
			Method:       ForecastMethodRecentVelocity,
			GrowthFactor: 1,
		}
	}
	return Forecast{Method: ForecastMethodNoDemand, GrowthFactor: 1}
}

func growthFactor(recent, lastYearRecent int64) float64 {
	if recent <= 0 || lastYearRecent <= 0 {
		return 1
	}
	g := float64(recent) / float64(lastYearRecent)
	return math.Min(MaxGrowthFactor, math.Max(MinGrowthFactor, g))
}

// ceilUnits rounds up, ignoring float noise (820*1.18 must not become 968.0000001 -> 969).
func ceilUnits(v float64) int64 {
	return int64(math.Ceil(v - 1e-9))
}
