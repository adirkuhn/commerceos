package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type stubProfitabilityReportRepo struct {
	out omsDomain.ProfitabilitySummary
	err error
}

func (s *stubProfitabilityReportRepo) ProfitabilityBetween(_ context.Context, _, _ time.Time, _ string) (omsDomain.ProfitabilitySummary, error) {
	return s.out, s.err
}

func TestGetProfitabilityReport_returnsRepoResultInCurrency(t *testing.T) {
	repo := &stubProfitabilityReportRepo{out: omsDomain.ProfitabilitySummary{
		RevenueMinorUnits: 1500, CostKnownRevenueMinorUnits: 1000, CostMinorUnits: 600, ProfitMinorUnits: 400, MarginBasisPoints: 4000,
	}}
	uc := appreports.NewGetProfitabilityReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Equal(t, int64(1500), result.Summary.RevenueMinorUnits)
	require.Equal(t, int64(400), result.Summary.ProfitMinorUnits)
}

func TestGetProfitabilityReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetProfitabilityReportUseCase(newMockGormDB(t), &stubProfitabilityReportRepo{}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour))

	require.Error(t, err)
}

func TestGetProfitabilityReport_propagatesRepoError(t *testing.T) {
	uc := appreports.NewGetProfitabilityReportUseCase(newMockGormDB(t), &stubProfitabilityReportRepo{err: context.DeadlineExceeded}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.Error(t, err)
}
