package pms_test

import (
	"context"
	"testing"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr2(v int64) *int64 { return &v }

func TestRecalculateCostPrices_weightedAveragesOnHandBatchesAndAppliesTenantDefaultMarkup(t *testing.T) {
	batchLister := &stubBatchListerForCostPrice{batches: []invDomain.Batch{
		{ID: "batch-1", Quantity: 10, LandedCostMinorUnits: int64Ptr2(800)},
		{ID: "batch-2", Quantity: 5, LandedCostMinorUnits: int64Ptr2(500)},
	}}
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1"}}
	historyRepo := &stubSKUCostPriceHistoryRepo{}
	settingsRepo := &stubSettingsRepoForPMS{settings: &settingsDomain.TenantSettings{DefaultTargetMarkupBasisPoints: 3000}}
	uc := apppms.NewRecalculateCostPricesUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, settingsRepo, historyRepo, batchLister, &stubPurchaseCostLineListerForCostPrice{},
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.False(t, result.Skipped)
	// weighted avg: (10*800 + 5*500) / 15 = (8000+2500)/15 = 700
	assert.Equal(t, int64(700), result.CostPriceMinorUnits)
	// recommended at 30% markup: 700*1.3 = 910
	assert.Equal(t, int64(910), result.RecommendedPriceMinorUnits)
	require.Len(t, skuRepo.updateCostPriceCalls, 1)
	assert.Equal(t, int64(700), skuRepo.updateCostPriceCalls[0].CostPriceMinorUnits)
	require.Len(t, historyRepo.upserted, 1)
	assert.Equal(t, int64(700), historyRepo.upserted[0].CostPriceMinorUnits)
	assert.Equal(t, int64(910), historyRepo.upserted[0].RecommendedPriceMinorUnits)
}

func TestRecalculateCostPrices_skipsBatchesWithNoLandedCost(t *testing.T) {
	batchLister := &stubBatchListerForCostPrice{batches: []invDomain.Batch{
		{ID: "batch-1", Quantity: 10, LandedCostMinorUnits: nil}, // pre-landed-cost stock, excluded
		{ID: "batch-2", Quantity: 5, LandedCostMinorUnits: int64Ptr2(500)},
	}}
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1"}}
	uc := apppms.NewRecalculateCostPricesUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, &stubSettingsRepoForPMS{}, &stubSKUCostPriceHistoryRepo{}, batchLister, &stubPurchaseCostLineListerForCostPrice{},
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.False(t, result.Skipped)
	assert.Equal(t, int64(500), result.CostPriceMinorUnits) // only batch-2 counted
}

func TestRecalculateCostPrices_skipsSKUWithNoCostBearingBatches(t *testing.T) {
	batchLister := &stubBatchListerForCostPrice{batches: []invDomain.Batch{
		{ID: "batch-1", Quantity: 10, LandedCostMinorUnits: nil},
	}}
	skuRepo := &stubSKURepo{}
	uc := apppms.NewRecalculateCostPricesUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, &stubSettingsRepoForPMS{}, &stubSKUCostPriceHistoryRepo{}, batchLister, &stubPurchaseCostLineListerForCostPrice{},
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	assert.True(t, result.Skipped)
	assert.Empty(t, skuRepo.updateCostPriceCalls)
}

func TestRecalculateCostPrices_categoryOverrideBeatsTenantDefault(t *testing.T) {
	batchLister := &stubBatchListerForCostPrice{batches: []invDomain.Batch{
		{ID: "batch-1", Quantity: 1, LandedCostMinorUnits: int64Ptr2(700)},
	}}
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1"}}
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "product-1", CategoryID: strPtr("category-1")}}
	categoryRepo := &stubCategoryRepo{findByIDResult: &pmsDomain.Category{ID: "category-1", TargetMarkupBasisPoints: intPtr(5000)}}
	settingsRepo := &stubSettingsRepoForPMS{settings: &settingsDomain.TenantSettings{DefaultTargetMarkupBasisPoints: 1000}}
	uc := apppms.NewRecalculateCostPricesUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, productRepo, categoryRepo, settingsRepo, &stubSKUCostPriceHistoryRepo{}, batchLister, &stubPurchaseCostLineListerForCostPrice{},
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	// 700 * 1.5 = 1050, not the tenant default's 700*1.1=770
	assert.Equal(t, int64(1050), result.RecommendedPriceMinorUnits)
}

func TestRecalculateCostPrices_includesReceivedPurchaseOrderLinesWithNoBatch(t *testing.T) {
	// The exact gap this covers: a PO marked "received" directly via the
	// Edit Purchase Order form's status field, which never creates an
	// inventory.Batch (see UpdatePurchaseOrderUseCase) -- so there's no
	// batch to average, only the PO's own line item.
	batchLister := &stubBatchListerForCostPrice{}
	purchaseCostLister := &stubPurchaseCostLineListerForCostPrice{lines: []apppms.PurchaseCostLine{
		{PurchaseOrderLineItemID: "line-1", Quantity: 10, LandedUnitCostMinorUnits: 900},
	}}
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1"}}
	uc := apppms.NewRecalculateCostPricesUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, &stubSettingsRepoForPMS{}, &stubSKUCostPriceHistoryRepo{}, batchLister, purchaseCostLister,
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.False(t, result.Skipped)
	assert.Equal(t, int64(900), result.CostPriceMinorUnits)
}

func TestRecalculateCostPrices_doesNotDoubleCountAPurchaseOrderLineAlreadyBookedToABatch(t *testing.T) {
	batchLister := &stubBatchListerForCostPrice{batches: []invDomain.Batch{
		{ID: "batch-1", Quantity: 10, LandedCostMinorUnits: int64Ptr2(900), PurchaseOrderLineItemID: strPtr("line-1")},
	}}
	// Same line item the batch above already came from -- must be excluded,
	// or this SKU's cost basis would count the same receipt twice.
	purchaseCostLister := &stubPurchaseCostLineListerForCostPrice{lines: []apppms.PurchaseCostLine{
		{PurchaseOrderLineItemID: "line-1", Quantity: 10, LandedUnitCostMinorUnits: 900},
	}}
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1"}}
	uc := apppms.NewRecalculateCostPricesUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, &stubSettingsRepoForPMS{}, &stubSKUCostPriceHistoryRepo{}, batchLister, purchaseCostLister,
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.False(t, result.Skipped)
	assert.Equal(t, int64(900), result.CostPriceMinorUnits) // not counted twice via both sources
}

func TestRecalculateCostPrices_skipsSKUWithManualCostPriceOverride(t *testing.T) {
	// A human set this via SetManualCostPriceUseCase -- the cron must leave
	// it alone entirely, not even look at batches/POs.
	batchLister := &stubBatchListerForCostPrice{batches: []invDomain.Batch{
		{ID: "batch-1", Quantity: 10, LandedCostMinorUnits: int64Ptr2(800)},
	}}
	purchaseCostLister := &stubPurchaseCostLineListerForCostPrice{}
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", CostPriceSource: pmsDomain.SKUCostPriceSourceManual}}
	uc := apppms.NewRecalculateCostPricesUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, &stubSettingsRepoForPMS{}, &stubSKUCostPriceHistoryRepo{}, batchLister, purchaseCostLister,
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	assert.True(t, result.Skipped)
	assert.Empty(t, skuRepo.updateCostPriceCalls)
}
