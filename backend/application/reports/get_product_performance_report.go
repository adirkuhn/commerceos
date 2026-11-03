package reports

import (
	"context"
	"errors"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// skuCostPriceLookup narrows pms.SKURepository to the one method this
// report needs (ISP) -- defined here at point of use rather than importing
// the whole domain/pms.SKURepository interface, same pattern as
// application/tax's skuTaxClassChecker / application/promotions'
// skuExistenceChecker.
type skuCostPriceLookup interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.SKU, error)
}

// ProductPerformanceRow is one SKU's sales figures plus profit/margin, when
// a cost price is known for it. CostPriceMinorUnits/ProfitMinorUnits/
// MarginBasisPoints are all nil together when: the line item has no linked
// SKU (adhoc/unmatched), the SKU has since been deleted, or the SKU's
// CostPriceMinorUnits has never been computed (see pms.SKU's own doc) --
// there's no cost basis to join against revenue that actually happened.
type ProductPerformanceRow struct {
	omsDomain.ProductPerformance
	CostPriceMinorUnits *int64
	ProfitMinorUnits    *int64
	MarginBasisPoints   *int64
}

// ProductPerformanceReport is the Phase 0 "Product Performance" report:
// every SKU sold in [From, To), ranked by revenue.
type ProductPerformanceReport struct {
	CurrencyCode string
	Rows         []ProductPerformanceRow
}

type GetProductPerformanceReportUseCase struct {
	db           *gorm.DB
	perfRepo     omsDomain.ProductPerformanceRepository
	settingsRepo settingsDomain.Repository
	skuLookup    skuCostPriceLookup
}

func NewGetProductPerformanceReportUseCase(db *gorm.DB, perfRepo omsDomain.ProductPerformanceRepository, settingsRepo settingsDomain.Repository, skuLookup skuCostPriceLookup) *GetProductPerformanceReportUseCase {
	return &GetProductPerformanceReportUseCase{db: db, perfRepo: perfRepo, settingsRepo: settingsRepo, skuLookup: skuLookup}
}

func (uc *GetProductPerformanceReportUseCase) Execute(ctx context.Context, from, to time.Time) (*ProductPerformanceReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getProductPerformanceReport: %w", ErrToBeforeFrom)
	}

	var report ProductPerformanceReport
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		settings, err := uc.settingsRepo.Get(ctx)
		if err != nil {
			return err
		}
		currency := settings.DefaultCurrency
		if currency == "" {
			currency = "USD"
		}
		report.CurrencyCode = currency

		perf, err := uc.perfRepo.ProductPerformanceBetween(ctx, from, to, currency)
		if err != nil {
			return err
		}
		report.Rows = make([]ProductPerformanceRow, len(perf))
		for i, p := range perf {
			row := ProductPerformanceRow{ProductPerformance: p}
			if p.SKUID != nil {
				sku, err := uc.skuLookup.FindByID(ctx, *p.SKUID)
				if err != nil && !errors.Is(err, pmsDomain.ErrSKUNotFound) {
					return err
				}
				if actual := actualCost(sku); actual != nil {
					row.CostPriceMinorUnits = actual
					profit := p.RevenueMinorUnits - *actual*p.QuantitySold
					row.ProfitMinorUnits = &profit
					if p.RevenueMinorUnits != 0 {
						margin := profit * 10000 / p.RevenueMinorUnits
						row.MarginBasisPoints = &margin
					}
				}
			}
			report.Rows[i] = row
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getProductPerformanceReport: %w", err)
	}
	return &report, nil
}

// actualCost is what a unit of sku actually cost (pms.SKU.ActualCostMinorUnits);
// nil for a missing SKU.
func actualCost(sku *pmsDomain.SKU) *int64 {
	if sku == nil {
		return nil
	}
	return sku.ActualCostMinorUnits()
}
