package inventory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReceiveStock_createsNewBatchAndReceiptMovement(t *testing.T) {
	db := newMockGormDB(t)
	batchRepo := &stubBatchRepo{}
	movementRepo := &stubStockMovementRepo{}
	outboxRepo := &stubOutboxRepo{}
	uc := appinventory.NewReceiveStockUseCase(db, batchRepo, movementRepo, outboxRepo, nil)

	result, err := uc.Execute(context.Background(), appinventory.ReceiveStockInput{
		SKUID: "sku-1", WarehouseID: "warehouse-1", Quantity: 10,
		LotNumber: "LOT-1", Reason: "receiving invoice-1",
	})

	require.NoError(t, err)
	require.True(t, batchRepo.createCalled)
	require.NotNil(t, result.Batch)
	assert.Equal(t, "batch-1", result.Batch.ID)
	require.Len(t, movementRepo.created, 1)
	assert.Equal(t, invDomain.MovementTypeReceipt, movementRepo.created[0].Type)
	assert.Equal(t, int64(10), movementRepo.created[0].QuantityDelta)
	require.NotNil(t, movementRepo.created[0].BatchID)
	assert.Equal(t, "batch-1", *movementRepo.created[0].BatchID)
	assert.Equal(t, []string{"inventory.stock.changed"}, outboxRepo.calls)
}

func TestReceiveStock_targetsExistingBatchWithoutCreatingANewOne(t *testing.T) {
	db := newMockGormDB(t)
	batchRepo := &stubBatchRepo{}
	movementRepo := &stubStockMovementRepo{}
	outboxRepo := &stubOutboxRepo{}
	uc := appinventory.NewReceiveStockUseCase(db, batchRepo, movementRepo, outboxRepo, nil)

	existingBatchID := "existing-batch-1"
	result, err := uc.Execute(context.Background(), appinventory.ReceiveStockInput{
		SKUID: "sku-1", WarehouseID: "warehouse-1", Quantity: 5, BatchID: &existingBatchID,
	})

	require.NoError(t, err)
	assert.False(t, batchRepo.createCalled)
	assert.Nil(t, result.Batch)
	require.Len(t, movementRepo.created, 1)
	require.NotNil(t, movementRepo.created[0].BatchID)
	assert.Equal(t, existingBatchID, *movementRepo.created[0].BatchID)
}

func TestReceiveStock_rejectsNonPositiveQuantity(t *testing.T) {
	db := newMockGormDB(t)
	uc := appinventory.NewReceiveStockUseCase(db, &stubBatchRepo{}, &stubStockMovementRepo{}, &stubOutboxRepo{}, nil)

	_, err := uc.Execute(context.Background(), appinventory.ReceiveStockInput{SKUID: "sku-1", WarehouseID: "warehouse-1", Quantity: 0})

	require.ErrorIs(t, err, invDomain.ErrInvalidQuantity)
}

func TestReceiveStock_propagatesBatchCreateError(t *testing.T) {
	db := newMockGormDB(t)
	batchRepo := &stubBatchRepo{createErr: errStub}
	uc := appinventory.NewReceiveStockUseCase(db, batchRepo, &stubStockMovementRepo{}, &stubOutboxRepo{}, nil)

	_, err := uc.Execute(context.Background(), appinventory.ReceiveStockInput{SKUID: "sku-1", WarehouseID: "warehouse-1", Quantity: 5})

	require.Error(t, err)
}

func TestReceiveStock_propagatesMovementCreateError(t *testing.T) {
	db := newMockGormDB(t)
	movementRepo := &stubStockMovementRepo{createErr: errStub}
	existingBatchID := "existing-batch-1"
	uc := appinventory.NewReceiveStockUseCase(db, &stubBatchRepo{}, movementRepo, &stubOutboxRepo{}, nil)

	_, err := uc.Execute(context.Background(), appinventory.ReceiveStockInput{
		SKUID: "sku-1", WarehouseID: "warehouse-1", Quantity: 5, BatchID: &existingBatchID,
	})

	require.Error(t, err)
}

func TestReceiveStock_setsExpiryDateOnNewBatch(t *testing.T) {
	db := newMockGormDB(t)
	batchRepo := &stubBatchRepo{}
	uc := appinventory.NewReceiveStockUseCase(db, batchRepo, &stubStockMovementRepo{}, &stubOutboxRepo{}, nil)

	expiry := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	result, err := uc.Execute(context.Background(), appinventory.ReceiveStockInput{
		SKUID: "sku-1", WarehouseID: "warehouse-1", Quantity: 5, ExpiryDate: &expiry,
	})

	require.NoError(t, err)
	require.NotNil(t, result.Batch)
	require.NotNil(t, result.Batch.ExpiryDate)
	assert.Equal(t, expiry, *result.Batch.ExpiryDate)
}

func TestReceiveStock_persistsLandedCostAndPurchaseOrderLineItemIDOnNewBatch(t *testing.T) {
	db := newMockGormDB(t)
	batchRepo := &stubBatchRepo{}
	costUpdater := &stubSKUCostUpdater{}
	uc := appinventory.NewReceiveStockUseCase(db, batchRepo, &stubStockMovementRepo{}, &stubOutboxRepo{}, costUpdater)

	lineItemID := "line-item-1"
	unitCost := int64(1250)
	result, err := uc.Execute(context.Background(), appinventory.ReceiveStockInput{
		SKUID: "sku-1", WarehouseID: "warehouse-1", Quantity: 5,
		PurchaseOrderLineItemID: &lineItemID, UnitCostMinorUnits: &unitCost,
	})

	require.NoError(t, err)
	require.NotNil(t, result.Batch)
	require.NotNil(t, result.Batch.PurchaseOrderLineItemID)
	assert.Equal(t, lineItemID, *result.Batch.PurchaseOrderLineItemID)
	require.NotNil(t, result.Batch.LandedCostMinorUnits)
	assert.Equal(t, unitCost, *result.Batch.LandedCostMinorUnits)

	// A costed receipt against a known PO line item must also drive the
	// MWAC hook, referencing that line item (see
	// skuCostUpdaterForReceiveStock's doc comment).
	require.Len(t, costUpdater.calls, 1)
	call := costUpdater.calls[0]
	assert.Equal(t, "sku-1", call.SKUID)
	assert.Equal(t, int64(5), call.ReceivedQuantity)
	assert.Equal(t, unitCost, call.LandedUnitCostMinorUnits)
	assert.Equal(t, "purchase_order_line_item", call.ReferenceType)
	assert.Equal(t, lineItemID, call.ReferenceID)
}

func TestReceiveStock_costedPOLineReceiptRaisesGoodsReceivedEvent(t *testing.T) {
	db := newMockGormDB(t)
	outboxRepo := &stubOutboxRepo{}
	uc := appinventory.NewReceiveStockUseCase(db, &stubBatchRepo{}, &stubStockMovementRepo{}, outboxRepo, nil)

	lineItemID := "line-item-1"
	unitCost := int64(1250)
	_, err := uc.Execute(context.Background(), appinventory.ReceiveStockInput{
		SKUID: "sku-1", WarehouseID: "warehouse-1", Quantity: 5,
		PurchaseOrderLineItemID: &lineItemID, UnitCostMinorUnits: &unitCost,
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"procurement.goods.received", "inventory.stock.changed"}, outboxRepo.calls)
}

func TestReceiveStock_manualStockInWithNoPOLineDoesNotRaiseGoodsReceivedEvent(t *testing.T) {
	db := newMockGormDB(t)
	outboxRepo := &stubOutboxRepo{}
	uc := appinventory.NewReceiveStockUseCase(db, &stubBatchRepo{}, &stubStockMovementRepo{}, outboxRepo, nil)

	_, err := uc.Execute(context.Background(), appinventory.ReceiveStockInput{
		SKUID: "sku-1", WarehouseID: "warehouse-1", Quantity: 5,
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"inventory.stock.changed"}, outboxRepo.calls)
}

func TestReceiveStock_costUpdaterNotCalledWhenNoUnitCostSupplied(t *testing.T) {
	db := newMockGormDB(t)
	costUpdater := &stubSKUCostUpdater{}
	uc := appinventory.NewReceiveStockUseCase(db, &stubBatchRepo{}, &stubStockMovementRepo{}, &stubOutboxRepo{}, costUpdater)

	_, err := uc.Execute(context.Background(), appinventory.ReceiveStockInput{SKUID: "sku-1", WarehouseID: "warehouse-1", Quantity: 5})

	require.NoError(t, err)
	assert.Empty(t, costUpdater.calls)
}

func TestReceiveStock_costUpdaterReferencesBatchWhenNoLineItemID(t *testing.T) {
	db := newMockGormDB(t)
	costUpdater := &stubSKUCostUpdater{}
	uc := appinventory.NewReceiveStockUseCase(db, &stubBatchRepo{}, &stubStockMovementRepo{}, &stubOutboxRepo{}, costUpdater)

	unitCost := int64(900)
	result, err := uc.Execute(context.Background(), appinventory.ReceiveStockInput{
		SKUID: "sku-1", WarehouseID: "warehouse-1", Quantity: 3, UnitCostMinorUnits: &unitCost,
	})

	require.NoError(t, err)
	require.Len(t, costUpdater.calls, 1)
	assert.Equal(t, "batch", costUpdater.calls[0].ReferenceType)
	assert.Equal(t, result.Batch.ID, costUpdater.calls[0].ReferenceID)
}

func TestReceiveStock_propagatesCostUpdaterError(t *testing.T) {
	db := newMockGormDB(t)
	costUpdater := &stubSKUCostUpdater{err: errStub}
	uc := appinventory.NewReceiveStockUseCase(db, &stubBatchRepo{}, &stubStockMovementRepo{}, &stubOutboxRepo{}, costUpdater)

	unitCost := int64(900)
	_, err := uc.Execute(context.Background(), appinventory.ReceiveStockInput{
		SKUID: "sku-1", WarehouseID: "warehouse-1", Quantity: 3, UnitCostMinorUnits: &unitCost,
	})

	require.Error(t, err)
}

func TestReceiveStock_leavesLandedCostNilWhenNotSupplied(t *testing.T) {
	db := newMockGormDB(t)
	batchRepo := &stubBatchRepo{}
	uc := appinventory.NewReceiveStockUseCase(db, batchRepo, &stubStockMovementRepo{}, &stubOutboxRepo{}, nil)

	result, err := uc.Execute(context.Background(), appinventory.ReceiveStockInput{
		SKUID: "sku-1", WarehouseID: "warehouse-1", Quantity: 5,
	})

	require.NoError(t, err)
	require.NotNil(t, result.Batch)
	assert.Nil(t, result.Batch.PurchaseOrderLineItemID)
	assert.Nil(t, result.Batch.LandedCostMinorUnits)
}

var errStub = errors.New("stub error")
