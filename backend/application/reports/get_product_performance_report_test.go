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

type stubProductPerformanceRepo struct {
	out []omsDomain.ProductPerformance
	err error
}

func (s *stubProductPerformanceRepo) ProductPerformanceBetween(_ context.Context, _, _ time.Time, _ string) ([]omsDomain.ProductPerformance, error) {
	return s.out, s.err
}

func TestGetProductPerformanceReport_returnsRepoResultInCurrency(t *testing.T) {
	repo := &stubProductPerformanceRepo{out: []omsDomain.ProductPerformance{
		{SKUCode: "SKU-A", ProductName: "Acai", QuantitySold: 5, RevenueMinorUnits: 2500, OrderCount: 2},
	}}
	uc := appreports.NewGetProductPerformanceReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{settings: &settingsDomain.TenantSettings{DefaultCurrency: "EUR"}})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Equal(t, "EUR", result.CurrencyCode)
	require.Len(t, result.Rows, 1)
	require.Equal(t, "SKU-A", result.Rows[0].SKUCode)
}

func TestGetProductPerformanceReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetProductPerformanceReportUseCase(newMockGormDB(t), &stubProductPerformanceRepo{}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour))

	require.Error(t, err)
}

func TestGetProductPerformanceReport_propagatesRepoError(t *testing.T) {
	uc := appreports.NewGetProductPerformanceReportUseCase(newMockGormDB(t), &stubProductPerformanceRepo{err: context.DeadlineExceeded}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.Error(t, err)
}
