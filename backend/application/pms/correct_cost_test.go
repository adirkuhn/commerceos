package pms_test

import (
	"context"
	"testing"
	"time"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyValueCorrectionToCost_addsAdjustmentValueAndReblends(t *testing.T) {
	// 100 units @ 300 (value 30000), invoice correction adds 5000 more
	// value (e.g. January should've been priced higher) -> (30000+5000)/100 = 350.
	got, err := apppms.ApplyValueCorrectionToCost(100, 300, 5000)

	require.NoError(t, err)
	assert.Equal(t, int64(350), got)
}

func TestApplyValueCorrectionToCost_negativeAdjustmentLowersCost(t *testing.T) {
	got, err := apppms.ApplyValueCorrectionToCost(100, 300, -2000)

	require.NoError(t, err)
	assert.Equal(t, int64(280), got)
}

func TestApplyValueCorrectionToCost_roundsHalfUp(t *testing.T) {
	// value 300 / 200 qty = 1.5 -> rounds to 2
	got, err := apppms.ApplyValueCorrectionToCost(200, 1, 100)

	require.NoError(t, err)
	assert.Equal(t, int64(2), got)
}

func TestApplyValueCorrectionToCost_rejectsNonPositiveExistingQuantity(t *testing.T) {
	_, err := apppms.ApplyValueCorrectionToCost(0, 300, 5000)

	require.Error(t, err)
}

func TestCorrectCost_writesNewCostAndCorrectionLedgerEntryWithoutTouchingQuantity(t *testing.T) {
	existingAvg := int64(300)
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &existingAvg, CostPriceSource: pmsDomain.SKUCostPriceSourceComputed}}
	batchLister := &stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 100}}}
	ledgerRepo := &stubCostLedgerRepo{}
	occurredAt := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	uc := apppms.NewCorrectCostUseCase(newMockGormDB(t), skuRepo, ledgerRepo, batchLister)

	_, err := uc.Execute(context.Background(), apppms.CorrectCostInput{
		SKUID: "sku-1", AdjustmentValueMinorUnits: 5000, ReferenceType: "purchase_order_line_item", ReferenceID: "line-1", OccurredAt: occurredAt,
	})

	require.NoError(t, err)
	require.Len(t, skuRepo.updateCalls, 1)
	assert.Equal(t, int64(350), skuRepo.updateCalls[0].costPriceMinorUnits)
	require.Len(t, ledgerRepo.created, 1)
	entry := ledgerRepo.created[0]
	assert.Equal(t, pmsDomain.SKUCostPriceLedgerSourceCorrection, entry.SourceType)
	assert.Equal(t, int64(0), entry.QuantityDelta, "a correction changes cost, never quantity")
	assert.Equal(t, int64(100), entry.QuantityAfter)
	require.NotNil(t, entry.CostPriceMinorUnits)
	assert.Equal(t, int64(350), *entry.CostPriceMinorUnits)
	require.NotNil(t, entry.InventoryValueMinorUnits)
	assert.Equal(t, int64(35000), *entry.InventoryValueMinorUnits)
}

func TestCorrectCost_rejectsWhenCostIsUnknown(t *testing.T) {
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: nil}}
	uc := apppms.NewCorrectCostUseCase(newMockGormDB(t), skuRepo, &stubCostLedgerRepo{}, &stubBatchListerForReceiptCost{})

	_, err := uc.Execute(context.Background(), apppms.CorrectCostInput{SKUID: "sku-1", AdjustmentValueMinorUnits: 5000, OccurredAt: time.Now()})

	require.ErrorIs(t, err, pmsDomain.ErrCostPriceUnknownForCorrection)
	assert.Empty(t, skuRepo.updateCalls)
}

func TestCorrectCost_rejectsWhenNoStockOnHand(t *testing.T) {
	existingAvg := int64(300)
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &existingAvg}}
	batchLister := &stubBatchListerForReceiptCost{}
	uc := apppms.NewCorrectCostUseCase(newMockGormDB(t), skuRepo, &stubCostLedgerRepo{}, batchLister)

	_, err := uc.Execute(context.Background(), apppms.CorrectCostInput{SKUID: "sku-1", AdjustmentValueMinorUnits: 5000, OccurredAt: time.Now()})

	require.ErrorIs(t, err, pmsDomain.ErrNoStockOnHandForCorrection)
}

func TestCorrectCost_neverTouchesHistoricalSalesJustFutureCost(t *testing.T) {
	// Documents the locked design decision: CorrectCostUseCase only ever
	// writes the SKU's current cost + a forward-dated ledger entry. It has
	// no dependency on omsDomain.SalesOrderLineItemRepository at all --
	// there is nothing here that could touch a frozen cost_at_sale.
	existingAvg := int64(300)
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &existingAvg}}
	batchLister := &stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 10}}}
	ledgerRepo := &stubCostLedgerRepo{}
	uc := apppms.NewCorrectCostUseCase(newMockGormDB(t), skuRepo, ledgerRepo, batchLister)

	_, err := uc.Execute(context.Background(), apppms.CorrectCostInput{SKUID: "sku-1", AdjustmentValueMinorUnits: 100, OccurredAt: time.Now()})

	require.NoError(t, err)
}
