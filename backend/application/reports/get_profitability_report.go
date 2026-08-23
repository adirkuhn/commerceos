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

// ProfitabilityReport is the Phase 0 "Profitability Report": revenue,
// COGS, profit, and margin for [From, To).
type ProfitabilityReport struct {
	CurrencyCode string
	Summary      omsDomain.ProfitabilitySummary
}

type GetProfitabilityReportUseCase struct {
	db            *gorm.DB
	profitability omsDomain.ProfitabilityReportRepository
	settingsRepo  settingsDomain.Repository
}

func NewGetProfitabilityReportUseCase(db *gorm.DB, profitability omsDomain.ProfitabilityReportRepository, settingsRepo settingsDomain.Repository) *GetProfitabilityReportUseCase {
	return &GetProfitabilityReportUseCase{db: db, profitability: profitability, settingsRepo: settingsRepo}
}

func (uc *GetProfitabilityReportUseCase) Execute(ctx context.Context, from, to time.Time) (*ProfitabilityReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getProfitabilityReport: %w", ErrToBeforeFrom)
	}

	var report ProfitabilityReport
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

		report.Summary, err = uc.profitability.ProfitabilityBetween(ctx, from, to, currency)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getProfitabilityReport: %w", err)
	}
	return &report, nil
}
