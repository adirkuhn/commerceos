package pms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordCostLedgerMovement_writesEntryAtTheSKUsCurrentCostWithoutChangingIt(t *testing.T) {
	cost := int64(440)
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: &cost}}
	batchLister := &stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 95}}} // after a -5 damaged movement
	ledgerRepo := &stubCostLedgerRepo{}
	occurredAt := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	uc := apppms.NewRecordCostLedgerMovementUseCase(newMockGormDB(t), skuRepo, ledgerRepo, batchLister)

	err := uc.Execute(context.Background(), apppms.RecordCostLedgerMovementInput{
		SKUID: "sku-1", QuantityDelta: -5, SourceType: pmsDomain.SKUCostPriceLedgerSourceDamaged,
		ReferenceType: "stock_movement", ReferenceID: "movement-1", OccurredAt: occurredAt,
	})

	require.NoError(t, err)
	assert.Empty(t, skuRepo.updateCalls, "must never change cost_price_minor_units")
	require.Len(t, ledgerRepo.created, 1)
	entry := ledgerRepo.created[0]
	assert.Equal(t, pmsDomain.SKUCostPriceLedgerSourceDamaged, entry.SourceType)
	assert.Equal(t, int64(-5), entry.QuantityDelta)
	assert.Equal(t, int64(95), entry.QuantityAfter)
	require.NotNil(t, entry.CostPriceMinorUnits)
	assert.Equal(t, int64(440), *entry.CostPriceMinorUnits)
	require.NotNil(t, entry.InventoryValueMinorUnits)
	assert.Equal(t, int64(95*440), *entry.InventoryValueMinorUnits)
	assert.Equal(t, occurredAt, entry.OccurredAt)
}

func TestRecordCostLedgerMovement_leavesCostAndValueNilWhenSKUCostIsUnknown(t *testing.T) {
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1", CostPriceMinorUnits: nil}}
	batchLister := &stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 10}}}
	ledgerRepo := &stubCostLedgerRepo{}
	uc := apppms.NewRecordCostLedgerMovementUseCase(newMockGormDB(t), skuRepo, ledgerRepo, batchLister)

	err := uc.Execute(context.Background(), apppms.RecordCostLedgerMovementInput{
		SKUID: "sku-1", QuantityDelta: 2, SourceType: pmsDomain.SKUCostPriceLedgerSourceAdjustment, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Len(t, ledgerRepo.created, 1)
	assert.Nil(t, ledgerRepo.created[0].CostPriceMinorUnits)
	assert.Nil(t, ledgerRepo.created[0].InventoryValueMinorUnits)
}

func TestRecordCostLedgerMovement_rejectsZeroQuantityDelta(t *testing.T) {
	uc := apppms.NewRecordCostLedgerMovementUseCase(newMockGormDB(t), &stubSKURepoForReceiptCost{}, &stubCostLedgerRepo{}, &stubBatchListerForReceiptCost{})

	err := uc.Execute(context.Background(), apppms.RecordCostLedgerMovementInput{SKUID: "sku-1", QuantityDelta: 0, OccurredAt: time.Now()})

	require.Error(t, err)
}

func TestRecordCostLedgerMovement_propagatesLedgerCreateError(t *testing.T) {
	skuRepo := &stubSKURepoForReceiptCost{findByIDResult: &pmsDomain.SKU{ID: "sku-1"}}
	batchLister := &stubBatchListerForReceiptCost{batches: []invDomain.Batch{{Quantity: 1}}}
	ledgerRepo := &stubCostLedgerRepo{err: errors.New("db down")}
	uc := apppms.NewRecordCostLedgerMovementUseCase(newMockGormDB(t), skuRepo, ledgerRepo, batchLister)

	err := uc.Execute(context.Background(), apppms.RecordCostLedgerMovementInput{SKUID: "sku-1", QuantityDelta: -1, OccurredAt: time.Now()})

	require.Error(t, err)
}
