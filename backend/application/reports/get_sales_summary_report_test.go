package reports_test

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

func newMockGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.MatchExpectationsInOrder(false)
	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	return gormDB
}

type stubSalesReportRepo struct {
	byRange map[string]omsDomain.SalesSummary
	err     error
	calls   []struct{ From, To time.Time }
}

func (s *stubSalesReportRepo) SalesSummaryBetween(_ context.Context, from, to time.Time, _ string) (omsDomain.SalesSummary, error) {
	s.calls = append(s.calls, struct{ From, To time.Time }{from, to})
	if s.err != nil {
		return omsDomain.SalesSummary{}, s.err
	}
	return s.byRange[from.Format(time.RFC3339)], nil
}

type stubSettingsRepo struct {
	settings *settingsDomain.TenantSettings
}

func (s *stubSettingsRepo) Get(_ context.Context) (*settingsDomain.TenantSettings, error) {
	if s.settings != nil {
		return s.settings, nil
	}
	return &settingsDomain.TenantSettings{}, nil
}
func (s *stubSettingsRepo) Upsert(_ context.Context, _ *settingsDomain.TenantSettings) error {
	return nil
}
func (s *stubSettingsRepo) GetStorefrontKeyHash(_ context.Context) (string, error) {
	return "", nil
}
func (s *stubSettingsRepo) SetStorefrontKeyHash(_ context.Context, _ string) error {
	return nil
}
func (s *stubSettingsRepo) FindTenantIDByStorefrontKeyHash(_ context.Context, _ string) (string, error) {
	return "", settingsDomain.ErrStorefrontKeyNotFound
}

func TestGetSalesSummaryReport_computesPercentChangeVsPreviousEqualLengthPeriod(t *testing.T) {
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	prevFrom := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)

	salesRepo := &stubSalesReportRepo{byRange: map[string]omsDomain.SalesSummary{
		from.Format(time.RFC3339):     {RevenueMinorUnits: 118000, OrderCount: 112, UnitsSold: 217, AverageOrderValueMinorUnits: 1053},
		prevFrom.Format(time.RFC3339): {RevenueMinorUnits: 100000, OrderCount: 100, UnitsSold: 178, AverageOrderValueMinorUnits: 1000},
	}}
	uc := appreports.NewGetSalesSummaryReportUseCase(newMockGormDB(t), salesRepo, &stubSettingsRepo{settings: &settingsDomain.TenantSettings{DefaultCurrency: "EUR"}})

	result, err := uc.Execute(context.Background(), from, to)

	require.NoError(t, err)
	require.Equal(t, "EUR", result.CurrencyCode)
	require.Equal(t, int64(118000), result.Current.RevenueMinorUnits)
	require.Equal(t, int64(100000), result.Previous.RevenueMinorUnits)
	require.NotNil(t, result.RevenueChangePercent)
	require.InDelta(t, 18.0, *result.RevenueChangePercent, 0.01)
	require.NotNil(t, result.OrderCountChangePercent)
	require.InDelta(t, 12.0, *result.OrderCountChangePercent, 0.01)
	require.NotNil(t, result.UnitsSoldChangePercent)
	require.InDelta(t, 21.9, *result.UnitsSoldChangePercent, 0.1)
	require.NotNil(t, result.AOVChangePercent)
	require.InDelta(t, 5.3, *result.AOVChangePercent, 0.1)
}

func TestGetSalesSummaryReport_queriesPreviousPeriodOfEqualLength(t *testing.T) {
	from := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC) // 7-day window
	salesRepo := &stubSalesReportRepo{}
	uc := appreports.NewGetSalesSummaryReportUseCase(newMockGormDB(t), salesRepo, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), from, to)

	require.NoError(t, err)
	require.Len(t, salesRepo.calls, 2)
	require.Equal(t, from, salesRepo.calls[0].From)
	require.Equal(t, to, salesRepo.calls[0].To)
	require.Equal(t, time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC), salesRepo.calls[1].From)
	require.Equal(t, from, salesRepo.calls[1].To)
}

func TestGetSalesSummaryReport_nilPercentChangeWhenPreviousBaselineIsZero(t *testing.T) {
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	salesRepo := &stubSalesReportRepo{byRange: map[string]omsDomain.SalesSummary{
		from.Format(time.RFC3339): {RevenueMinorUnits: 5000, OrderCount: 3, UnitsSold: 3, AverageOrderValueMinorUnits: 1666},
	}}
	uc := appreports.NewGetSalesSummaryReportUseCase(newMockGormDB(t), salesRepo, &stubSettingsRepo{})

	result, err := uc.Execute(context.Background(), from, to)

	require.NoError(t, err)
	require.Nil(t, result.RevenueChangePercent, "can't compute a meaningful percent change off a zero baseline")
}

func TestGetSalesSummaryReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetSalesSummaryReportUseCase(newMockGormDB(t), &stubSalesReportRepo{}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour))

	require.Error(t, err)
}

func TestGetSalesSummaryReport_propagatesRepoError(t *testing.T) {
	uc := appreports.NewGetSalesSummaryReportUseCase(newMockGormDB(t), &stubSalesReportRepo{err: context.DeadlineExceeded}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.Error(t, err)
}
