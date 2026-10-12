package dashboard_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appdashboard "github.com/adirkuhn/commerceos/application/dashboard"
	dashboardDomain "github.com/adirkuhn/commerceos/domain/dashboard"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

// stubSalesRangeReader answers per range start (unix seconds: the same
// instant in another *time.Location is a different map key), recording
// every call.
type stubSalesRangeReader struct {
	summaries  map[int64]omsDomain.SalesSummary
	channels   map[int64][]omsDomain.ChannelRevenue
	summaryErr error
	channelErr error
	gotRanges  []dashboardDomain.DateRange
	gotCcy     string
}

func (s *stubSalesRangeReader) SalesSummaryBetween(_ context.Context, from, to time.Time, currencyCode string) (omsDomain.SalesSummary, error) {
	s.gotRanges = append(s.gotRanges, dashboardDomain.DateRange{From: from, To: to})
	s.gotCcy = currencyCode
	return s.summaries[from.Unix()], s.summaryErr
}

func (s *stubSalesRangeReader) RevenueByChannelBetween(_ context.Context, from, _ time.Time, _ string) ([]omsDomain.ChannelRevenue, error) {
	return s.channels[from.Unix()], s.channelErr
}

func TestGetSalesComparison_comparesThePeriodSoFarWithTheWholePreviousOne(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)
	weekStart := time.Date(2026, 10, 11, 0, 0, 0, 0, loc)
	prevWeekStart := time.Date(2026, 10, 4, 0, 0, 0, 0, loc)
	reader := &stubSalesRangeReader{
		summaries: map[int64]omsDomain.SalesSummary{
			weekStart.Unix():     {RevenueMinorUnits: 12000, OrderCount: 6},
			prevWeekStart.Unix(): {RevenueMinorUnits: 10000, OrderCount: 5},
		},
		channels: map[int64][]omsDomain.ChannelRevenue{
			weekStart.Unix():     {{Source: omsDomain.SalesOrderSourcePOS, TotalMinorUnits: 12000}},
			prevWeekStart.Unix(): {{Source: omsDomain.SalesOrderSourcePOS, TotalMinorUnits: 10000}},
		},
	}
	settings := &stubSettingsRepo{settings: &settingsDomain.TenantSettings{DefaultCurrency: "EUR", Timezone: "Europe/Amsterdam"}}
	uc := appdashboard.NewGetSalesComparisonUseCase(newMockGormDB(t), reader, settings)
	uc.Now = func() time.Time { return time.Date(2026, 10, 14, 9, 0, 0, 0, loc) } // Wednesday

	got, err := uc.Execute(context.Background(), dashboardDomain.SalesPeriodThisWeek)

	require.NoError(t, err)
	assert.Equal(t, dashboardDomain.SalesPeriodThisWeek, got.Period)
	assert.Equal(t, "EUR", got.CurrencyCode)
	assert.Equal(t, "EUR", reader.gotCcy)
	assert.True(t, got.Current.Range.From.Equal(weekStart))
	assert.True(t, got.Previous.Range.From.Equal(prevWeekStart))
	assert.True(t, got.Previous.Range.To.Equal(weekStart))
	assert.Equal(t, int64(12000), got.Current.RevenueMinorUnits)
	assert.Equal(t, int64(6), got.Current.OrderCount)
	assert.Equal(t, int64(10000), got.Previous.RevenueMinorUnits)
	assert.Equal(t, []omsDomain.ChannelRevenue{{Source: omsDomain.SalesOrderSourcePOS, TotalMinorUnits: 10000}}, got.Previous.ByChannel)
	require.NotNil(t, got.ChangePercent)
	assert.InDelta(t, 20.0, *got.ChangePercent, 0.0001)
}

func TestGetSalesComparison_noPreviousSalesHasNoChangePercent(t *testing.T) {
	uc := appdashboard.NewGetSalesComparisonUseCase(newMockGormDB(t), &stubSalesRangeReader{}, &stubSettingsRepo{})
	uc.Now = fixedNow

	got, err := uc.Execute(context.Background(), dashboardDomain.SalesPeriodToday)

	require.NoError(t, err)
	assert.Equal(t, "USD", got.CurrencyCode, "never-saved settings fall back like the stats")
	assert.Nil(t, got.ChangePercent)
	assert.NotNil(t, got.Current.ByChannel, "empty list, not null")
}

func TestGetSalesComparison_propagatesErrors(t *testing.T) {
	for name, uc := range map[string]*appdashboard.GetSalesComparisonUseCase{
		"settings": appdashboard.NewGetSalesComparisonUseCase(newMockGormDB(t), &stubSalesRangeReader{}, &stubSettingsRepo{err: assert.AnError}),
		"summary":  appdashboard.NewGetSalesComparisonUseCase(newMockGormDB(t), &stubSalesRangeReader{summaryErr: assert.AnError}, &stubSettingsRepo{}),
		"channels": appdashboard.NewGetSalesComparisonUseCase(newMockGormDB(t), &stubSalesRangeReader{channelErr: assert.AnError}, &stubSettingsRepo{}),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := uc.Execute(context.Background(), dashboardDomain.SalesPeriodThisMonth)
			assert.ErrorIs(t, err, assert.AnError)
		})
	}
}
