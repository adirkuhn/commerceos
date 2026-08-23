package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

type stubProductPerformanceRepo struct {
	out []omsDomain.ProductPerformance
	err error
}

func (s *stubProductPerformanceRepo) ProductPerformanceBetween(_ context.Context, _, _ time.Time, _ string) ([]omsDomain.ProductPerformance, error) {
	return s.out, s.err
}

type stubSKUCostPriceLookup struct {
	byID map[string]*pmsDomain.SKU
	err  error
}

func (s *stubSKUCostPriceLookup) FindByID(_ context.Context, id string) (*pmsDomain.SKU, error) {
	if s.err != nil {
		return nil, s.err
	}
	sku, ok := s.byID[id]
	if !ok {
		return nil, pmsDomain.ErrSKUNotFound
	}
	return sku, nil
}

func int64Ptr(v int64) *int64 { return &v }

func TestGetProductPerformanceReport_returnsRepoResultInCurrency(t *testing.T) {
	repo := &stubProductPerformanceRepo{out: []omsDomain.ProductPerformance{
		{SKUCode: "SKU-A", ProductName: "Acai", QuantitySold: 5, RevenueMinorUnits: 2500, OrderCount: 2},
	}}
	uc := appreports.NewGetProductPerformanceReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{settings: &settingsDomain.TenantSettings{DefaultCurrency: "EUR"}}, &stubSKUCostPriceLookup{})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Equal(t, "EUR", result.CurrencyCode)
	require.Len(t, result.Rows, 1)
	require.Equal(t, "SKU-A", result.Rows[0].SKUCode)
}

func TestGetProductPerformanceReport_computesProfitAndMarginWhenCostPriceKnown(t *testing.T) {
	skuID := "sku-1"
	repo := &stubProductPerformanceRepo{out: []omsDomain.ProductPerformance{
		{SKUID: &skuID, SKUCode: "SKU-A", QuantitySold: 5, RevenueMinorUnits: 2500},
	}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{
		"sku-1": {ID: "sku-1", CostPriceMinorUnits: int64Ptr(300)},
	}}
	uc := appreports.NewGetProductPerformanceReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{}, skuLookup)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Len(t, result.Rows, 1)
	row := result.Rows[0]
	require.NotNil(t, row.CostPriceMinorUnits)
	require.Equal(t, int64(300), *row.CostPriceMinorUnits)
	require.NotNil(t, row.ProfitMinorUnits)
	require.Equal(t, int64(1000), *row.ProfitMinorUnits) // 2500 - 300*5
	require.NotNil(t, row.MarginBasisPoints)
	require.Equal(t, int64(4000), *row.MarginBasisPoints) // 1000/2500 = 40.00%
}

func TestGetProductPerformanceReport_nilMarginWhenCostPriceUnknown(t *testing.T) {
	skuID := "sku-1"
	repo := &stubProductPerformanceRepo{out: []omsDomain.ProductPerformance{
		{SKUID: &skuID, SKUCode: "SKU-A", QuantitySold: 5, RevenueMinorUnits: 2500},
	}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{
		"sku-1": {ID: "sku-1", CostPriceMinorUnits: nil},
	}}
	uc := appreports.NewGetProductPerformanceReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{}, skuLookup)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Nil(t, result.Rows[0].CostPriceMinorUnits)
	require.Nil(t, result.Rows[0].ProfitMinorUnits)
	require.Nil(t, result.Rows[0].MarginBasisPoints)
}

func TestGetProductPerformanceReport_nilMarginForAdhocLineItemWithNoSKUID(t *testing.T) {
	repo := &stubProductPerformanceRepo{out: []omsDomain.ProductPerformance{
		{SKUID: nil, SKUCode: "ADHOC", QuantitySold: 1, RevenueMinorUnits: 500},
	}}
	uc := appreports.NewGetProductPerformanceReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{}, &stubSKUCostPriceLookup{})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Nil(t, result.Rows[0].CostPriceMinorUnits)
}

func TestGetProductPerformanceReport_nilMarginWhenSKUSinceDeleted(t *testing.T) {
	skuID := "sku-deleted"
	repo := &stubProductPerformanceRepo{out: []omsDomain.ProductPerformance{
		{SKUID: &skuID, SKUCode: "SKU-GONE", QuantitySold: 1, RevenueMinorUnits: 500},
	}}
	uc := appreports.NewGetProductPerformanceReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{}, &stubSKUCostPriceLookup{})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.NoError(t, err)
	require.Nil(t, result.Rows[0].CostPriceMinorUnits)
}

func TestGetProductPerformanceReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetProductPerformanceReportUseCase(newMockGormDB(t), &stubProductPerformanceRepo{}, &stubSettingsRepo{}, &stubSKUCostPriceLookup{})

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour))

	require.Error(t, err)
}

func TestGetProductPerformanceReport_propagatesRepoError(t *testing.T) {
	uc := appreports.NewGetProductPerformanceReportUseCase(newMockGormDB(t), &stubProductPerformanceRepo{err: context.DeadlineExceeded}, &stubSettingsRepo{}, &stubSKUCostPriceLookup{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now())

	require.Error(t, err)
}
