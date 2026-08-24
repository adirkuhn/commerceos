package reports

import (
	"context"
	"fmt"
	"math"
	"time"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const (
	stockoutRiskStockLevelsPerPage  = 500
	stockoutRiskDefaultLeadTimeDays = 14
)

// StockoutRiskRow is one SKU's stock-vs-demand picture -- the Phase 2 v1
// "Stockout Risk / Reorder Point" report. DailyVelocity is units sold in
// the queried period divided by the period's length in days.
// DaysOfStockRemaining is nil when DailyVelocity is 0 (no recent demand,
// so no meaningful "days until empty"). AtRisk is true when stock will run
// out before leadTimeDays elapses -- i.e. a reorder placed today, with
// this lead time, would still arrive too late.
type StockoutRiskRow struct {
	SKUID                    string
	SKUCode                  string
	ProductName              string
	StockOnHand              int64
	QuantitySoldInPeriod     int64
	DailyVelocity            float64
	DaysOfStockRemaining     *float64
	AtRisk                   bool
	SuggestedReorderQuantity int64
}

// StockoutRiskReport lists every SKU with stock history (same "every
// stocked-or-sold SKU" scope as Inventory + Sales), flagging which ones
// will stock out before a reorder placed today -- at the given lead time
// -- could arrive. Capped to the first stockoutRiskStockLevelsPerPage
// stocked SKUs, same v1 limitation as the other stock-level-backed
// reports. Rule-based v1: no demand forecasting model, just a simple
// average-velocity projection over the queried period -- consistent with
// every other Phase 0/1 report's "no ML yet" approach.
type StockoutRiskReport struct {
	Rows []StockoutRiskRow
}

type GetStockoutRiskReportUseCase struct {
	db                 *gorm.DB
	stockLevels        stockLevelLister
	productPerformance productPerformanceReporter
	skuLookup          skuCostPriceLookup
	variantLookup      variantLookupForSlowMoving
	productLookup      productLookupForSlowMoving
}

func NewGetStockoutRiskReportUseCase(
	db *gorm.DB,
	stockLevels stockLevelLister,
	productPerformance productPerformanceReporter,
	skuLookup skuCostPriceLookup,
	variantLookup variantLookupForSlowMoving,
	productLookup productLookupForSlowMoving,
) *GetStockoutRiskReportUseCase {
	return &GetStockoutRiskReportUseCase{
		db: db, stockLevels: stockLevels, productPerformance: productPerformance,
		skuLookup: skuLookup, variantLookup: variantLookup, productLookup: productLookup,
	}
}

// Execute reports stockout risk for [from, to). leadTimeDays <= 0 means
// "use the default" (stockoutRiskDefaultLeadTimeDays), not "no lead time".
func (uc *GetStockoutRiskReportUseCase) Execute(ctx context.Context, from, to time.Time, leadTimeDays int) (*StockoutRiskReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getStockoutRiskReport: %w", ErrToBeforeFrom)
	}
	if leadTimeDays <= 0 {
		leadTimeDays = stockoutRiskDefaultLeadTimeDays
	}
	periodDays := to.Sub(from).Hours() / 24

	// Each of these opens and closes its own transaction -- sequential,
	// not nested, since neither is in progress while the other runs.
	levels, err := uc.stockLevels.Execute(ctx, invapp.ListStockLevelsInput{Page: 1, PerPage: stockoutRiskStockLevelsPerPage})
	if err != nil {
		return nil, fmt.Errorf("getStockoutRiskReport: %w", err)
	}
	perf, err := uc.productPerformance.Execute(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("getStockoutRiskReport: %w", err)
	}

	soldInPeriod := make(map[string]int64, len(perf.Rows))
	for _, row := range perf.Rows {
		if row.SKUID != nil {
			soldInPeriod[*row.SKUID] = row.QuantitySold
		}
	}

	var report StockoutRiskReport
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		for _, level := range levels.Levels {
			sku, err := uc.skuLookup.FindByID(ctx, level.SKUID)
			if err != nil {
				return err
			}
			productName := ""
			if variant, err := uc.variantLookup.FindByID(ctx, sku.VariantID); err == nil {
				if product, err := uc.productLookup.FindByID(ctx, variant.ProductID); err == nil {
					productName = product.Name
				}
			}

			qtySold := soldInPeriod[level.SKUID]
			velocity := float64(qtySold) / periodDays

			var daysRemaining *float64
			var atRisk bool
			var reorderQty int64
			if velocity > 0 {
				d := float64(level.Total) / velocity
				daysRemaining = &d
				atRisk = d < float64(leadTimeDays)
				qty := int64(math.Ceil(velocity*float64(leadTimeDays) - float64(level.Total)))
				if qty > 0 {
					reorderQty = qty
				}
			}

			report.Rows = append(report.Rows, StockoutRiskRow{
				SKUID: level.SKUID, SKUCode: sku.SKUCode, ProductName: productName,
				StockOnHand: level.Total, QuantitySoldInPeriod: qtySold, DailyVelocity: velocity,
				DaysOfStockRemaining: daysRemaining, AtRisk: atRisk, SuggestedReorderQuantity: reorderQty,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getStockoutRiskReport: %w", err)
	}
	return &report, nil
}
