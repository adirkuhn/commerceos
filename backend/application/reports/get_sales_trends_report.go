package reports

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/domain/errs"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ErrInvalidGranularity guards GetSalesTrendsReportUseCase.Execute's
// granularity argument.
var ErrInvalidGranularity = errs.New("REPORT_INVALID_GRANULARITY", "granularity must be one of day, week, month")

// SalesTrendsReport is the Phase 0 "Sales Trends & Seasonality" report --
// a bucketed revenue/orders/units time series (Points) plus a breakdown of
// revenue/orders summed by weekday across the whole period (ByWeekday),
// the "seasonality" half.
type SalesTrendsReport struct {
	CurrencyCode string
	Granularity  omsDomain.SalesTrendGranularity
	Points       []omsDomain.SalesTrendPoint
	ByWeekday    []omsDomain.WeekdaySales
}

type GetSalesTrendsReportUseCase struct {
	db           *gorm.DB
	salesTrends  omsDomain.SalesTrendRepository
	settingsRepo settingsDomain.Repository
}

func NewGetSalesTrendsReportUseCase(db *gorm.DB, salesTrends omsDomain.SalesTrendRepository, settingsRepo settingsDomain.Repository) *GetSalesTrendsReportUseCase {
	return &GetSalesTrendsReportUseCase{db: db, salesTrends: salesTrends, settingsRepo: settingsRepo}
}

func (uc *GetSalesTrendsReportUseCase) Execute(ctx context.Context, from, to time.Time, granularity omsDomain.SalesTrendGranularity) (*SalesTrendsReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getSalesTrendsReport: %w", ErrToBeforeFrom)
	}
	if granularity == "" {
		granularity = omsDomain.SalesTrendGranularityDay
	}
	if !granularity.Valid() {
		return nil, fmt.Errorf("getSalesTrendsReport: %w", ErrInvalidGranularity)
	}

	var report SalesTrendsReport
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

		report.Points, err = uc.salesTrends.SalesTrendBetween(ctx, from, to, currency, granularity, settings.ReportingTimezone())
		if err != nil {
			return err
		}
		report.ByWeekday, err = uc.salesTrends.SalesByWeekdayBetween(ctx, from, to, currency, settings.ReportingTimezone())
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getSalesTrendsReport: %w", err)
	}
	return &report, nil
}
