package procurement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/domain/procurement/planning"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const (
	DefaultPlanningCoverageDays = 30
	DefaultPlanningSafetyDays   = 7
	maxPlanningCoverageDays     = 365
	maxPlanningSafetyDays       = 90
)

var ErrInvalidPlanningWindow = errors.New("coverage_days must be 1-365 and safety_days 0-90")

type ReorderRecommendationsInput struct {
	// CoverageDays is how long an order placed today should last once it
	// arrives -- the forecast window is lead time + coverage.
	CoverageDays int
	SafetyDays   int
	// SupplierID narrows the result to one supplier's group.
	SupplierID *string
}

type ReorderRecommendations struct {
	AsOf         time.Time
	CoverageDays int
	SafetyDays   int
	Suppliers    []SupplierPlan
}

// GetReorderRecommendationsUseCase answers "what should I buy now, from
// whom, and by when": per SKU, forecast demand over lead time + coverage
// (seasonal when last year's same window sold, recent velocity otherwise),
// subtract stock on hand and on open POs, round up to the supplier's
// purchased item, group by the supplier it was last bought from.
type GetReorderRecommendationsUseCase struct {
	db      *gorm.DB
	catalog planning.CatalogReader
	demand  planning.DemandReader
	supply  planning.SupplyReader
	now     func() time.Time
}

func NewGetReorderRecommendationsUseCase(
	db *gorm.DB,
	catalog planning.CatalogReader,
	demand planning.DemandReader,
	supply planning.SupplyReader,
	now func() time.Time,
) *GetReorderRecommendationsUseCase {
	return &GetReorderRecommendationsUseCase{db: db, catalog: catalog, demand: demand, supply: supply, now: now}
}

func (uc *GetReorderRecommendationsUseCase) Execute(ctx context.Context, in ReorderRecommendationsInput) (*ReorderRecommendations, error) {
	if in.CoverageDays < 1 || in.CoverageDays > maxPlanningCoverageDays ||
		in.SafetyDays < 0 || in.SafetyDays > maxPlanningSafetyDays {
		return nil, fmt.Errorf("getReorderRecommendations: %w", ErrInvalidPlanningWindow)
	}
	asOf := startOfDay(uc.now())
	lastYear := asOf.AddDate(-1, 0, 0)

	var lines []plannedSourcedLine
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		snap, err := loadPlanningSnapshot(ctx, uc.catalog, uc.demand, uc.supply, asOf)
		if err != nil {
			return err
		}
		// One "last year" read per distinct forecast window length --
		// lead times differ per supplier, so windows do too.
		lastYearByDays := map[int]map[string]int64{}
		for _, sku := range snap.plannable {
			source := snap.sourcingFor(sku)
			if in.SupplierID != nil && (source.supplierID == nil || *source.supplierID != *in.SupplierID) {
				continue
			}
			days := source.leadTimeDays + in.CoverageDays
			lastYearSold, ok := lastYearByDays[days]
			if !ok {
				sold, err := uc.demand.UnitsSoldPerSKUBetween(ctx, lastYear, lastYear.AddDate(0, 0, days))
				if err != nil {
					return err
				}
				lastYearSold = snap.rollup.Units(sold)
				lastYearByDays[days] = lastYearSold
			}

			history := snap.history(sku.SKUID, lastYear)
			history.LastYearTargetUnits = lastYearSold[sku.SKUID]
			forecast := planning.ForecastDemand(history, days)
			onOrder := snap.onOrder[sku.SKUID]
			reorder := planning.PlanReorder(planning.ReorderInput{
				AsOf: asOf, ForecastUnits: forecast.Units, ForecastDays: days, SafetyDays: in.SafetyDays,
				OnHand: snap.onHand[sku.SKUID], OnOrder: onOrder.Total(),
				UnitsPerPurchasedItem: source.unitsPerPurchasedItem, LeadTimeDays: source.leadTimeDays,
			})
			if reorder.PurchaseQuantity == 0 {
				continue
			}
			lines = append(lines, plannedSourcedLine{
				source: source,
				line:   newPlannedLine(sku, snap.onHand[sku.SKUID], onOrder, history, forecast, days, reorder, source),
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getReorderRecommendations: %w", err)
	}
	return &ReorderRecommendations{
		AsOf: asOf, CoverageDays: in.CoverageDays, SafetyDays: in.SafetyDays,
		Suppliers: groupBySupplier(lines),
	}, nil
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
