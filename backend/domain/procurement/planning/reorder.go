package planning

import "time"

// DefaultLeadTimeDays is assumed for a supplier with no LeadTimeDays set.
const DefaultLeadTimeDays = 14

// ReorderInput is one SKU's forecast against what's available. All
// quantities are stock units (grams for a weight pool).
type ReorderInput struct {
	AsOf          time.Time
	ForecastUnits int64
	// ForecastDays is the window ForecastUnits covers.
	ForecastDays int
	// SafetyDays of average demand are kept on top of the forecast.
	SafetyDays int
	OnHand     int64
	// OnOrder is already ordered from a supplier but not received yet.
	OnOrder int64
	// UnitsPerPurchasedItem is how many stock units one purchased item
	// holds (36 for a box of 36, 1000 for a weight pool bought per kg).
	UnitsPerPurchasedItem int64
	LeadTimeDays          int
}

type Reorder struct {
	DailyDemand float64
	SafetyStock int64
	// NeededUnits = forecast + safety stock - on hand - on order, floored at 0.
	NeededUnits int64
	// PurchaseQuantity is NeededUnits in whole purchased items, rounded up.
	PurchaseQuantity int64
	// OrderUnits is PurchaseQuantity in stock units.
	OrderUnits int64
	// StockoutDate is when on hand + on order runs out at DailyDemand; nil
	// without demand.
	StockoutDate *time.Time
	// OrderByDate is StockoutDate minus the lead time -- the last day an
	// order still arrives in time. Before AsOf means it's already late.
	OrderByDate *time.Time
}

func PlanReorder(in ReorderInput) Reorder {
	if in.ForecastDays <= 0 || in.ForecastUnits <= 0 {
		return Reorder{}
	}
	perItem := in.UnitsPerPurchasedItem
	if perItem <= 0 {
		perItem = 1
	}
	daily := float64(in.ForecastUnits) / float64(in.ForecastDays)
	safety := ceilUnits(daily * float64(in.SafetyDays))
	available := max(in.OnHand, 0) + max(in.OnOrder, 0)

	needed := max(in.ForecastUnits+safety-available, 0)
	quantity := (needed + perItem - 1) / perItem

	stockout := in.AsOf.Add(time.Duration(float64(available) / daily * float64(24*time.Hour)))
	stockout = truncateToDay(stockout, in.AsOf)
	orderBy := stockout.AddDate(0, 0, -in.LeadTimeDays)

	return Reorder{
		DailyDemand: daily, SafetyStock: safety, NeededUnits: needed,
		PurchaseQuantity: quantity, OrderUnits: quantity * perItem,
		StockoutDate: &stockout, OrderByDate: &orderBy,
	}
}

// truncateToDay drops the time of day so dates read as calendar days in
// AsOf's location.
func truncateToDay(t, asOf time.Time) time.Time {
	t = t.In(asOf.Location())
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, asOf.Location())
}
