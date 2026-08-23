package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

type stubCustomerReportRepo struct {
	out []omsDomain.CustomerPerformance
	err error
}

func (s *stubCustomerReportRepo) CustomerPerformanceBetween(_ context.Context, _, _ time.Time, _ string) ([]omsDomain.CustomerPerformance, error) {
	return s.out, s.err
}

func TestGetCustomerPerformanceReport_returnsRepoResultInCurrency(t *testing.T) {
	repo := &stubCustomerReportRepo{out: []omsDomain.CustomerPerformance{
		{CustomerID: "cust-1", CustomerName: "Jane Doe", OrderCount: 2, RevenueMinorUnits: 1500, AverageOrderValueMinorUnits: 750},
	}}
	uc := appreports.NewGetCustomerPerformanceReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{settings: &settingsDomain.TenantSettings{DefaultCurrency: "EUR"}})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Equal(t, "EUR", result.CurrencyCode)
	require.Len(t, result.Rows, 1)
	require.Equal(t, "Jane Doe", result.Rows[0].CustomerName)
}

func TestGetCustomerPerformanceReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetCustomerPerformanceReportUseCase(newMockGormDB(t), &stubCustomerReportRepo{}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour))

	require.Error(t, err)
}

func TestGetCustomerPerformanceReport_propagatesRepoError(t *testing.T) {
	uc := appreports.NewGetCustomerPerformanceReportUseCase(newMockGormDB(t), &stubCustomerReportRepo{err: context.DeadlineExceeded}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.Error(t, err)
}
