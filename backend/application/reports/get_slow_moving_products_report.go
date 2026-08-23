package reports

import (
	"context"
	"fmt"
	"time"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const slowMovingStockLevelsPerPage = 500

// stockLevelLister narrows application/inventory.ListStockLevelsUseCase to
// the one method this report needs (ISP), and avoids depending on its
// concrete type.
type stockLevelLister interface {
	Execute(ctx context.Context, in invapp.ListStockLevelsInput) (*invapp.ListStockLevelsResult, error)
}

// variantLookupForSlowMoving/productLookupForSlowMoving narrow pms's
// Variant/ProductRepository to the one method needed to resolve a SKU's
// product name for display (ISP).
type variantLookupForSlowMoving interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Variant, error)
}
type productLookupForSlowMoving interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Product, error)
}

// SlowMovingProductRow is one SKU with stock on hand but no sales in the
// queried period -- the Phase 0 "Slow-Moving Products" report.
// QuantitySoldInPeriod is always 0 by construction (that's the row's
// inclusion criterion); kept on the row for symmetry with the other
// performance reports and so a future looser threshold doesn't need a
// shape change.
type SlowMovingProductRow struct {
	SKUID                string
	SKUCode              string
	ProductName          string
	StockOnHand          int64
	QuantitySoldInPeriod int64
}

// SlowMovingProductsReport lists every SKU with stock on hand (Total > 0)
// and zero units sold in [From, To). Capped to the first
// slowMovingStockLevelsPerPage stocked SKUs (by whatever order
// ListStockLevelsUseCase returns them in) -- a v1 limitation for very
// large catalogs, not a ranked/paginated report yet.
type SlowMovingProductsReport struct {
	Rows []SlowMovingProductRow
}

type GetSlowMovingProductsReportUseCase struct {
	db                 *gorm.DB
	stockLevels        stockLevelLister
	productPerformance productPerformanceReporter
	skuLookup          skuCostPriceLookup
	variantLookup      variantLookupForSlowMoving
	productLookup      productLookupForSlowMoving
}

func NewGetSlowMovingProductsReportUseCase(
	db *gorm.DB,
	stockLevels stockLevelLister,
	productPerformance productPerformanceReporter,
	skuLookup skuCostPriceLookup,
	variantLookup variantLookupForSlowMoving,
	productLookup productLookupForSlowMoving,
) *GetSlowMovingProductsReportUseCase {
	return &GetSlowMovingProductsReportUseCase{
		db: db, stockLevels: stockLevels, productPerformance: productPerformance,
		skuLookup: skuLookup, variantLookup: variantLookup, productLookup: productLookup,
	}
}

func (uc *GetSlowMovingProductsReportUseCase) Execute(ctx context.Context, from, to time.Time) (*SlowMovingProductsReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getSlowMovingProductsReport: %w", ErrToBeforeFrom)
	}

	// Each of these opens and closes its own transaction -- sequential,
	// not nested, since neither is in progress while the other runs.
	levels, err := uc.stockLevels.Execute(ctx, invapp.ListStockLevelsInput{Page: 1, PerPage: slowMovingStockLevelsPerPage})
	if err != nil {
		return nil, fmt.Errorf("getSlowMovingProductsReport: %w", err)
	}
	perf, err := uc.productPerformance.Execute(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("getSlowMovingProductsReport: %w", err)
	}

	soldInPeriod := make(map[string]int64, len(perf.Rows))
	for _, row := range perf.Rows {
		if row.SKUID != nil {
			soldInPeriod[*row.SKUID] = row.QuantitySold
		}
	}

	var report SlowMovingProductsReport
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		for _, level := range levels.Levels {
			if level.Total <= 0 || soldInPeriod[level.SKUID] > 0 {
				continue
			}
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
			report.Rows = append(report.Rows, SlowMovingProductRow{
				SKUID: level.SKUID, SKUCode: sku.SKUCode, ProductName: productName,
				StockOnHand: level.Total, QuantitySoldInPeriod: soldInPeriod[level.SKUID],
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getSlowMovingProductsReport: %w", err)
	}
	return &report, nil
}
