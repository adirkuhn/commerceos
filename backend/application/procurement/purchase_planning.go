package procurement

import (
	"context"
	"sort"
	"time"

	"github.com/adirkuhn/commerceos/domain/procurement/planning"
)

// weightPoolUnitsPerPurchasedItem: a weight pool is bought per kg (its
// stock is grams) unless a past PO says otherwise -- same default the PO
// line form uses.
const weightPoolUnitsPerPurchasedItem = 1000

// PlannedLine is one SKU's purchase recommendation. Quantities are stock
// units (grams for a weight pool) except Reorder.PurchaseQuantity, which
// is in purchased items.
type PlannedLine struct {
	SKUID        string
	SKUCode      string
	ProductName  string
	VariantName  string
	IsWeightPool bool
	OnHand       int64
	OnOrder      planning.OpenOrderQuantity
	// RecentUnits sold in the last planning.RecentWindowDays.
	RecentUnits int64
	// LastYearUnits sold in the forecast window one year earlier.
	LastYearUnits         int64
	Forecast              planning.Forecast
	ForecastDays          int
	Reorder               planning.Reorder
	UnitsPerPurchasedItem int64
	// UnitCostMinorUnits/EstimatedCostMinorUnits are nil when the SKU was
	// never bought on a PO. Cost is per purchased item.
	UnitCostMinorUnits      *int64
	EstimatedCostMinorUnits *int64
	CurrencyCode            string
	LastOrderedAt           *time.Time
}

// SupplierPlan groups the lines to order from one supplier. SupplierID is
// nil for SKUs never bought on a PO.
type SupplierPlan struct {
	SupplierID      *string
	SupplierName    string
	LeadTimeDays    int
	LeadTimeAssumed bool
	Lines           []PlannedLine
	// EstimatedCostMinorUnits is nil unless every line has a cost in one currency.
	EstimatedCostMinorUnits *int64
	CurrencyCode            string
	EarliestOrderByDate     *time.Time
}

// planningSnapshot is everything a plan reads once, rolled up so weight
// pool packages count as their pool.
type planningSnapshot struct {
	rollup    *planning.Rollup
	plannable []planning.PlanningSKU
	onHand    map[string]int64
	onOrder   map[string]planning.OpenOrderQuantity
	sourcing  map[string]planning.Sourcing
	firstSale map[string]time.Time
	// recent/lastYearRecent: sold in the last planning.RecentWindowDays,
	// and in the same days one year earlier.
	recent         map[string]int64
	lastYearRecent map[string]int64
}

func loadPlanningSnapshot(ctx context.Context, catalog planning.CatalogReader, demand planning.DemandReader, supply planning.SupplyReader, asOf time.Time) (*planningSnapshot, error) {
	skus, err := catalog.ListPlanningSKUs(ctx)
	if err != nil {
		return nil, err
	}
	rollup := planning.NewRollup(skus)
	stock := make(map[string]int64, len(skus))
	for _, s := range skus {
		stock[s.SKUID] = s.OnHand
	}

	recentFrom := asOf.AddDate(0, 0, -planning.RecentWindowDays)
	recent, err := demand.UnitsSoldPerSKUBetween(ctx, recentFrom, asOf)
	if err != nil {
		return nil, err
	}
	lastYearRecent, err := demand.UnitsSoldPerSKUBetween(ctx, recentFrom.AddDate(-1, 0, 0), asOf.AddDate(-1, 0, 0))
	if err != nil {
		return nil, err
	}
	firstSale, err := demand.FirstSalePerSKU(ctx)
	if err != nil {
		return nil, err
	}
	openOrders, err := supply.OpenOrderQuantities(ctx)
	if err != nil {
		return nil, err
	}
	sourcing, err := supply.LatestSourcing(ctx)
	if err != nil {
		return nil, err
	}

	return &planningSnapshot{
		rollup: rollup, plannable: rollup.Plannable(),
		onHand: rollup.Units(stock), onOrder: openOrders, sourcing: sourcing,
		firstSale:      rollup.Earliest(firstSale),
		recent:         rollup.Units(recent),
		lastYearRecent: rollup.Units(lastYearRecent),
	}, nil
}

// history is a SKU's demand history minus the forecast window itself,
// which differs per plan. lastYearWindowStart decides whether the SKU
// already sold back then.
func (s *planningSnapshot) history(skuID string, lastYearWindowStart time.Time) planning.DemandHistory {
	first, sold := s.firstSale[skuID]
	return planning.DemandHistory{
		RecentUnits: s.recent[skuID], RecentDays: planning.RecentWindowDays,
		LastYearRecentUnits: s.lastYearRecent[skuID],
		HasLastYearHistory:  sold && first.Before(lastYearWindowStart),
	}
}

// skuSource is where to buy a SKU, with defaults filled in for a SKU (or
// supplier) the PO history knows nothing about.
type skuSource struct {
	supplierID            *string
	supplierName          string
	leadTimeDays          int
	leadTimeAssumed       bool
	unitsPerPurchasedItem int64
	unitCostMinorUnits    *int64
	currencyCode          string
	lastOrderedAt         *time.Time
}

func (s *planningSnapshot) sourcingFor(sku planning.PlanningSKU) skuSource {
	src := skuSource{leadTimeDays: planning.DefaultLeadTimeDays, leadTimeAssumed: true, unitsPerPurchasedItem: 1}
	if sku.IsWeightPool {
		src.unitsPerPurchasedItem = weightPoolUnitsPerPurchasedItem
	}
	last, ok := s.sourcing[sku.SKUID]
	if !ok {
		return src
	}
	supplierID, cost, orderedAt := last.SupplierID, last.UnitCostMinorUnits, last.LastOrderedAt
	src.supplierID, src.supplierName = &supplierID, last.SupplierName
	src.currencyCode, src.lastOrderedAt = last.CurrencyCode, &orderedAt
	if cost > 0 { // an unpriced PO line is an unknown cost, not a free product
		src.unitCostMinorUnits = &cost
	}
	if last.UnitsPerPurchasedItem > 0 {
		src.unitsPerPurchasedItem = last.UnitsPerPurchasedItem
	}
	if last.LeadTimeDays != nil {
		src.leadTimeDays, src.leadTimeAssumed = *last.LeadTimeDays, false
	}
	return src
}

func newPlannedLine(sku planning.PlanningSKU, onHand int64, onOrder planning.OpenOrderQuantity, history planning.DemandHistory, forecast planning.Forecast, days int, reorder planning.Reorder, src skuSource) PlannedLine {
	line := PlannedLine{
		SKUID: sku.SKUID, SKUCode: sku.SKUCode, ProductName: sku.ProductName, VariantName: sku.VariantName,
		IsWeightPool: sku.IsWeightPool, OnHand: onHand, OnOrder: onOrder,
		RecentUnits: history.RecentUnits, LastYearUnits: history.LastYearTargetUnits,
		Forecast: forecast, ForecastDays: days, Reorder: reorder,
		UnitsPerPurchasedItem: src.unitsPerPurchasedItem, UnitCostMinorUnits: src.unitCostMinorUnits,
		CurrencyCode: src.currencyCode, LastOrderedAt: src.lastOrderedAt,
	}
	if src.unitCostMinorUnits != nil {
		cost := *src.unitCostMinorUnits * reorder.PurchaseQuantity
		line.EstimatedCostMinorUnits = &cost
	}
	return line
}

type plannedSourcedLine struct {
	source skuSource
	line   PlannedLine
}

// groupBySupplier groups lines per supplier, most urgent first within and
// across groups; SKUs without a supplier come last.
func groupBySupplier(lines []plannedSourcedLine) []SupplierPlan {
	index := map[string]int{}
	var plans []SupplierPlan
	var unsourced *SupplierPlan
	for _, l := range lines {
		if l.source.supplierID == nil {
			if unsourced == nil {
				unsourced = &SupplierPlan{LeadTimeDays: l.source.leadTimeDays, LeadTimeAssumed: true}
			}
			unsourced.Lines = append(unsourced.Lines, l.line)
			continue
		}
		i, ok := index[*l.source.supplierID]
		if !ok {
			i = len(plans)
			index[*l.source.supplierID] = i
			plans = append(plans, SupplierPlan{
				SupplierID: l.source.supplierID, SupplierName: l.source.supplierName,
				LeadTimeDays: l.source.leadTimeDays, LeadTimeAssumed: l.source.leadTimeAssumed,
			})
		}
		plans[i].Lines = append(plans[i].Lines, l.line)
	}
	if unsourced != nil {
		plans = append(plans, *unsourced)
	}
	for i := range plans {
		finishSupplierPlan(&plans[i])
	}
	sorted := plans
	if unsourced != nil {
		sorted = plans[:len(plans)-1]
	}
	sort.SliceStable(sorted, func(a, b int) bool {
		return dateBefore(sorted[a].EarliestOrderByDate, sorted[b].EarliestOrderByDate)
	})
	return plans
}

func finishSupplierPlan(p *SupplierPlan) {
	sort.SliceStable(p.Lines, func(a, b int) bool {
		return dateBefore(p.Lines[a].Reorder.OrderByDate, p.Lines[b].Reorder.OrderByDate)
	})
	p.EarliestOrderByDate = p.Lines[0].Reorder.OrderByDate
	p.CurrencyCode = p.Lines[0].CurrencyCode
	var total int64
	for _, l := range p.Lines {
		if l.EstimatedCostMinorUnits == nil || l.CurrencyCode != p.CurrencyCode {
			return
		}
		total += *l.EstimatedCostMinorUnits
	}
	p.EstimatedCostMinorUnits = &total
}

// dateBefore orders nil (no demand date) last.
func dateBefore(a, b *time.Time) bool {
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	return a.Before(*b)
}
