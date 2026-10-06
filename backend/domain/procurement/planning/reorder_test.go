package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/procurement/planning"
)

var asOf = time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)

func TestPlanReorder_NeedCoversForecastPlusSafetyMinusAvailable(t *testing.T) {
	r := planning.PlanReorder(planning.ReorderInput{
		AsOf: asOf, ForecastUnits: 100, ForecastDays: 50, SafetyDays: 5,
		OnHand: 20, OnOrder: 10, UnitsPerPurchasedItem: 1, LeadTimeDays: 14,
	})

	assert.InDelta(t, 2.0, r.DailyDemand, 0.0001)
	assert.Equal(t, int64(10), r.SafetyStock)
	assert.Equal(t, int64(80), r.NeededUnits) // 100 + 10 - 20 - 10
	assert.Equal(t, int64(80), r.PurchaseQuantity)
	assert.Equal(t, int64(80), r.OrderUnits)
	require.NotNil(t, r.StockoutDate)
	assert.Equal(t, asOf.AddDate(0, 0, 15), *r.StockoutDate) // 30 available / 2 per day
	require.NotNil(t, r.OrderByDate)
	assert.Equal(t, asOf.AddDate(0, 0, 1), *r.OrderByDate) // stockout - 14d lead time
}

// A supplier selling 36-unit boxes, or a weight pool booked per kg
// (1000 g per purchased item), is ordered in whole purchased items.
func TestPlanReorder_RoundsUpToWholePurchasedItems(t *testing.T) {
	r := planning.PlanReorder(planning.ReorderInput{
		AsOf: asOf, ForecastUnits: 12300, ForecastDays: 30,
		OnHand: 0, UnitsPerPurchasedItem: 1000, LeadTimeDays: 7,
	})
	assert.Equal(t, int64(12300), r.NeededUnits)
	assert.Equal(t, int64(13), r.PurchaseQuantity)
	assert.Equal(t, int64(13000), r.OrderUnits)
}

func TestPlanReorder_NothingNeededWhenStockCovers(t *testing.T) {
	r := planning.PlanReorder(planning.ReorderInput{
		AsOf: asOf, ForecastUnits: 30, ForecastDays: 30, SafetyDays: 7,
		OnHand: 100, UnitsPerPurchasedItem: 1, LeadTimeDays: 14,
	})
	assert.Equal(t, int64(0), r.NeededUnits)
	assert.Equal(t, int64(0), r.PurchaseQuantity)
	assert.Equal(t, int64(0), r.OrderUnits)
}

// Oversold stock (negative on hand) is nothing available, not a debt that
// inflates the order beyond the forecast.
func TestPlanReorder_NegativeOnHandCountsAsZero(t *testing.T) {
	r := planning.PlanReorder(planning.ReorderInput{
		AsOf: asOf, ForecastUnits: 10, ForecastDays: 10,
		OnHand: -5, UnitsPerPurchasedItem: 1, LeadTimeDays: 3,
	})
	assert.Equal(t, int64(10), r.NeededUnits)
	require.NotNil(t, r.StockoutDate)
	assert.Equal(t, asOf, *r.StockoutDate)
}

func TestPlanReorder_NoDemandHasNoDates(t *testing.T) {
	r := planning.PlanReorder(planning.ReorderInput{
		AsOf: asOf, ForecastUnits: 0, ForecastDays: 30, SafetyDays: 7,
		OnHand: 5, UnitsPerPurchasedItem: 1, LeadTimeDays: 14,
	})
	assert.Equal(t, 0.0, r.DailyDemand)
	assert.Nil(t, r.StockoutDate)
	assert.Nil(t, r.OrderByDate)
	assert.Equal(t, int64(0), r.PurchaseQuantity)
}

func TestPlanReorder_ZeroUnitsPerPurchasedItemMeansOne(t *testing.T) {
	r := planning.PlanReorder(planning.ReorderInput{
		AsOf: asOf, ForecastUnits: 7, ForecastDays: 7, LeadTimeDays: 1,
	})
	assert.Equal(t, int64(7), r.PurchaseQuantity)
	assert.Equal(t, int64(7), r.OrderUnits)
}
