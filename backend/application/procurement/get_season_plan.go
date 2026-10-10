package procurement

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/domain/errs"
	"time"

	"github.com/adirkuhn/commerceos/domain/procurement/planning"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const maxCustomSeasonDays = 366

var (
	ErrInvalidSeason      = errs.New("PLANNING_SEASON_INVALID", "pick either a season event (with a year up to next year) or a custom date range of at most 366 days")
	ErrUnknownSeasonEvent = errs.New("PLANNING_UNKNOWN_SEASON_EVENT", "unknown season event")
	ErrSeasonOver         = errs.New("PLANNING_SEASON_OVER", "this season is already over")
)

type SeasonPlanInput struct {
	// Event or Custom, not both.
	Event *planning.SeasonEventKey
	// Year of Event; 0 means its next occurrence.
	Year int
	// Custom is compared with the same dates last year.
	Custom     *planning.Window
	SafetyDays int
	SupplierID *string
	// AllProducts includes products that don't sell more in this season.
	AllProducts bool
}

type SeasonPlan struct {
	AsOf  time.Time
	Event *planning.SeasonEventKey
	// Year is the event's year (Window.Start's year for a custom season).
	Year int
	// Window is the season (what's left of it when it already started);
	// LastYearWindow the matching part of last year's season.
	Window         planning.Window
	LastYearWindow planning.Window
	SafetyDays     int
	Suppliers      []SupplierPlan
}

// GetSeasonPlanUseCase answers "what do I need for Christmas, and when must
// I order it": per SKU, forecast the season from last year's same season
// (aligned on the holiday, so Easter compares with last Easter) scaled by
// this year's growth, plus what sells until the season starts; subtract
// stock and open POs; order so it arrives before the season starts. By
// default only products that sell clearly more in the season are listed.
type GetSeasonPlanUseCase struct {
	db      *gorm.DB
	catalog planning.CatalogReader
	demand  planning.DemandReader
	supply  planning.SupplyReader
	now     func() time.Time
}

func NewGetSeasonPlanUseCase(
	db *gorm.DB,
	catalog planning.CatalogReader,
	demand planning.DemandReader,
	supply planning.SupplyReader,
	now func() time.Time,
) *GetSeasonPlanUseCase {
	return &GetSeasonPlanUseCase{db: db, catalog: catalog, demand: demand, supply: supply, now: now}
}

// seasonWindows are the windows a season plan reads, this year and last.
type seasonWindows struct {
	season, lastSeason        planning.Window
	lastPreSeason             planning.Window
	lastBaseline              planning.Window
	preSeasonDays, seasonDays int
}

func (uc *GetSeasonPlanUseCase) Execute(ctx context.Context, in SeasonPlanInput) (*SeasonPlan, error) {
	if in.SafetyDays < 0 || in.SafetyDays > maxPlanningSafetyDays {
		return nil, fmt.Errorf("getSeasonPlan: %w", ErrInvalidPlanningWindow)
	}
	asOf := startOfDay(uc.now().UTC())
	year, season, lastSeason, err := resolveSeason(in, asOf)
	if err != nil {
		return nil, fmt.Errorf("getSeasonPlan: %w", err)
	}
	w := planningWindows(asOf, season, lastSeason)

	var lines []plannedSourcedLine
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		snap, err := loadPlanningSnapshot(ctx, uc.catalog, uc.demand, uc.supply, asOf)
		if err != nil {
			return err
		}
		lastSeasonSold, err := uc.soldBetween(ctx, snap, w.lastSeason)
		if err != nil {
			return err
		}
		lastPreSold, err := uc.soldBetween(ctx, snap, w.lastPreSeason)
		if err != nil {
			return err
		}
		lastBaselineSold, err := uc.soldBetween(ctx, snap, w.lastBaseline)
		if err != nil {
			return err
		}

		for _, sku := range snap.plannable {
			source := snap.sourcingFor(sku)
			if in.SupplierID != nil && (source.supplierID == nil || *source.supplierID != *in.SupplierID) {
				continue
			}
			lift := planning.SeasonalLift(lastSeasonSold[sku.SKUID], w.seasonDays, lastBaselineSold[sku.SKUID])
			if !in.AllProducts && !planning.IsSeasonal(lift, lastSeasonSold[sku.SKUID]) {
				continue
			}

			seasonHistory := snap.history(sku.SKUID, w.lastSeason.Start)
			seasonHistory.LastYearTargetUnits = lastSeasonSold[sku.SKUID]
			seasonForecast := planning.ForecastDemand(seasonHistory, w.seasonDays)
			preHistory := snap.history(sku.SKUID, w.lastPreSeason.Start)
			preHistory.LastYearTargetUnits = lastPreSold[sku.SKUID]
			preForecast := planning.ForecastDemand(preHistory, w.preSeasonDays)

			onOrder := snap.onOrder[sku.SKUID]
			reorder := planning.PlanSeason(planning.SeasonInput{
				AsOf: asOf, Window: w.season,
				PreSeasonUnits: preForecast.Units, SeasonUnits: seasonForecast.Units, SafetyDays: in.SafetyDays,
				OnHand: snap.onHand[sku.SKUID], OnOrder: onOrder.Total(),
				UnitsPerPurchasedItem: source.unitsPerPurchasedItem, LeadTimeDays: source.leadTimeDays,
			})
			if reorder.PurchaseQuantity == 0 {
				continue
			}
			line := newPlannedLine(sku, snap.onHand[sku.SKUID], onOrder, seasonHistory, seasonForecast, w.seasonDays, reorder, source)
			line.PreSeason, line.SeasonalLift = &preForecast, lift
			lines = append(lines, plannedSourcedLine{source: source, line: line})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getSeasonPlan: %w", err)
	}
	return &SeasonPlan{
		AsOf: asOf, Event: in.Event, Year: year, Window: w.season, LastYearWindow: w.lastSeason,
		SafetyDays: in.SafetyDays, Suppliers: groupBySupplier(lines),
	}, nil
}

func (uc *GetSeasonPlanUseCase) soldBetween(ctx context.Context, snap *planningSnapshot, w planning.Window) (map[string]int64, error) {
	if w.Days() <= 0 {
		return map[string]int64{}, nil
	}
	sold, err := uc.demand.UnitsSoldPerSKUBetween(ctx, w.Start, w.End)
	if err != nil {
		return nil, err
	}
	return snap.rollup.Units(sold), nil
}

// resolveSeason returns the season's year, its window and last year's.
func resolveSeason(in SeasonPlanInput, asOf time.Time) (int, planning.Window, planning.Window, error) {
	none := planning.Window{}
	switch {
	case (in.Event == nil) == (in.Custom == nil):
		return 0, none, none, ErrInvalidSeason
	case in.Custom != nil:
		c := *in.Custom
		if !c.End.After(c.Start) || c.Days() > maxCustomSeasonDays {
			return 0, none, none, ErrInvalidSeason
		}
		if !c.End.After(asOf) {
			return 0, none, none, ErrSeasonOver
		}
		return c.Start.Year(), c, c.ShiftYears(-1), nil
	}

	event, ok := planning.FindSeasonEvent(*in.Event)
	if !ok {
		return 0, none, none, ErrUnknownSeasonEvent
	}
	year := in.Year
	if year == 0 {
		year = nextOccurrenceYear(event, asOf)
	}
	if year > asOf.Year()+1 {
		return 0, none, none, ErrInvalidSeason
	}
	if !event.Window(year).End.After(asOf) {
		return 0, none, none, ErrSeasonOver
	}
	return year, event.Window(year), event.Window(year - 1), nil
}

func nextOccurrenceYear(event planning.SeasonEvent, asOf time.Time) int {
	if event.Window(asOf.Year()).End.After(asOf) {
		return asOf.Year()
	}
	return asOf.Year() + 1
}

// planningWindows trims a season that already started to what's left of
// it (and last year's to the same part), then derives the pre-season and
// the baseline before last year's season that seasonal lift compares with.
func planningWindows(asOf time.Time, season, lastSeason planning.Window) seasonWindows {
	if asOf.After(season.Start) {
		elapsed := planning.Window{Start: season.Start, End: asOf}.Days()
		season.Start = asOf
		lastSeason.Start = lastSeason.Start.AddDate(0, 0, elapsed)
	}
	preDays := planning.Window{Start: asOf, End: season.Start}.Days()
	return seasonWindows{
		season: season, lastSeason: lastSeason,
		lastPreSeason: planning.Window{Start: lastSeason.Start.AddDate(0, 0, -preDays), End: lastSeason.Start},
		lastBaseline:  planning.Window{Start: lastSeason.Start.AddDate(0, 0, -planning.RecentWindowDays), End: lastSeason.Start},
		preSeasonDays: preDays, seasonDays: season.Days(),
	}
}
