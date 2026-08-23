package reports

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ProductPerformanceReport is the Phase 0 "Product Performance" report:
// every SKU sold in [From, To), ranked by revenue.
type ProductPerformanceReport struct {
	CurrencyCode string
	Rows         []omsDomain.ProductPerformance
}

type GetProductPerformanceReportUseCase struct {
	db           *gorm.DB
	perfRepo     omsDomain.ProductPerformanceRepository
	settingsRepo settingsDomain.Repository
}

func NewGetProductPerformanceReportUseCase(db *gorm.DB, perfRepo omsDomain.ProductPerformanceRepository, settingsRepo settingsDomain.Repository) *GetProductPerformanceReportUseCase {
	return &GetProductPerformanceReportUseCase{db: db, perfRepo: perfRepo, settingsRepo: settingsRepo}
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

		report.Rows, err = uc.perfRepo.ProductPerformanceBetween(ctx, from, to, currency)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getProductPerformanceReport: %w", err)
	}
	return &report, nil
}
