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

// CustomerPerformanceReport is the Phase 0 "Customer Reports" item: every
// customer with a qualifying order in [From, To), ranked by revenue. Guest
// orders (no linked customer) are excluded -- see
// omsDomain.CustomerReportRepository's doc.
type CustomerPerformanceReport struct {
	CurrencyCode string
	Rows         []omsDomain.CustomerPerformance
}

type GetCustomerPerformanceReportUseCase struct {
	db           *gorm.DB
	customerRepo omsDomain.CustomerReportRepository
	settingsRepo settingsDomain.Repository
}

func NewGetCustomerPerformanceReportUseCase(db *gorm.DB, customerRepo omsDomain.CustomerReportRepository, settingsRepo settingsDomain.Repository) *GetCustomerPerformanceReportUseCase {
	return &GetCustomerPerformanceReportUseCase{db: db, customerRepo: customerRepo, settingsRepo: settingsRepo}
}

func (uc *GetCustomerPerformanceReportUseCase) Execute(ctx context.Context, from, to time.Time) (*CustomerPerformanceReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getCustomerPerformanceReport: %w", ErrToBeforeFrom)
	}

	var report CustomerPerformanceReport
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

		report.Rows, err = uc.customerRepo.CustomerPerformanceBetween(ctx, from, to, currency)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getCustomerPerformanceReport: %w", err)
	}
	return &report, nil
}
