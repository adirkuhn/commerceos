package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type stubProfitabilityTrendRepo struct {
	points         []omsDomain.ProfitabilityPoint
	err            error
	gotGranularity omsDomain.SalesTrendGranularity
}

func (s *stubProfitabilityTrendRepo) ProfitabilityTrendBetween(_ context.Context, _, _ time.Time, _ string, granularity omsDomain.SalesTrendGranularity) ([]omsDomain.ProfitabilityPoint, error) {
	s.gotGranularity = granularity
	if s.err != nil {
		return nil, s.err
	}
	return s.points, nil
}

func TestGetProfitabilityTrendReport_returnsPointsInCurrency(t *testing.T) {
	repo := &stubProfitabilityTrendRepo{points: []omsDomain.ProfitabilityPoint{{RevenueMinorUnits: 1500, ProfitMinorUnits: 400}}}
	uc := appreports.NewGetProfitabilityTrendReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), omsDomain.SalesTrendGranularityDay)

	require.NoError(t, err)
	require.Len(t, result.Points, 1)
	require.Equal(t, int64(400), result.Points[0].ProfitMinorUnits)
	require.Equal(t, omsDomain.SalesTrendGranularityDay, repo.gotGranularity)
}

func TestGetProfitabilityTrendReport_defaultsGranularityToDayWhenEmpty(t *testing.T) {
	repo := &stubProfitabilityTrendRepo{}
	uc := appreports.NewGetProfitabilityTrendReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), "")

	require.NoError(t, err)
	require.Equal(t, omsDomain.SalesTrendGranularityDay, repo.gotGranularity)
}

func TestGetProfitabilityTrendReport_rejectsInvalidGranularity(t *testing.T) {
	uc := appreports.NewGetProfitabilityTrendReportUseCase(newMockGormDB(t), &stubProfitabilityTrendRepo{}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), omsDomain.SalesTrendGranularity("fortnight"))

	require.Error(t, err)
}

func TestGetProfitabilityTrendReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetProfitabilityTrendReportUseCase(newMockGormDB(t), &stubProfitabilityTrendRepo{}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour), omsDomain.SalesTrendGranularityDay)

	require.Error(t, err)
}

func TestGetProfitabilityTrendReport_propagatesRepoError(t *testing.T) {
	uc := appreports.NewGetProfitabilityTrendReportUseCase(newMockGormDB(t), &stubProfitabilityTrendRepo{err: context.DeadlineExceeded}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), omsDomain.SalesTrendGranularityDay)

	require.Error(t, err)
}
