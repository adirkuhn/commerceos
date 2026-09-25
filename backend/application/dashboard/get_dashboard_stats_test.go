package dashboard_test

import (
	"context"
	"testing"
	"time"

	appdashboard "github.com/adirkuhn/commerceos/application/dashboard"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubStockReportRepo struct {
	totalOnHand   int64
	totalErr      error
	dailyFlow     []invDomain.DailyStockFlow
	dailyFlowErr  error
	gotDailySince time.Time
	gotTimezone   string
}

func (s *stubStockReportRepo) TotalOnHand(_ context.Context) (int64, error) {
	return s.totalOnHand, s.totalErr
}
func (s *stubStockReportRepo) DailyFlow(_ context.Context, since time.Time, timezone string) ([]invDomain.DailyStockFlow, error) {
	s.gotDailySince, s.gotTimezone = since, timezone
	return s.dailyFlow, s.dailyFlowErr
}

type stubSalesReportRepo struct {
	revenue         int64
	revenueErr      error
	gotRevenueSince time.Time
	gotRevenueCcy   string
	byChannel       []omsDomain.ChannelRevenue
	byChannelErr    error
	topProducts     []omsDomain.ProductSales
	topProductsErr  error
	gotTopProductsN int
}

func (s *stubSalesReportRepo) RevenueSince(_ context.Context, since time.Time, currencyCode string) (int64, error) {
	s.gotRevenueSince, s.gotRevenueCcy = since, currencyCode
	return s.revenue, s.revenueErr
}
func (s *stubSalesReportRepo) RevenueByChannelSince(_ context.Context, _ time.Time, _ string) ([]omsDomain.ChannelRevenue, error) {
	return s.byChannel, s.byChannelErr
}
func (s *stubSalesReportRepo) TopProductsSince(_ context.Context, _ time.Time, _ string, limit int) ([]omsDomain.ProductSales, error) {
	s.gotTopProductsN = limit
	return s.topProducts, s.topProductsErr
}

type stubSettingsRepo struct {
	settings *settingsDomain.TenantSettings
	err      error
}

func (s *stubSettingsRepo) Get(_ context.Context) (*settingsDomain.TenantSettings, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.settings != nil {
		return s.settings, nil
	}
	return &settingsDomain.TenantSettings{}, nil
}
func (s *stubSettingsRepo) Upsert(_ context.Context, _ *settingsDomain.TenantSettings) error {
	return nil
}
func fixedNow() time.Time { return time.Date(2026, 8, 22, 15, 4, 5, 0, time.UTC) }

func TestGetDashboardStats_composesAllFieldsFromRepos(t *testing.T) {
	stockRepo := &stubStockReportRepo{
		totalOnHand: 84293,
		dailyFlow:   []invDomain.DailyStockFlow{{Date: fixedNow(), In: 10, Out: 5}},
	}
	salesRepo := &stubSalesReportRepo{
		revenue:     11600000,
		byChannel:   []omsDomain.ChannelRevenue{{Source: omsDomain.SalesOrderSourceWooCommerce, TotalMinorUnits: 3150000}},
		topProducts: []omsDomain.ProductSales{{SKUCode: "WHP-001", ProductName: "Headphones", QuantitySold: 234, RevenueMinorUnits: 5850000}},
	}
	settingsRepo := &stubSettingsRepo{settings: &settingsDomain.TenantSettings{DefaultCurrency: "EUR"}}
	uc := appdashboard.NewGetDashboardStatsUseCase(newMockGormDB(t), stockRepo, salesRepo, settingsRepo)
	uc.Now = fixedNow

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(84293), result.TotalStockUnits)
	assert.Equal(t, "EUR", result.CurrencyCode)
	assert.Equal(t, int64(11600000), result.RevenueThisMonthMinorUnits)
	assert.Equal(t, stockRepo.dailyFlow, result.StockFlowLast7Days)
	assert.Equal(t, salesRepo.byChannel, result.SalesByChannelThisMonth)
	assert.Equal(t, salesRepo.topProducts, result.TopProductsThisMonth)
}

func TestGetDashboardStats_fallsBackToUSDWhenNoDefaultCurrencySet(t *testing.T) {
	settingsRepo := &stubSettingsRepo{settings: &settingsDomain.TenantSettings{DefaultCurrency: ""}}
	salesRepo := &stubSalesReportRepo{}
	uc := appdashboard.NewGetDashboardStatsUseCase(newMockGormDB(t), &stubStockReportRepo{}, salesRepo, settingsRepo)
	uc.Now = fixedNow

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "USD", result.CurrencyCode)
	assert.Equal(t, "USD", salesRepo.gotRevenueCcy)
}

func TestGetDashboardStats_revenueWindowIsCalendarMonthStartUTC(t *testing.T) {
	salesRepo := &stubSalesReportRepo{}
	uc := appdashboard.NewGetDashboardStatsUseCase(newMockGormDB(t), &stubStockReportRepo{}, salesRepo, &stubSettingsRepo{})
	uc.Now = fixedNow

	_, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), salesRepo.gotRevenueSince)
}

func TestGetDashboardStats_stockFlowWindowIsSevenDaysIncludingToday(t *testing.T) {
	stockRepo := &stubStockReportRepo{}
	uc := appdashboard.NewGetDashboardStatsUseCase(newMockGormDB(t), stockRepo, &stubSalesReportRepo{}, &stubSettingsRepo{})
	uc.Now = fixedNow

	_, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC), stockRepo.gotDailySince)
}

func TestGetDashboardStats_topProductsLimitedToFive(t *testing.T) {
	salesRepo := &stubSalesReportRepo{}
	uc := appdashboard.NewGetDashboardStatsUseCase(newMockGormDB(t), &stubStockReportRepo{}, salesRepo, &stubSettingsRepo{})
	uc.Now = fixedNow

	_, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 5, salesRepo.gotTopProductsN)
}

func TestGetDashboardStats_propagatesSettingsError(t *testing.T) {
	uc := appdashboard.NewGetDashboardStatsUseCase(newMockGormDB(t), &stubStockReportRepo{}, &stubSalesReportRepo{}, &stubSettingsRepo{err: assert.AnError})
	uc.Now = fixedNow

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}

func TestGetDashboardStats_propagatesStockRepoError(t *testing.T) {
	uc := appdashboard.NewGetDashboardStatsUseCase(newMockGormDB(t), &stubStockReportRepo{totalErr: assert.AnError}, &stubSalesReportRepo{}, &stubSettingsRepo{})
	uc.Now = fixedNow

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}

func TestGetDashboardStats_propagatesSalesRepoError(t *testing.T) {
	uc := appdashboard.NewGetDashboardStatsUseCase(newMockGormDB(t), &stubStockReportRepo{}, &stubSalesReportRepo{revenueErr: assert.AnError}, &stubSettingsRepo{})
	uc.Now = fixedNow

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}

// "This month" and "last 7 days" are the tenant's local calendar: at
// 22:30 UTC on Aug 31 it is already Sep 1 in Amsterdam (UTC+2), so the
// month -- and today -- started at Amsterdam midnight, not UTC midnight.
func TestGetDashboardStats_windowsFollowTheTenantTimezone(t *testing.T) {
	ams, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)
	settingsRepo := &stubSettingsRepo{settings: &settingsDomain.TenantSettings{Timezone: "Europe/Amsterdam"}}
	stockRepo := &stubStockReportRepo{}
	salesRepo := &stubSalesReportRepo{}
	uc := appdashboard.NewGetDashboardStatsUseCase(newMockGormDB(t), stockRepo, salesRepo, settingsRepo)
	uc.Now = func() time.Time { return time.Date(2026, 8, 31, 22, 30, 0, 0, time.UTC) }

	_, err = uc.Execute(context.Background())

	require.NoError(t, err)
	assert.True(t, time.Date(2026, 9, 1, 0, 0, 0, 0, ams).Equal(salesRepo.gotRevenueSince), salesRepo.gotRevenueSince)
	assert.True(t, time.Date(2026, 8, 26, 0, 0, 0, 0, ams).Equal(stockRepo.gotDailySince), stockRepo.gotDailySince)
	assert.Equal(t, "Europe/Amsterdam", stockRepo.gotTimezone)
}
