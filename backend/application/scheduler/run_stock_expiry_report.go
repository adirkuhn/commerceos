package scheduler

import (
	"context"
	"errors"
	"fmt"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// stockExpiryReporter is application/inventory's
// ReportStockCloseToExpiryUseCase narrowed to point of use.
type stockExpiryReporter interface {
	Execute(ctx context.Context, s settingsDomain.TenantSettings) error
}

// RunStockExpiryReportUseCase is the stock_expiry_report scheduled job: the
// daily stock-close-to-expiry report for every tenant that turned it on.
// Tenants are listed across RLS, then each report runs scoped to its own
// tenant. A failing tenant is reported in the job's error without stopping
// the others.
type RunStockExpiryReportUseCase struct {
	db       *gorm.DB
	lister   settingsDomain.ExpiryAlertTenantLister
	reporter stockExpiryReporter
}

func NewRunStockExpiryReportUseCase(db *gorm.DB, lister settingsDomain.ExpiryAlertTenantLister, reporter stockExpiryReporter) *RunStockExpiryReportUseCase {
	return &RunStockExpiryReportUseCase{db: db, lister: lister, reporter: reporter}
}

func (uc *RunStockExpiryReportUseCase) Run(ctx context.Context) error {
	var enabled []settingsDomain.TenantSettings
	err := tenant.WithTenant(tenant.WithSuperAdmin(ctx), uc.db, func(ctx context.Context) error {
		var err error
		enabled, err = uc.lister.ListExpiryAlertEnabled(ctx)
		return err
	})
	if err != nil {
		return fmt.Errorf("runStockExpiryReport: %w", err)
	}

	var failures []error
	for _, s := range enabled {
		// AsTenant, not the super-admin context run-now and the poller pass: the report must
		// read only this tenant's stock.
		if err := uc.reporter.Execute(tenant.AsTenant(ctx, s.TenantID), s); err != nil {
			failures = append(failures, fmt.Errorf("tenant %s: %w", s.TenantID, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("runStockExpiryReport: %w", errors.Join(failures...))
	}
	return nil
}
