package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type stubChannelRevenueReportRepo struct {
	channels       []omsDomain.ChannelRevenue
	points         []omsDomain.ChannelRevenuePoint
	channelsErr    error
	pointsErr      error
	gotGranularity omsDomain.SalesTrendGranularity
}

func (s *stubChannelRevenueReportRepo) RevenueByChannelBetween(_ context.Context, _, _ time.Time, _ string) ([]omsDomain.ChannelRevenue, error) {
	if s.channelsErr != nil {
		return nil, s.channelsErr
	}
	return s.channels, nil
}

func (s *stubChannelRevenueReportRepo) RevenueByChannelTrendBetween(_ context.Context, _, _ time.Time, _ string, granularity omsDomain.SalesTrendGranularity) ([]omsDomain.ChannelRevenuePoint, error) {
	s.gotGranularity = granularity
	if s.pointsErr != nil {
		return nil, s.pointsErr
	}
	return s.points, nil
}

func TestGetRevenueByChannelReport_returnsChannelsAndPointsInCurrency(t *testing.T) {
	repo := &stubChannelRevenueReportRepo{
		channels: []omsDomain.ChannelRevenue{{Source: omsDomain.SalesOrderSourcePOS, TotalMinorUnits: 1000}},
		points:   []omsDomain.ChannelRevenuePoint{{Source: omsDomain.SalesOrderSourcePOS, RevenueMinorUnits: 500}},
	}
	uc := appreports.NewGetRevenueByChannelReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), omsDomain.SalesTrendGranularityDay)

	require.NoError(t, err)
	require.Len(t, result.Channels, 1)
	require.Equal(t, int64(1000), result.Channels[0].TotalMinorUnits)
	require.Len(t, result.Points, 1)
	require.Equal(t, int64(500), result.Points[0].RevenueMinorUnits)
	require.Equal(t, omsDomain.SalesTrendGranularityDay, repo.gotGranularity)
}

func TestGetRevenueByChannelReport_defaultsGranularityToDayWhenEmpty(t *testing.T) {
	repo := &stubChannelRevenueReportRepo{}
	uc := appreports.NewGetRevenueByChannelReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), "")

	require.NoError(t, err)
	require.Equal(t, omsDomain.SalesTrendGranularityDay, repo.gotGranularity)
}

func TestGetRevenueByChannelReport_rejectsInvalidGranularity(t *testing.T) {
	uc := appreports.NewGetRevenueByChannelReportUseCase(newMockGormDB(t), &stubChannelRevenueReportRepo{}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), omsDomain.SalesTrendGranularity("fortnight"))

	require.Error(t, err)
}

func TestGetRevenueByChannelReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetRevenueByChannelReportUseCase(newMockGormDB(t), &stubChannelRevenueReportRepo{}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour), omsDomain.SalesTrendGranularityDay)

	require.Error(t, err)
}

func TestGetRevenueByChannelReport_propagatesChannelsRepoError(t *testing.T) {
	uc := appreports.NewGetRevenueByChannelReportUseCase(newMockGormDB(t), &stubChannelRevenueReportRepo{channelsErr: context.DeadlineExceeded}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), omsDomain.SalesTrendGranularityDay)

	require.Error(t, err)
}

func TestGetRevenueByChannelReport_propagatesPointsRepoError(t *testing.T) {
	uc := appreports.NewGetRevenueByChannelReportUseCase(newMockGormDB(t), &stubChannelRevenueReportRepo{pointsErr: context.DeadlineExceeded}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), omsDomain.SalesTrendGranularityDay)

	require.Error(t, err)
}
