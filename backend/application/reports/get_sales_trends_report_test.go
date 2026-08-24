package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type stubSalesTrendReportRepo struct {
	points         []omsDomain.SalesTrendPoint
	weekdays       []omsDomain.WeekdaySales
	trendErr       error
	weekdayErr     error
	gotGranularity omsDomain.SalesTrendGranularity
}

func (s *stubSalesTrendReportRepo) SalesTrendBetween(_ context.Context, _, _ time.Time, _ string, granularity omsDomain.SalesTrendGranularity) ([]omsDomain.SalesTrendPoint, error) {
	s.gotGranularity = granularity
	if s.trendErr != nil {
		return nil, s.trendErr
	}
	return s.points, nil
}

func (s *stubSalesTrendReportRepo) SalesByWeekdayBetween(_ context.Context, _, _ time.Time, _ string) ([]omsDomain.WeekdaySales, error) {
	if s.weekdayErr != nil {
		return nil, s.weekdayErr
	}
	return s.weekdays, nil
}

func TestGetSalesTrendsReport_returnsPointsAndWeekdaysInCurrency(t *testing.T) {
	repo := &stubSalesTrendReportRepo{
		points:   []omsDomain.SalesTrendPoint{{RevenueMinorUnits: 1000, OrderCount: 2, UnitsSold: 5}},
		weekdays: []omsDomain.WeekdaySales{{Weekday: time.Monday, RevenueMinorUnits: 1000, OrderCount: 2}},
	}
	uc := appreports.NewGetSalesTrendsReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), omsDomain.SalesTrendGranularityDay)

	require.NoError(t, err)
	require.Len(t, result.Points, 1)
	require.Equal(t, int64(1000), result.Points[0].RevenueMinorUnits)
	require.Len(t, result.ByWeekday, 1)
	require.Equal(t, time.Monday, result.ByWeekday[0].Weekday)
	require.Equal(t, omsDomain.SalesTrendGranularityDay, repo.gotGranularity)
}

func TestGetSalesTrendsReport_defaultsGranularityToDayWhenEmpty(t *testing.T) {
	repo := &stubSalesTrendReportRepo{}
	uc := appreports.NewGetSalesTrendsReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), "")

	require.NoError(t, err)
	require.Equal(t, omsDomain.SalesTrendGranularityDay, repo.gotGranularity)
}

func TestGetSalesTrendsReport_rejectsInvalidGranularity(t *testing.T) {
	uc := appreports.NewGetSalesTrendsReportUseCase(newMockGormDB(t), &stubSalesTrendReportRepo{}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), omsDomain.SalesTrendGranularity("fortnight"))

	require.Error(t, err)
}

func TestGetSalesTrendsReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetSalesTrendsReportUseCase(newMockGormDB(t), &stubSalesTrendReportRepo{}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour), omsDomain.SalesTrendGranularityDay)

	require.Error(t, err)
}

func TestGetSalesTrendsReport_propagatesRepoError(t *testing.T) {
	uc := appreports.NewGetSalesTrendsReportUseCase(newMockGormDB(t), &stubSalesTrendReportRepo{trendErr: context.DeadlineExceeded}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), omsDomain.SalesTrendGranularityDay)

	require.Error(t, err)
}
