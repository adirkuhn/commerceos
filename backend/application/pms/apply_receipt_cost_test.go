package pms_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"
	"time"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSKURepoForReceiptCost struct {
	findByIDResult *pmsDomain.SKU
	findByIDErr    error
	updateCalls    []struct {
		skuID               string
		costPriceMinorUnits int64
	}
	inventoryCalls []int64
	updateErr      error
}

func (s *stubSKURepoForReceiptCost) FindByID(_ context.Context, _ string) (*pmsDomain.SKU, error) {
	return s.findByIDResult, s.findByIDErr
}

func (s *stubSKURepoForReceiptCost) UpdateCostPriceFromReceipt(_ context.Context, skuID string, costPriceMinorUnits int64) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updateCalls = append(s.updateCalls, struct {
		skuID               string
		costPriceMinorUnits int64
	}{skuID, costPriceMinorUnits})
	return nil
}

func (s *stubSKURepoForReceiptCost) UpdateInventoryCost(_ context.Context, _ string, cost int64) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.inventoryCalls = append(s.inventoryCalls, cost)
	return nil
}

type stubBatchListerForReceiptCost struct {
	batches []invDomain.Batch
	err     error
}

func (s *stubBatchListerForReceiptCost) Execute(_ context.Context, _ string) ([]invDomain.Batch, error) {
	return s.batches, s.err
}

type stubCostLedgerRepo struct {
	created []pmsDomain.SKUCostPriceLedgerEntry
	err     error
}

func (s *stubCostLedgerRepo) Create(_ context.Context, e *pmsDomain.SKUCostPriceLedgerEntry) error {
	if s.err != nil {
		return s.err
	}
	e.ID = "ledger-1"
	s.created = append(s.created, *e)
	return nil
}

func (s *stubCostLedgerRepo) ListBySKU(_ context.Context, _ string, _ int) ([]pmsDomain.SKUCostPriceLedgerEntry, error) {
	return nil, nil
}

func TestApplyReceiptCost_blendsInventoryCostAndSetsCostPriceToTheLastReceipt(t *testing.T) {
	existingAvg, lastCost := int64(440), int64(450)
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &lastCost, InventoryCostMinorUnits: &existingAvg}}
	// batches already reflect the receipt having committed: 100 existing + 50 just received = 150.
	batchLister := &stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 150}}}
	ledgerRepo := &stubCostLedgerRepo{}
	outbox := &eventrecordtest.Outbox{}
	occurredAt := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	uc := apppms.NewApplyReceiptCostUseCase(newMockGormDB(t), skuRepo, ledgerRepo, batchLister, outbox)

	err := uc.Execute(context.Background(), apppms.ApplyReceiptCostInput{
		SKUID: "sku-1", ReceivedQuantity: 50, LandedUnitCostMinorUnits: 520,
		ReferenceType: apppms.ReceiptReferencePurchaseOrderLine, ReferenceID: "line-1", OccurredAt: occurredAt,
	})

	require.NoError(t, err)
	assert.Equal(t, []int64{467}, skuRepo.inventoryCalls, "moving average of the stock on hand")
	require.Len(t, skuRepo.updateCalls, 1)
	assert.Equal(t, int64(520), skuRepo.updateCalls[0].costPriceMinorUnits, "cost price is the last receipt's landed cost")
	require.Len(t, ledgerRepo.created, 1)
	entry := ledgerRepo.created[0]
	assert.Equal(t, pmsDomain.SKUCostPriceLedgerSourceReceipt, entry.SourceType)
	assert.Equal(t, int64(50), entry.QuantityDelta)
	assert.Equal(t, int64(150), entry.QuantityAfter)
	require.NotNil(t, entry.CostPriceMinorUnits)
	assert.Equal(t, int64(467), *entry.CostPriceMinorUnits, "the ledger keeps the moving average")
	require.NotNil(t, entry.InventoryValueMinorUnits)
	assert.Equal(t, int64(150*467), *entry.InventoryValueMinorUnits)
	require.NotNil(t, entry.ReferenceType)
	assert.Equal(t, "purchase_order_line_item", *entry.ReferenceType)
	require.NotNil(t, entry.ReferenceID)
	assert.Equal(t, "line-1", *entry.ReferenceID)
	assert.Equal(t, occurredAt, entry.OccurredAt)
	assert.ElementsMatch(t, []string{"pms.sku.inventory_cost_changed", "pms.sku.cost_price_changed"}, outbox.Subjects())
}

func TestApplyReceiptCost_withoutAnAverageYetBlendsFromTheCostPrice(t *testing.T) {
	cost := int64(440)
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &cost}}
	batchLister := &stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 150}}}
	uc := apppms.NewApplyReceiptCostUseCase(newMockGormDB(t), skuRepo, &stubCostLedgerRepo{}, batchLister, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), apppms.ApplyReceiptCostInput{
		SKUID: "sku-1", ReceivedQuantity: 50, LandedUnitCostMinorUnits: 520,
		ReferenceType: apppms.ReceiptReferencePurchaseOrderLine, ReferenceID: "line-1", OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	assert.Equal(t, []int64{467}, skuRepo.inventoryCalls)
}

func TestApplyReceiptCost_unknownCostIsFilledByACostedReceiptWithoutPurchaseOrder(t *testing.T) {
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceSource: pmsDomain.SKUCostPriceSourceUnknown}}
	batchLister := &stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 150}}}
	uc := apppms.NewApplyReceiptCostUseCase(newMockGormDB(t), skuRepo, &stubCostLedgerRepo{}, batchLister, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), apppms.ApplyReceiptCostInput{
		SKUID: "sku-1", ReceivedQuantity: 50, LandedUnitCostMinorUnits: 520,
		ReferenceType: "batch", ReferenceID: "batch-1", OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	assert.Equal(t, []int64{520}, skuRepo.inventoryCalls)
	require.Len(t, skuRepo.updateCalls, 1)
	assert.Equal(t, int64(520), skuRepo.updateCalls[0].costPriceMinorUnits)
}

func TestApplyReceiptCost_receiptWithoutPurchaseOrderKeepsAKnownCostPrice(t *testing.T) {
	cost, avg := int64(450), int64(440)
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &cost, InventoryCostMinorUnits: &avg}}
	batchLister := &stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 150}}}
	outbox := &eventrecordtest.Outbox{}
	uc := apppms.NewApplyReceiptCostUseCase(newMockGormDB(t), skuRepo, &stubCostLedgerRepo{}, batchLister, outbox)

	err := uc.Execute(context.Background(), apppms.ApplyReceiptCostInput{
		SKUID: "sku-1", ReceivedQuantity: 50, LandedUnitCostMinorUnits: 520,
		ReferenceType: "batch", ReferenceID: "batch-1", OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	assert.Equal(t, []int64{467}, skuRepo.inventoryCalls)
	assert.Empty(t, skuRepo.updateCalls)
	assert.Equal(t, []string{"pms.sku.inventory_cost_changed"}, outbox.Subjects())
}

func TestApplyReceiptCost_manualCostPriceStaysButTheAverageStillMoves(t *testing.T) {
	manual, avg := int64(900), int64(440)
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &manual,
		InventoryCostMinorUnits: &avg, CostPriceSource: pmsDomain.SKUCostPriceSourceManual}}
	batchLister := &stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 150}}}
	ledgerRepo := &stubCostLedgerRepo{}
	uc := apppms.NewApplyReceiptCostUseCase(newMockGormDB(t), skuRepo, ledgerRepo, batchLister, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), apppms.ApplyReceiptCostInput{
		SKUID: "sku-1", ReceivedQuantity: 50, LandedUnitCostMinorUnits: 520,
		ReferenceType: apppms.ReceiptReferencePurchaseOrderLine, ReferenceID: "line-1", OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	assert.Empty(t, skuRepo.updateCalls, "a manual cost price wins")
	assert.Equal(t, []int64{467}, skuRepo.inventoryCalls, "COGS still follows what the stock actually cost")
	require.Len(t, ledgerRepo.created, 1)
}

func TestApplyReceiptCost_zeroCostReceiptLowersTheAverageButNotTheCostPrice(t *testing.T) {
	cost, avg := int64(500), int64(500)
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &cost, InventoryCostMinorUnits: &avg}}
	batchLister := &stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 100}}}
	uc := apppms.NewApplyReceiptCostUseCase(newMockGormDB(t), skuRepo, &stubCostLedgerRepo{}, batchLister, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), apppms.ApplyReceiptCostInput{
		SKUID: "sku-1", ReceivedQuantity: 50, LandedUnitCostMinorUnits: 0,
		ReferenceType: apppms.ReceiptReferencePurchaseOrderLine, ReferenceID: "line-1", OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	assert.Equal(t, []int64{250}, skuRepo.inventoryCalls)
	assert.Empty(t, skuRepo.updateCalls)
}

func TestApplyReceiptCost_rejectsNonPositiveReceivedQuantity(t *testing.T) {
	uc := apppms.NewApplyReceiptCostUseCase(newMockGormDB(t), &stubSKURepoForReceiptCost{}, &stubCostLedgerRepo{}, &stubBatchListerForReceiptCost{}, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), apppms.ApplyReceiptCostInput{SKUID: "sku-1", ReceivedQuantity: 0, LandedUnitCostMinorUnits: 100})

	require.Error(t, err)
}

func TestApplyReceiptCost_propagatesLedgerCreateError(t *testing.T) {
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1"}}
	batchLister := &stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 50}}}
	ledgerRepo := &stubCostLedgerRepo{err: errors.New("db down")}
	uc := apppms.NewApplyReceiptCostUseCase(newMockGormDB(t), skuRepo, ledgerRepo, batchLister, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), apppms.ApplyReceiptCostInput{SKUID: "sku-1", ReceivedQuantity: 50, LandedUnitCostMinorUnits: 100, OccurredAt: time.Now()})

	require.Error(t, err)
}
