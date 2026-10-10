package scheduler_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appscheduler "github.com/adirkuhn/commerceos/application/scheduler"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubExpiryAlertLister struct {
	settings []settingsDomain.TenantSettings
	err      error
	sawAdmin bool
}

func (s *stubExpiryAlertLister) ListExpiryAlertEnabled(ctx context.Context) ([]settingsDomain.TenantSettings, error) {
	s.sawAdmin = tenant.IsSuperAdmin(ctx)
	return s.settings, s.err
}

type reportCall struct {
	ctxTenant string
	ctxAdmin  bool
	settings  string
}

type stubStockExpiryReporter struct {
	calls  []reportCall
	failOn string
}

func (s *stubStockExpiryReporter) Execute(ctx context.Context, ts settingsDomain.TenantSettings) error {
	s.calls = append(s.calls, reportCall{tenant.IDFromContext(ctx), tenant.IsSuperAdmin(ctx), ts.TenantID})
	if ts.TenantID == s.failOn {
		return errors.New("report failed")
	}
	return nil
}

func TestRunStockExpiryReport_reportsEachEnabledTenantInItsOwnScope(t *testing.T) {
	lister := &stubExpiryAlertLister{settings: []settingsDomain.TenantSettings{{TenantID: "t-1"}, {TenantID: "t-2"}}}
	reporter := &stubStockExpiryReporter{}
	uc := appscheduler.NewRunStockExpiryReportUseCase(newMockGormDB(t), lister, reporter)

	require.NoError(t, uc.Run(tenant.WithSuperAdmin(context.Background())), "run-now and the poller pass a super-admin context")

	assert.True(t, lister.sawAdmin, "listing crosses tenants")
	assert.Equal(t, []reportCall{{"t-1", false, "t-1"}, {"t-2", false, "t-2"}}, reporter.calls,
		"each report runs as that tenant, never with RLS bypassed")
}

func TestRunStockExpiryReport_oneTenantFailingDoesNotStopTheOthers(t *testing.T) {
	lister := &stubExpiryAlertLister{settings: []settingsDomain.TenantSettings{{TenantID: "t-1"}, {TenantID: "t-2"}}}
	reporter := &stubStockExpiryReporter{failOn: "t-1"}
	uc := appscheduler.NewRunStockExpiryReportUseCase(newMockGormDB(t), lister, reporter)

	err := uc.Run(context.Background())

	assert.ErrorContains(t, err, "t-1")
	assert.Len(t, reporter.calls, 2)
}

func TestRunStockExpiryReport_listErrorFails(t *testing.T) {
	boom := errors.New("db down")
	uc := appscheduler.NewRunStockExpiryReportUseCase(newMockGormDB(t), &stubExpiryAlertLister{err: boom}, &stubStockExpiryReporter{})

	assert.ErrorIs(t, uc.Run(context.Background()), boom)
}
