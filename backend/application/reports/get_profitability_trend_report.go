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

// ProfitabilityTrendReport is the Profitability report's daily/weekly/
// monthly view -- additive to ProfitabilityReport (GetProfitabilityReportUseCase's
// whole-period summary), a separate use case so that use case and its
// callers (business insights) stay untouched.
type ProfitabilityTrendReport struct {
	CurrencyCode string
	Granularity  omsDomain.SalesTrendGranularity
	Points       []omsDomain.ProfitabilityPoint
}

type GetProfitabilityTrendReportUseCase struct {
	db                 *gorm.DB
	profitabilityTrend omsDomain.ProfitabilityTrendRepository
	settingsRepo       settingsDomain.Repository
}

func NewGetProfitabilityTrendReportUseCase(db *gorm.DB, profitabilityTrend omsDomain.ProfitabilityTrendRepository, settingsRepo settingsDomain.Repository) *GetProfitabilityTrendReportUseCase {
	return &GetProfitabilityTrendReportUseCase{db: db, profitabilityTrend: profitabilityTrend, settingsRepo: settingsRepo}
}

func (uc *GetProfitabilityTrendReportUseCase) Execute(ctx context.Context, from, to time.Time, granularity omsDomain.SalesTrendGranularity) (*ProfitabilityTrendReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getProfitabilityTrendReport: %w", ErrToBeforeFrom)
	}
	if granularity == "" {
		granularity = omsDomain.SalesTrendGranularityDay
	}
	if !granularity.Valid() {
		return nil, fmt.Errorf("getProfitabilityTrendReport: %w", ErrInvalidGranularity)
	}

	var report ProfitabilityTrendReport
	report.Granularity = granularity
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

		report.Points, err = uc.profitabilityTrend.ProfitabilityTrendBetween(ctx, from, to, currency, granularity)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getProfitabilityTrendReport: %w", err)
	}
	return &report, nil
}
