package pms_test

import (
	"context"
	"testing"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListMarginReport_skipsSKUsWithNoCostPriceAndSortsWorstMarginFirst(t *testing.T) {
	skuRepo := &stubSKURepo{listActiveResult: []pmsDomain.SKU{
		{ID: "sku-no-cost", VariantID: "variant-1", PriceMinorUnits: 1000},
		{ID: "sku-good-margin", VariantID: "variant-1", PriceMinorUnits: 1000, CostPriceMinorUnits: int64Ptr2(500)},
		{ID: "sku-bad-margin", VariantID: "variant-1", PriceMinorUnits: 1000, CostPriceMinorUnits: int64Ptr2(950)},
	}}
	settingsRepo := &stubSettingsRepoForPMS{settings: &settingsDomain.TenantSettings{DefaultTargetMarginBasisPoints: 3000}}
	uc := apppms.NewListMarginReportUseCase(newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, settingsRepo)

	result, err := uc.Execute(context.Background(), apppms.ListMarginReportInput{})

	require.NoError(t, err)
	require.Equal(t, int64(2), result.Total) // sku-no-cost excluded
	require.Len(t, result.Rows, 2)
	assert.Equal(t, "sku-bad-margin", result.Rows[0].SKUID) // worst margin first
	assert.True(t, result.Rows[0].BelowTarget)
	assert.Equal(t, "sku-good-margin", result.Rows[1].SKUID)
	assert.False(t, result.Rows[1].BelowTarget)
}

func TestListMarginReport_belowTargetOnlyFiltersOutHealthyMargins(t *testing.T) {
	skuRepo := &stubSKURepo{listActiveResult: []pmsDomain.SKU{
		{ID: "sku-good-margin", VariantID: "variant-1", PriceMinorUnits: 1000, CostPriceMinorUnits: int64Ptr2(500)},
		{ID: "sku-bad-margin", VariantID: "variant-1", PriceMinorUnits: 1000, CostPriceMinorUnits: int64Ptr2(950)},
	}}
	settingsRepo := &stubSettingsRepoForPMS{settings: &settingsDomain.TenantSettings{DefaultTargetMarginBasisPoints: 3000}}
	uc := apppms.NewListMarginReportUseCase(newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, settingsRepo)

	result, err := uc.Execute(context.Background(), apppms.ListMarginReportInput{BelowTargetOnly: true})

	require.NoError(t, err)
	require.Len(t, result.Rows, 1)
	assert.Equal(t, "sku-bad-margin", result.Rows[0].SKUID)
}
