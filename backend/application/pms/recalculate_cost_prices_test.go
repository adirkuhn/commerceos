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

func TestRecalculateCostPrices_agreesWithLedgerAndSnapshotsHistoryFromTheActualCurrentCost(t *testing.T) {
	batchLister := &stubBatchListerForCostPrice{batches: []invDomain.Batch{
		{ID: "batch-1", Quantity: 10, LandedCostMinorUnits: int64Ptr2(800)},
		{ID: "batch-2", Quantity: 5, LandedCostMinorUnits: int64Ptr2(500)},
	}}
	// weighted avg: (10*800 + 5*500) / 15 = 700 -- matches the SKU's actual
	// current cost, set incrementally by ApplyReceiptCostUseCase at receipt
	// time (this job never writes it, only checks it).
	actual := int64(700)
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", CostPriceMinorUnits: &actual}}
	historyRepo := &stubSKUCostPriceHistoryRepo{}
	settingsRepo := &stubSettingsRepoForPMS{settings: &settingsDomain.TenantSettings{DefaultTargetMarkupBasisPoints: 3000}}
	uc := apppms.NewRecalculateCostPricesUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, settingsRepo, historyRepo, batchLister, &stubPurchaseCostLineListerForCostPrice{},
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.False(t, result.Skipped)
	assert.True(t, result.Reconciled)
	assert.Equal(t, int64(700), result.CostPriceMinorUnits)
	assert.Equal(t, int64(700), result.ExpectedCostPriceMinorUnits)
	// recommended at 30% markup on the ACTUAL cost: 700*1.3 = 910
	assert.Equal(t, int64(910), result.RecommendedPriceMinorUnits)
	assert.Empty(t, skuRepo.updateCostPriceCalls, "this job never writes cost_price_minor_units -- only ApplyReceiptCostUseCase does")
	require.Len(t, historyRepo.upserted, 1)
	assert.Equal(t, int64(700), historyRepo.upserted[0].CostPriceMinorUnits)
	assert.Equal(t, int64(910), historyRepo.upserted[0].RecommendedPriceMinorUnits)
}

// The safety net checks the moving average (inventory cost) -- the cost
// price is the last receipt's and legitimately differs; the margin-trend
// snapshot and recommended price use the cost price.
func TestRecalculateCostPrices_reconcilesTheInventoryCostAndSnapshotsTheCostPrice(t *testing.T) {
	batchLister := &stubBatchListerForCostPrice{batches: []invDomain.Batch{
		{ID: "batch-1", Quantity: 10, LandedCostMinorUnits: int64Ptr2(800)},
		{ID: "batch-2", Quantity: 5, LandedCostMinorUnits: int64Ptr2(500)},
	}}
	average, lastReceipt := int64(700), int64(500)
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1",
		CostPriceMinorUnits: &lastReceipt, InventoryCostMinorUnits: &average}}
	historyRepo := &stubSKUCostPriceHistoryRepo{}
	settingsRepo := &stubSettingsRepoForPMS{settings: &settingsDomain.TenantSettings{DefaultTargetMarkupBasisPoints: 3000}}
	uc := apppms.NewRecalculateCostPricesUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, settingsRepo, historyRepo, batchLister, &stubPurchaseCostLineListerForCostPrice{},
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	assert.True(t, result.Reconciled)
	assert.Equal(t, int64(700), result.InventoryCostMinorUnits)
	assert.Equal(t, int64(500), result.CostPriceMinorUnits)
	assert.Equal(t, int64(650), result.RecommendedPriceMinorUnits, "30% on the last receipt's cost")
	require.Len(t, historyRepo.upserted, 1)
	assert.Equal(t, int64(500), historyRepo.upserted[0].CostPriceMinorUnits)
}

func TestRecalculateCostPrices_flagsDriftWhenLedgerDisagreesWithTheActualCurrentCost(t *testing.T) {
	batchLister := &stubBatchListerForCostPrice{batches: []invDomain.Batch{
		{ID: "batch-1", Quantity: 10, LandedCostMinorUnits: int64Ptr2(800)},
	}}
	// SKU's actual cost (500) disagrees with what the ledger says it should
	// be (800) -- e.g. a bug, or a manual DB fix bypassing the app.
	actual := int64(500)
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", CostPriceMinorUnits: &actual}}
	uc := apppms.NewRecalculateCostPricesUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, &stubSettingsRepoForPMS{}, &stubSKUCostPriceHistoryRepo{}, batchLister, &stubPurchaseCostLineListerForCostPrice{},
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.False(t, result.Skipped)
	assert.False(t, result.Reconciled)
	assert.Equal(t, int64(500), result.CostPriceMinorUnits, "still reports the actual current cost, not the ledger figure")
	assert.Equal(t, int64(800), result.ExpectedCostPriceMinorUnits)
}

func TestRecalculateCostPrices_skipsSKUWithNoCurrentCost(t *testing.T) {
	// cost_price_source unknown (nil CostPriceMinorUnits) -- nothing to
	// reconcile against yet, even if batches exist.
	batchLister := &stubBatchListerForCostPrice{batches: []invDomain.Batch{
		{ID: "batch-1", Quantity: 10, LandedCostMinorUnits: int64Ptr2(800)},
	}}
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1"}}
	uc := apppms.NewRecalculateCostPricesUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, &stubSettingsRepoForPMS{}, &stubSKUCostPriceHistoryRepo{}, batchLister, &stubPurchaseCostLineListerForCostPrice{},
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	assert.True(t, result.Skipped)
}

func TestRecalculateCostPrices_skipsSKUWithNoCostBearingBatches(t *testing.T) {
	batchLister := &stubBatchListerForCostPrice{batches: []invDomain.Batch{
		{ID: "batch-1", Quantity: 10, LandedCostMinorUnits: nil},
	}}
	actual := int64(500)
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", CostPriceMinorUnits: &actual}}
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
	actual := int64(700)
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", CostPriceMinorUnits: &actual}}
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
	actual := int64(900)
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", CostPriceMinorUnits: &actual}}
	uc := apppms.NewRecalculateCostPricesUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, &stubSettingsRepoForPMS{}, &stubSKUCostPriceHistoryRepo{}, batchLister, purchaseCostLister,
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.False(t, result.Skipped)
	assert.True(t, result.Reconciled)
	assert.Equal(t, int64(900), result.ExpectedCostPriceMinorUnits)
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
	actual := int64(900)
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", CostPriceMinorUnits: &actual}}
	uc := apppms.NewRecalculateCostPricesUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, &stubSettingsRepoForPMS{}, &stubSKUCostPriceHistoryRepo{}, batchLister, purchaseCostLister,
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.False(t, result.Skipped)
	assert.Equal(t, int64(900), result.ExpectedCostPriceMinorUnits) // not counted twice via both sources
}

func TestRecalculateCostPrices_skipsSKUWithManualCostPriceOverride(t *testing.T) {
	// A human set this via SetManualCostPriceUseCase -- this job must leave
	// it alone entirely, not even look at batches/POs.
	batchLister := &stubBatchListerForCostPrice{batches: []invDomain.Batch{
		{ID: "batch-1", Quantity: 10, LandedCostMinorUnits: int64Ptr2(800)},
	}}
	purchaseCostLister := &stubPurchaseCostLineListerForCostPrice{}
	actual := int64(500)
	skuRepo := &stubSKURepo{findByIDResult: &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", CostPriceMinorUnits: &actual, CostPriceSource: pmsDomain.SKUCostPriceSourceManual}}
	uc := apppms.NewRecalculateCostPricesUseCase(
		newMockGormDB(t), skuRepo, &stubVariantRepo{}, &stubProductRepo{}, &stubCategoryRepo{}, &stubSettingsRepoForPMS{}, &stubSKUCostPriceHistoryRepo{}, batchLister, purchaseCostLister,
	)

	result, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	assert.True(t, result.Skipped)
	assert.Empty(t, skuRepo.updateCostPriceCalls)
}
