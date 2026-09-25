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

// CohortRetentionCell is one (cohort, months-since-acquisition) cell,
// enriched with a retention rate relative to its cohort's own size.
type CohortRetentionCell struct {
	MonthsSinceCohort int64
	RetainedCustomers int64
	RevenueMinorUnits int64
	// RetainedCustomers / the cohort's CohortSize, in basis points (1bp =
	// 0.01%). 0 when CohortSize is 0.
	RetentionRateBasisPoints int64
}

// CustomerCohortRow is one acquisition cohort (all customers whose
// all-time first order fell in CohortMonth) plus its retention across
// however many subsequent months the queried period covers.
type CustomerCohortRow struct {
	CohortMonth time.Time
	CohortSize  int64
	Retention   []CohortRetentionCell
}

// CustomerCohortsReport is the Phase 0 "Customer Cohorts" report.
type CustomerCohortsReport struct {
	CurrencyCode string
	Cohorts      []CustomerCohortRow
}

type GetCustomerCohortsReportUseCase struct {
	db           *gorm.DB
	cohortRepo   omsDomain.CustomerCohortRepository
	settingsRepo settingsDomain.Repository
}

func NewGetCustomerCohortsReportUseCase(db *gorm.DB, cohortRepo omsDomain.CustomerCohortRepository, settingsRepo settingsDomain.Repository) *GetCustomerCohortsReportUseCase {
	return &GetCustomerCohortsReportUseCase{db: db, cohortRepo: cohortRepo, settingsRepo: settingsRepo}
}

func (uc *GetCustomerCohortsReportUseCase) Execute(ctx context.Context, from, to time.Time) (*CustomerCohortsReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getCustomerCohortsReport: %w", ErrToBeforeFrom)
	}

	var report CustomerCohortsReport
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

		sizes, err := uc.cohortRepo.CohortSizesBetween(ctx, from, to, currency, settings.ReportingTimezone())
		if err != nil {
			return err
		}
		retention, err := uc.cohortRepo.CohortRetentionBetween(ctx, from, to, currency, settings.ReportingTimezone())
		if err != nil {
			return err
		}

		byMonth := make(map[int64]*CustomerCohortRow, len(sizes))
		var order []int64
		for _, size := range sizes {
			key := size.CohortMonth.Unix()
			byMonth[key] = &CustomerCohortRow{CohortMonth: size.CohortMonth, CohortSize: size.CustomerCount}
			order = append(order, key)
		}
		for _, point := range retention {
			row, ok := byMonth[point.CohortMonth.Unix()]
			if !ok {
				continue
			}
			var rate int64
			if row.CohortSize > 0 {
				rate = point.RetainedCustomers * 10000 / row.CohortSize
			}
			row.Retention = append(row.Retention, CohortRetentionCell{
				MonthsSinceCohort: point.MonthsSinceCohort, RetainedCustomers: point.RetainedCustomers,
				RevenueMinorUnits: point.RevenueMinorUnits, RetentionRateBasisPoints: rate,
			})
		}
		for _, key := range order {
			report.Cohorts = append(report.Cohorts, *byMonth[key])
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getCustomerCohortsReport: %w", err)
	}
	return &report, nil
}
