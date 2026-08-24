package reports

import (
	"context"
	"fmt"
	"time"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const inventorySalesStockLevelsPerPage = 500

// InventorySalesRow is one SKU's stock on hand alongside units sold in the
// queried period -- the Phase 0 "Inventory + Sales Report". Unlike the
// Slow-Moving report, every stocked-or-sold SKU appears here, not just the
// zero-sales ones.
type InventorySalesRow struct {
	SKUID                string
	SKUCode              string
	ProductName          string
	StockOnHand          int64
	QuantitySoldInPeriod int64
	// Units sold as a fraction of current stock on hand, in basis points
	// (1bp = 0.01%, so 2500 = 25.00%, can exceed 10000 if the SKU was
	// restocked mid-period). Nil when StockOnHand is 0 -- no denominator
	// to compute a rate from.
	TurnoverBasisPoints *int64
}

// InventorySalesReport lists every SKU with stock history (movement ledger
// activity), joining its current stock on hand with units sold in
// [From, To). Capped to the first inventorySalesStockLevelsPerPage stocked
// SKUs, same v1 limitation as the Slow-Moving report.
type InventorySalesReport struct {
	Rows []InventorySalesRow
}

type GetInventorySalesReportUseCase struct {
	db                 *gorm.DB
	stockLevels        stockLevelLister
	productPerformance productPerformanceReporter
	skuLookup          skuCostPriceLookup
	variantLookup      variantLookupForSlowMoving
	productLookup      productLookupForSlowMoving
}

func NewGetInventorySalesReportUseCase(
	db *gorm.DB,
	stockLevels stockLevelLister,
	productPerformance productPerformanceReporter,
	skuLookup skuCostPriceLookup,
	variantLookup variantLookupForSlowMoving,
	productLookup productLookupForSlowMoving,
) *GetInventorySalesReportUseCase {
	return &GetInventorySalesReportUseCase{
		db: db, stockLevels: stockLevels, productPerformance: productPerformance,
		skuLookup: skuLookup, variantLookup: variantLookup, productLookup: productLookup,
	}
}

func (uc *GetInventorySalesReportUseCase) Execute(ctx context.Context, from, to time.Time) (*InventorySalesReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getInventorySalesReport: %w", ErrToBeforeFrom)
	}

	// Each of these opens and closes its own transaction -- sequential,
	// not nested, since neither is in progress while the other runs.
	levels, err := uc.stockLevels.Execute(ctx, invapp.ListStockLevelsInput{Page: 1, PerPage: inventorySalesStockLevelsPerPage})
	if err != nil {
		return nil, fmt.Errorf("getInventorySalesReport: %w", err)
	}
	perf, err := uc.productPerformance.Execute(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("getInventorySalesReport: %w", err)
	}

	soldInPeriod := make(map[string]int64, len(perf.Rows))
	for _, row := range perf.Rows {
		if row.SKUID != nil {
			soldInPeriod[*row.SKUID] = row.QuantitySold
		}
	}

	var report InventorySalesReport
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
			var turnover *int64
			if level.Total > 0 {
				bp := qtySold * 10000 / level.Total
				turnover = &bp
			}
			report.Rows = append(report.Rows, InventorySalesRow{
				SKUID: level.SKUID, SKUCode: sku.SKUCode, ProductName: productName,
				StockOnHand: level.Total, QuantitySoldInPeriod: qtySold, TurnoverBasisPoints: turnover,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getInventorySalesReport: %w", err)
	}
	return &report, nil
}
