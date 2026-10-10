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

// ErrToBeforeFrom guards GetSalesSummaryReportUseCase.Execute's date range.
var ErrToBeforeFrom = errs.New("REPORT_END_BEFORE_START", "report end date must be after the start date")

// SalesSummaryReport compares an arbitrary [From, To) period against the
// immediately preceding period of equal length (e.g. "this month" queried
// against "last month", or "last 7 days" against the 7 days before that) --
// the Phase 0 roadmap's "Revenue +18.4%, Orders +12.1%..." comparison view.
// Each *ChangePercent is nil when the previous period's corresponding
// figure is zero (no meaningful percent change off a zero baseline).
type SalesSummaryReport struct {
	CurrencyCode            string
	Current                 omsDomain.SalesSummary
	Previous                omsDomain.SalesSummary
	RevenueChangePercent    *float64
	OrderCountChangePercent *float64
	UnitsSoldChangePercent  *float64
	AOVChangePercent        *float64
}

type GetSalesSummaryReportUseCase struct {
	db           *gorm.DB
	salesRepo    omsDomain.SalesReportRepository
	settingsRepo settingsDomain.Repository
}

func NewGetSalesSummaryReportUseCase(db *gorm.DB, salesRepo omsDomain.SalesReportRepository, settingsRepo settingsDomain.Repository) *GetSalesSummaryReportUseCase {
	return &GetSalesSummaryReportUseCase{db: db, salesRepo: salesRepo, settingsRepo: settingsRepo}
}

func (uc *GetSalesSummaryReportUseCase) Execute(ctx context.Context, from, to time.Time) (*SalesSummaryReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getSalesSummaryReport: %w", ErrToBeforeFrom)
	}
	duration := to.Sub(from)
	prevFrom := from.Add(-duration)
	prevTo := from

	var report SalesSummaryReport
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

		report.Current, err = uc.salesRepo.SalesSummaryBetween(ctx, from, to, currency)
		if err != nil {
			return err
		}
		report.Previous, err = uc.salesRepo.SalesSummaryBetween(ctx, prevFrom, prevTo, currency)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getSalesSummaryReport: %w", err)
	}

	report.RevenueChangePercent = percentChange(report.Previous.RevenueMinorUnits, report.Current.RevenueMinorUnits)
	report.OrderCountChangePercent = percentChange(report.Previous.OrderCount, report.Current.OrderCount)
	report.UnitsSoldChangePercent = percentChange(report.Previous.UnitsSold, report.Current.UnitsSold)
	report.AOVChangePercent = percentChange(report.Previous.AverageOrderValueMinorUnits, report.Current.AverageOrderValueMinorUnits)
	return &report, nil
}

// percentChange returns nil when previous is 0 -- no meaningful percent
// change off a zero baseline (0 -> N is an undefined/infinite increase,
// not "0%").
func percentChange(previous, current int64) *float64 {
	if previous == 0 {
		return nil
	}
	pct := float64(current-previous) / float64(previous) * 100
	return &pct
}
