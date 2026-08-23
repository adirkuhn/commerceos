package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type stubCategoryReportRepo struct {
	out []omsDomain.CategoryPerformance
	err error
}

func (s *stubCategoryReportRepo) CategoryPerformanceBetween(_ context.Context, _, _ time.Time, _ string) ([]omsDomain.CategoryPerformance, error) {
	return s.out, s.err
}

func TestGetCategoryPerformanceReport_returnsRepoResultInCurrency(t *testing.T) {
	repo := &stubCategoryReportRepo{out: []omsDomain.CategoryPerformance{
		{CategoryName: "Beverages", QuantitySold: 2, RevenueMinorUnits: 1000, OrderCount: 1},
	}}
	uc := appreports.NewGetCategoryPerformanceReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Len(t, result.Rows, 1)
	require.Equal(t, "Beverages", result.Rows[0].CategoryName)
}

func TestGetCategoryPerformanceReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetCategoryPerformanceReportUseCase(newMockGormDB(t), &stubCategoryReportRepo{}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour))

	require.Error(t, err)
}

func TestGetCategoryPerformanceReport_propagatesRepoError(t *testing.T) {
	uc := appreports.NewGetCategoryPerformanceReportUseCase(newMockGormDB(t), &stubCategoryReportRepo{err: context.DeadlineExceeded}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.Error(t, err)
}
