package pms_test

import (
	"context"
	"testing"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSKUMarginInsight_reportsUnavailableWhenCronHasNeverRun(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", PriceMinorUnits: 1000}}
	uc := apppms.NewGetSKUMarginInsightUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, &stubSettingsRepoForPMS{},
		&stubBatchListerForCostPrice{}, &stubDiscardListerForMarginInsight{},
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	assert.False(t, result.CostPriceAvailable)
	assert.Equal(t, int64(0), result.CostPriceMinorUnits)
	assert.Equal(t, pmsDomain.SKUCostPriceSourceComputed, result.CostPriceSource)
}

func TestGetSKUMarginInsight_reportsManualCostPriceSource(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{
		ID: "sku-1", VariantID: "variant-1", PriceMinorUnits: 1000,
		CostPriceMinorUnits: int64Ptr2(700), CostPriceSource: pmsDomain.SKUCostPriceSourceManual,
	}}
	uc := apppms.NewGetSKUMarginInsightUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, &stubSettingsRepoForPMS{},
		&stubBatchListerForCostPrice{}, &stubDiscardListerForMarginInsight{},
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	assert.Equal(t, pmsDomain.SKUCostPriceSourceManual, result.CostPriceSource)
}

func TestGetSKUMarginInsight_computesProfitAndMarginAtCurrentAndSalePrice(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{
		ID: "sku-1", VariantID: "variant-1", CostPriceMinorUnits: int64Ptr2(700),
		PriceMinorUnits: 1000, SalePriceMinorUnits: int64Ptr2(800),
	}}
	batchLister := &stubBatchListerForCostPrice{batches: []invDomain.Batch{{ID: "batch-1", Quantity: 5, LandedCostMinorUnits: int64Ptr2(700)}}}
	settingsRepo := &stubSettingsRepoForPMS{settings: &settingsDomain.TenantSettings{DefaultTargetMarginBasisPoints: 3000}}
	uc := apppms.NewGetSKUMarginInsightUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, settingsRepo,
		batchLister, &stubDiscardListerForMarginInsight{},
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.True(t, result.CostPriceAvailable)
	assert.Equal(t, int64(700), result.CostPriceMinorUnits)
	assert.Equal(t, int64(1000), result.RecommendedPriceMinorUnits) // 700/0.7
	assert.Equal(t, int64(300), result.ProfitAtPriceMinorUnits)     // 1000-700
	assert.Equal(t, 3000, result.MarginAtPriceBasisPoints)          // 300/1000
	require.NotNil(t, result.ProfitAtSalePriceMinorUnits)
	assert.Equal(t, int64(100), *result.ProfitAtSalePriceMinorUnits) // 800-700
	require.NotNil(t, result.MarginAtSalePriceBasisPoints)
	assert.Equal(t, 1250, *result.MarginAtSalePriceBasisPoints) // 100/800 = 12.5%
	assert.Equal(t, int64(5), result.TotalOnHandQuantity)
	assert.Equal(t, int64(500), result.TotalProfitPotentialMinorUnits) // (800-700)*5, sale price is effective
}

func TestGetSKUMarginInsight_computesShrinkageFromDiscardsUsingBatchOrFallbackCost(t *testing.T) {
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", CostPriceMinorUnits: int64Ptr2(100)}}
	batchLister := &stubBatchListerForCostPrice{batches: []invDomain.Batch{{ID: "batch-1", Quantity: 5, LandedCostMinorUnits: int64Ptr2(200)}}}
	discardLister := &stubDiscardListerForMarginInsight{result: &invapp.ListDiscardsResult{Discards: []invDomain.DiscardRecordReport{
		{DiscardRecord: invDomain.DiscardRecord{BatchID: strPtr("batch-1"), Quantity: 2}}, // known batch cost 200 -> 400
		{DiscardRecord: invDomain.DiscardRecord{BatchID: nil, Quantity: 3}},               // general ledger -> fallback cost 100 -> 300
	}}}
	uc := apppms.NewGetSKUMarginInsightUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, &stubSettingsRepoForPMS{},
		batchLister, discardLister,
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	assert.Equal(t, int64(700), result.ShrinkageMinorUnits) // 400 + 300
}
