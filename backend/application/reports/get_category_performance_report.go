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

// CategoryPerformanceReport is the Phase 0 "Category Analysis" report:
// every category (plus one "Uncategorized" bucket when applicable) for
// line items sold in [From, To), ranked by revenue.
type CategoryPerformanceReport struct {
	CurrencyCode string
	Rows         []omsDomain.CategoryPerformance
}

type GetCategoryPerformanceReportUseCase struct {
	db           *gorm.DB
	categoryRepo omsDomain.CategoryReportRepository
	settingsRepo settingsDomain.Repository
}

func NewGetCategoryPerformanceReportUseCase(db *gorm.DB, categoryRepo omsDomain.CategoryReportRepository, settingsRepo settingsDomain.Repository) *GetCategoryPerformanceReportUseCase {
	return &GetCategoryPerformanceReportUseCase{db: db, categoryRepo: categoryRepo, settingsRepo: settingsRepo}
}

func (uc *GetCategoryPerformanceReportUseCase) Execute(ctx context.Context, from, to time.Time) (*CategoryPerformanceReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getCategoryPerformanceReport: %w", ErrToBeforeFrom)
	}

	var report CategoryPerformanceReport
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

		report.Rows, err = uc.categoryRepo.CategoryPerformanceBetween(ctx, from, to, currency)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getCategoryPerformanceReport: %w", err)
	}
	return &report, nil
}
