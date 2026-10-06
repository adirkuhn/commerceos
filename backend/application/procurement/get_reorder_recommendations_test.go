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

var planningNow = time.Date(2026, 10, 6, 15, 30, 0, 0, time.UTC)
var planningToday = time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)

type stubCatalogReader struct {
	skus []planning.PlanningSKU
	err  error
}

func (s *stubCatalogReader) ListPlanningSKUs(context.Context) ([]planning.PlanningSKU, error) {
	return s.skus, s.err
}

type window struct{ from, to time.Time }

type stubDemandReader struct {
	sold      map[window]map[string]int64
	firstSale map[string]time.Time
	asked     []window
	err       error
}

func (s *stubDemandReader) UnitsSoldPerSKUBetween(_ context.Context, from, to time.Time) (map[string]int64, error) {
	s.asked = append(s.asked, window{from, to})
	return s.sold[window{from, to}], s.err
}

func (s *stubDemandReader) FirstSalePerSKU(context.Context) (map[string]time.Time, error) {
	return s.firstSale, nil
}

type stubSupplyReader struct {
	open     map[string]planning.OpenOrderQuantity
	sourcing map[string]planning.Sourcing
	err      error
}

func (s *stubSupplyReader) OpenOrderQuantities(context.Context) (map[string]planning.OpenOrderQuantity, error) {
	return s.open, s.err
}

func (s *stubSupplyReader) LatestSourcing(context.Context) (map[string]planning.Sourcing, error) {
	return s.sourcing, nil
}

func intPtr(i int) *int { return &i }

// Windows the use case reads, for today = 2026-10-06.
var (
	recentWindow         = window{planningToday.AddDate(0, 0, -56), planningToday}
	lastYearRecentWindow = window{planningToday.AddDate(-1, 0, -56), planningToday.AddDate(-1, 0, 0)}
)

func lastYearTargetWindow(days int) window {
	start := planningToday.AddDate(-1, 0, 0)
	return window{start, start.AddDate(0, 0, days)}
}

func newReorderUC(t *testing.T, catalog *stubCatalogReader, demand *stubDemandReader, supply *stubSupplyReader) *appprocurement.GetReorderRecommendationsUseCase {
	return appprocurement.NewGetReorderRecommendationsUseCase(
		newMockGormDB(t), catalog, demand, supply, func() time.Time { return planningNow })
}

func TestGetReorderRecommendations_SeasonalProductGroupedUnderItsSupplier(t *testing.T) {
	catalog := &stubCatalogReader{skus: []planning.PlanningSKU{
		{SKUID: "panettone", SKUCode: "PAN-1", ProductName: "Panettone", Active: true, OnHand: 12},
	}}
	// lead 21 + coverage 30 = 51 days forecast window.
	demand := &stubDemandReader{
		sold: map[window]map[string]int64{
			recentWindow:             {"panettone": 118},
			lastYearRecentWindow:     {"panettone": 100},
			lastYearTargetWindow(51): {"panettone": 820},
		},
		firstSale: map[string]time.Time{"panettone": time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC)},
	}
	supply := &stubSupplyReader{
		open: map[string]planning.OpenOrderQuantity{"panettone": {Sent: 50, Draft: 8}},
		sourcing: map[string]planning.Sourcing{"panettone": {
			SupplierID: "sup-1", SupplierName: "Bauducco NL", LeadTimeDays: intPtr(21),
			UnitCostMinorUnits: 450, CurrencyCode: "EUR", UnitsPerPurchasedItem: 6,
		}},
	}

	out, err := newReorderUC(t, catalog, demand, supply).Execute(context.Background(), appprocurement.ReorderRecommendationsInput{CoverageDays: 30, SafetyDays: 7})

	require.NoError(t, err)
	assert.Equal(t, planningToday, out.AsOf)
	require.Len(t, out.Suppliers, 1)
	group := out.Suppliers[0]
	require.NotNil(t, group.SupplierID)
	assert.Equal(t, "sup-1", *group.SupplierID)
	assert.Equal(t, "Bauducco NL", group.SupplierName)
	assert.Equal(t, 21, group.LeadTimeDays)
	assert.False(t, group.LeadTimeAssumed)
	require.Len(t, group.Lines, 1)

	line := group.Lines[0]
	assert.Equal(t, planning.ForecastMethodSeasonal, line.Forecast.Method)
	assert.Equal(t, int64(968), line.Forecast.Units) // ceil(820 * 1.18)
	assert.Equal(t, 51, line.ForecastDays)
	assert.Equal(t, int64(118), line.RecentUnits)
	assert.Equal(t, int64(820), line.LastYearUnits)
	assert.Equal(t, planning.OpenOrderQuantity{Sent: 50, Draft: 8}, line.OnOrder)
	// safety = ceil(968/51*7) = 133; need = 968+133-12-58 = 1031 -> 172 boxes of 6.
	assert.Equal(t, int64(1031), line.Reorder.NeededUnits)
	assert.Equal(t, int64(172), line.Reorder.PurchaseQuantity)
	assert.Equal(t, int64(6), line.UnitsPerPurchasedItem)
	require.NotNil(t, line.EstimatedCostMinorUnits)
	assert.Equal(t, int64(172*450), *line.EstimatedCostMinorUnits)
	require.NotNil(t, group.EstimatedCostMinorUnits)
	assert.Equal(t, int64(172*450), *group.EstimatedCostMinorUnits)
	assert.Equal(t, "EUR", group.CurrencyCode)
	require.NotNil(t, group.EarliestOrderByDate)
	assert.Equal(t, *line.Reorder.OrderByDate, *group.EarliestOrderByDate)
}

func TestGetReorderRecommendations_NewProductUsesRecentVelocity(t *testing.T) {
	catalog := &stubCatalogReader{skus: []planning.PlanningSKU{{SKUID: "new", Active: true}}}
	demand := &stubDemandReader{
		sold: map[window]map[string]int64{
			recentWindow:             {"new": 56},
			lastYearTargetWindow(44): {"new": 999}, // can't be: launched this year
		},
		firstSale: map[string]time.Time{"new": planningToday.AddDate(0, -2, 0)},
	}
	out, err := newReorderUC(t, catalog, demand, &stubSupplyReader{}).Execute(context.Background(), appprocurement.ReorderRecommendationsInput{CoverageDays: 30, SafetyDays: 0})

	require.NoError(t, err)
	require.Len(t, out.Suppliers, 1)
	line := out.Suppliers[0].Lines[0]
	assert.Equal(t, planning.ForecastMethodRecentVelocity, line.Forecast.Method)
	assert.Equal(t, int64(44), line.Forecast.Units) // 1/day over 14 default lead + 30
}

// Never bought on a PO: no supplier to order from, default lead time.
func TestGetReorderRecommendations_UnsourcedSKUsGoLastWithAssumedLeadTime(t *testing.T) {
	catalog := &stubCatalogReader{skus: []planning.PlanningSKU{
		{SKUID: "a", Active: true}, {SKUID: "b", Active: true},
	}}
	demand := &stubDemandReader{sold: map[window]map[string]int64{recentWindow: {"a": 560, "b": 560}}}
	supply := &stubSupplyReader{sourcing: map[string]planning.Sourcing{
		"b": {SupplierID: "sup-1", SupplierName: "S", UnitsPerPurchasedItem: 1, CurrencyCode: "EUR"},
	}}

	out, err := newReorderUC(t, catalog, demand, supply).Execute(context.Background(), appprocurement.ReorderRecommendationsInput{CoverageDays: 30, SafetyDays: 7})

	require.NoError(t, err)
	require.Len(t, out.Suppliers, 2)
	assert.Equal(t, "sup-1", *out.Suppliers[0].SupplierID)
	assert.True(t, out.Suppliers[0].LeadTimeAssumed, "supplier without lead time")
	assert.Nil(t, out.Suppliers[1].SupplierID)
	assert.True(t, out.Suppliers[1].LeadTimeAssumed)
	assert.Equal(t, planning.DefaultLeadTimeDays, out.Suppliers[1].LeadTimeDays)
	assert.Nil(t, out.Suppliers[1].Lines[0].EstimatedCostMinorUnits)
	assert.Nil(t, out.Suppliers[1].EstimatedCostMinorUnits)
}

func TestGetReorderRecommendations_SupplierFilter(t *testing.T) {
	catalog := &stubCatalogReader{skus: []planning.PlanningSKU{{SKUID: "a", Active: true}, {SKUID: "b", Active: true}}}
	demand := &stubDemandReader{sold: map[window]map[string]int64{recentWindow: {"a": 560, "b": 560}}}
	supply := &stubSupplyReader{sourcing: map[string]planning.Sourcing{
		"a": {SupplierID: "sup-a", UnitsPerPurchasedItem: 1},
		"b": {SupplierID: "sup-b", UnitsPerPurchasedItem: 1},
	}}
	sup := "sup-b"

	out, err := newReorderUC(t, catalog, demand, supply).Execute(context.Background(), appprocurement.ReorderRecommendationsInput{CoverageDays: 30, SupplierID: &sup})

	require.NoError(t, err)
	require.Len(t, out.Suppliers, 1)
	assert.Equal(t, "sup-b", *out.Suppliers[0].SupplierID)
}

func TestGetReorderRecommendations_SkipsSKUsNeedingNothing(t *testing.T) {
	catalog := &stubCatalogReader{skus: []planning.PlanningSKU{
		{SKUID: "covered", Active: true, OnHand: 1000},
		{SKUID: "dead", Active: true},
		{SKUID: "inactive", Active: false},
	}}
	demand := &stubDemandReader{sold: map[window]map[string]int64{recentWindow: {"covered": 56, "inactive": 560}}}

	out, err := newReorderUC(t, catalog, demand, &stubSupplyReader{}).Execute(context.Background(), appprocurement.ReorderRecommendationsInput{CoverageDays: 30, SafetyDays: 7})

	require.NoError(t, err)
	assert.Empty(t, out.Suppliers)
}

// Carne seca packages sell; the pool (grams) is what gets bought, per kg.
func TestGetReorderRecommendations_WeightPoolBoughtPerKgFromPackageSales(t *testing.T) {
	pool := "pool"
	grams := int64(500)
	catalog := &stubCatalogReader{skus: []planning.PlanningSKU{
		{SKUID: pool, IsWeightPool: true, Active: true, OnHand: 1000},
		{SKUID: "p500", PortionedFromSKUID: &pool, WeightGrams: &grams, Active: true, OnHand: 2},
	}}
	// 112 packages x 500g in 56 days = 1000 g/day.
	demand := &stubDemandReader{sold: map[window]map[string]int64{recentWindow: {"p500": 112}}}

	out, err := newReorderUC(t, catalog, demand, &stubSupplyReader{}).Execute(context.Background(), appprocurement.ReorderRecommendationsInput{CoverageDays: 30, SafetyDays: 0})

	require.NoError(t, err)
	require.Len(t, out.Suppliers, 1)
	require.Len(t, out.Suppliers[0].Lines, 1)
	line := out.Suppliers[0].Lines[0]
	assert.Equal(t, pool, line.SKUID)
	assert.True(t, line.IsWeightPool)
	assert.Equal(t, int64(2000), line.OnHand) // 1000 g + 2 x 500 g packed
	assert.Equal(t, int64(1000), line.UnitsPerPurchasedItem)
	assert.Equal(t, int64(44000-2000), line.Reorder.NeededUnits) // 44 days x 1000 g
	assert.Equal(t, int64(42), line.Reorder.PurchaseQuantity)    // kg
}

func TestGetReorderRecommendations_LinesSortedByOrderByDate(t *testing.T) {
	catalog := &stubCatalogReader{skus: []planning.PlanningSKU{
		{SKUID: "later", Active: true, OnHand: 200}, {SKUID: "sooner", Active: true},
	}}
	demand := &stubDemandReader{sold: map[window]map[string]int64{recentWindow: {"later": 560, "sooner": 560}}}

	out, err := newReorderUC(t, catalog, demand, &stubSupplyReader{}).Execute(context.Background(), appprocurement.ReorderRecommendationsInput{CoverageDays: 30})

	require.NoError(t, err)
	lines := out.Suppliers[0].Lines
	require.Len(t, lines, 2)
	assert.Equal(t, "sooner", lines[0].SKUID)
	assert.Equal(t, "later", lines[1].SKUID)
}

func TestGetReorderRecommendations_RejectsOutOfRangeWindows(t *testing.T) {
	uc := newReorderUC(t, &stubCatalogReader{}, &stubDemandReader{}, &stubSupplyReader{})
	for _, in := range []appprocurement.ReorderRecommendationsInput{
		{CoverageDays: 0}, {CoverageDays: 366}, {CoverageDays: 30, SafetyDays: -1}, {CoverageDays: 30, SafetyDays: 91},
	} {
		_, err := uc.Execute(context.Background(), in)
		assert.ErrorIs(t, err, appprocurement.ErrInvalidPlanningWindow, "%+v", in)
	}
}

func TestGetReorderRecommendations_PropagatesReaderErrors(t *testing.T) {
	boom := errors.New("boom")
	for name, uc := range map[string]*appprocurement.GetReorderRecommendationsUseCase{
		"catalog": newReorderUC(t, &stubCatalogReader{err: boom}, &stubDemandReader{}, &stubSupplyReader{}),
		"demand":  newReorderUC(t, &stubCatalogReader{}, &stubDemandReader{err: boom}, &stubSupplyReader{}),
		"supply":  newReorderUC(t, &stubCatalogReader{}, &stubDemandReader{}, &stubSupplyReader{err: boom}),
	} {
		_, err := uc.Execute(context.Background(), appprocurement.ReorderRecommendationsInput{CoverageDays: 30})
		assert.ErrorIs(t, err, boom, name)
	}
}

// A PO line without a price (cost 0, e.g. a parsed invoice nobody priced)
// is an unknown cost, not a free product.
func TestGetReorderRecommendations_ZeroUnitCostIsUnknown(t *testing.T) {
	catalog := &stubCatalogReader{skus: []planning.PlanningSKU{{SKUID: "a", Active: true}}}
	demand := &stubDemandReader{sold: map[window]map[string]int64{recentWindow: {"a": 560}}}
	supply := &stubSupplyReader{sourcing: map[string]planning.Sourcing{
		"a": {SupplierID: "sup", UnitsPerPurchasedItem: 1, CurrencyCode: "EUR"},
	}}

	out, err := newReorderUC(t, catalog, demand, supply).Execute(context.Background(), appprocurement.ReorderRecommendationsInput{CoverageDays: 30})

	require.NoError(t, err)
	line := out.Suppliers[0].Lines[0]
	assert.Nil(t, line.UnitCostMinorUnits)
	assert.Nil(t, line.EstimatedCostMinorUnits)
	assert.Nil(t, out.Suppliers[0].EstimatedCostMinorUnits)
}
