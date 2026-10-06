package planning

import "time"

// MinSeasonalLift: a season sells at least this many times its usual daily
// rate to count as seasonal for the product.
const MinSeasonalLift = 1.5

// SeasonInput is one SKU's stock against a season. Quantities are stock units.
type SeasonInput struct {
	AsOf time.Time
	// Window is the season; when it already started, it's counted from AsOf.
	Window Window
	// PreSeasonUnits will sell between AsOf and the season start.
	PreSeasonUnits int64
	// SeasonUnits will sell during the (remaining) season.
	SeasonUnits           int64
	SafetyDays            int
	OnHand                int64
	OnOrder               int64
	UnitsPerPurchasedItem int64
	LeadTimeDays          int
}

// PlanSeason is how much to buy so stock lasts through the season, and by
// when: the order must arrive before the season starts, or earlier if
// stock runs out before then.
func PlanSeason(in SeasonInput) Reorder {
	start := in.Window.Start
	if in.AsOf.After(start) {
		start = in.AsOf
	}
	seasonDays := Window{Start: start, End: in.Window.End}.Days()
	if seasonDays <= 0 || in.PreSeasonUnits+in.SeasonUnits <= 0 {
		return Reorder{}
	}
	perItem := in.UnitsPerPurchasedItem
	if perItem <= 0 {
		perItem = 1
	}

	daily := float64(in.SeasonUnits) / float64(seasonDays)
	safety := ceilUnits(daily * float64(in.SafetyDays))
	available := max(in.OnHand, 0) + max(in.OnOrder, 0)
	needed := max(in.PreSeasonUnits+in.SeasonUnits+safety-available, 0)
	quantity := (needed + perItem - 1) / perItem

	stockout := seasonStockout(in, start, daily, available)
	orderBy := start
	if stockout.Before(orderBy) {
		orderBy = stockout
	}
	orderBy = orderBy.AddDate(0, 0, -in.LeadTimeDays)

	return Reorder{
		DailyDemand: daily, SafetyStock: safety, NeededUnits: needed,
		PurchaseQuantity: quantity, OrderUnits: quantity * perItem,
		StockoutDate: &stockout, OrderByDate: &orderBy,
	}
}

// seasonStockout is when available stock runs out: during the pre-season
// at its own pace, else during the season at daily, else at the season end.
func seasonStockout(in SeasonInput, start time.Time, daily float64, available int64) time.Time {
	preDays := Window{Start: in.AsOf, End: start}.Days()
	if available < in.PreSeasonUnits && preDays > 0 {
		preDaily := float64(in.PreSeasonUnits) / float64(preDays)
		return truncateToDay(addDays(in.AsOf, float64(available)/preDaily), in.AsOf)
	}
	left := available - in.PreSeasonUnits
	if daily <= 0 {
		return in.Window.End
	}
	stockout := truncateToDay(addDays(start, float64(left)/daily), in.AsOf)
	if stockout.After(in.Window.End) {
		return in.Window.End
	}
	return stockout
}

func addDays(t time.Time, days float64) time.Time {
	return t.Add(time.Duration(days * float64(24*time.Hour)))
}

// SeasonalLift is how many times its usual daily rate a product sold in
// a season: season units/day vs the RecentWindowDays before the season.
// Nil when nothing sold before -- unboundedly seasonal.
func SeasonalLift(seasonUnits int64, seasonDays int, preSeasonUnits int64) *float64 {
	if preSeasonUnits <= 0 || seasonDays <= 0 {
		return nil
	}
	lift := (float64(seasonUnits) / float64(seasonDays)) / (float64(preSeasonUnits) / RecentWindowDays)
	return &lift
}

// IsSeasonal: sold in the season, well above its usual rate.
func IsSeasonal(lift *float64, seasonUnits int64) bool {
	if seasonUnits <= 0 {
		return false
	}
	return lift == nil || *lift >= MinSeasonalLift
}
