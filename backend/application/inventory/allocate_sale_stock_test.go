package inventory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func strPtrAlloc(s string) *string { return &s }

func TestAllocateSaleStock_rejectsNonPositiveQuantity(t *testing.T) {
	uc := appinv.NewAllocateSaleStockUseCase(newMockGormDB(t), &stubStockMovementRepo{}, &stubBatchRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.AllocateSaleStockInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeSale, Quantity: 0, OccurredAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInvalidQuantity))
}

func TestAllocateSaleStock_rejectsNonStockOutType(t *testing.T) {
	uc := appinv.NewAllocateSaleStockUseCase(newMockGormDB(t), &stubStockMovementRepo{}, &stubBatchRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.AllocateSaleStockInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeReceipt, Quantity: 5, OccurredAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInvalidQuantity))
}

func TestAllocateSaleStock_fallsBackToGeneralLedgerWhenNoBatchesExist(t *testing.T) {
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{WarehouseID: "wh-1", BatchID: nil, QuantityDelta: 10},
	}}
	uc := appinv.NewAllocateSaleStockUseCase(newMockGormDB(t), movementRepo, &stubBatchRepo{}, &stubOutboxRepo{})

	movements, err := uc.Execute(context.Background(), appinv.AllocateSaleStockInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeSale, Quantity: 3, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Len(t, movements, 1)
	require.Nil(t, movements[0].BatchID)
	require.Equal(t, int64(-3), movements[0].QuantityDelta)
}

func TestAllocateSaleStock_drawsFromARealBatchWhenGeneralLedgerIsEmpty(t *testing.T) {
	// Reproduces the production bug: general ledger is 0, but a real batch
	// has plenty -- the old single-scope check would reject this.
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{
		{ID: "batch-1", WarehouseID: "wh-1", Status: invDomain.BatchStatusActive, ReceivedAt: time.Now()},
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{WarehouseID: "wh-1", BatchID: nil, QuantityDelta: 0},
		{WarehouseID: "wh-1", BatchID: strPtrAlloc("batch-1"), QuantityDelta: 48},
	}}
	uc := appinv.NewAllocateSaleStockUseCase(newMockGormDB(t), movementRepo, batchRepo, &stubOutboxRepo{})

	movements, err := uc.Execute(context.Background(), appinv.AllocateSaleStockInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeSale, Quantity: 3, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Len(t, movements, 1)
	require.NotNil(t, movements[0].BatchID)
	require.Equal(t, "batch-1", *movements[0].BatchID)
	require.Equal(t, int64(-3), movements[0].QuantityDelta)
}

func TestAllocateSaleStock_picksSoonestExpiringBatchFirst(t *testing.T) {
	soon := time.Now().Add(24 * time.Hour)
	later := time.Now().Add(24 * 30 * time.Hour)
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{
		{ID: "batch-later", WarehouseID: "wh-1", Status: invDomain.BatchStatusActive, ExpiryDate: &later, ReceivedAt: time.Now()},
		{ID: "batch-soon", WarehouseID: "wh-1", Status: invDomain.BatchStatusActive, ExpiryDate: &soon, ReceivedAt: time.Now()},
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{WarehouseID: "wh-1", BatchID: strPtrAlloc("batch-later"), QuantityDelta: 10},
		{WarehouseID: "wh-1", BatchID: strPtrAlloc("batch-soon"), QuantityDelta: 10},
	}}
	uc := appinv.NewAllocateSaleStockUseCase(newMockGormDB(t), movementRepo, batchRepo, &stubOutboxRepo{})

	movements, err := uc.Execute(context.Background(), appinv.AllocateSaleStockInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeSale, Quantity: 5, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Len(t, movements, 1)
	require.Equal(t, "batch-soon", *movements[0].BatchID)
}

func TestAllocateSaleStock_fallsBackToFIFOWhenNoBatchHasExpiry(t *testing.T) {
	older := time.Now().Add(-48 * time.Hour)
	newer := time.Now().Add(-1 * time.Hour)
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{
		{ID: "batch-newer", WarehouseID: "wh-1", Status: invDomain.BatchStatusActive, ReceivedAt: newer},
		{ID: "batch-older", WarehouseID: "wh-1", Status: invDomain.BatchStatusActive, ReceivedAt: older},
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{WarehouseID: "wh-1", BatchID: strPtrAlloc("batch-newer"), QuantityDelta: 10},
		{WarehouseID: "wh-1", BatchID: strPtrAlloc("batch-older"), QuantityDelta: 10},
	}}
	uc := appinv.NewAllocateSaleStockUseCase(newMockGormDB(t), movementRepo, batchRepo, &stubOutboxRepo{})

	movements, err := uc.Execute(context.Background(), appinv.AllocateSaleStockInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeSale, Quantity: 5, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Len(t, movements, 1)
	require.Equal(t, "batch-older", *movements[0].BatchID)
}

func TestAllocateSaleStock_excludesExpiredStatusBatches(t *testing.T) {
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{
		{ID: "batch-expired", WarehouseID: "wh-1", Status: invDomain.BatchStatusExpired, ReceivedAt: time.Now()},
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{WarehouseID: "wh-1", BatchID: strPtrAlloc("batch-expired"), QuantityDelta: 10},
	}}
	uc := appinv.NewAllocateSaleStockUseCase(newMockGormDB(t), movementRepo, batchRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.AllocateSaleStockInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeSale, Quantity: 3, OccurredAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInsufficientStock))
}

func TestAllocateSaleStock_splitsAcrossBatchThenGeneralLedgerWhenOneAlone_isNotEnough(t *testing.T) {
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{
		{ID: "batch-1", WarehouseID: "wh-1", Status: invDomain.BatchStatusActive, ReceivedAt: time.Now()},
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{WarehouseID: "wh-1", BatchID: strPtrAlloc("batch-1"), QuantityDelta: 2},
		{WarehouseID: "wh-1", BatchID: nil, QuantityDelta: 5},
	}}
	uc := appinv.NewAllocateSaleStockUseCase(newMockGormDB(t), movementRepo, batchRepo, &stubOutboxRepo{})

	movements, err := uc.Execute(context.Background(), appinv.AllocateSaleStockInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeSale, Quantity: 4, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Len(t, movements, 2)
	require.Equal(t, "batch-1", *movements[0].BatchID)
	require.Equal(t, int64(-2), movements[0].QuantityDelta)
	require.Nil(t, movements[1].BatchID)
	require.Equal(t, int64(-2), movements[1].QuantityDelta)
}

func TestAllocateSaleStock_trueInsufficiency_writesNoMovementsAtAll(t *testing.T) {
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{
		{ID: "batch-1", WarehouseID: "wh-1", Status: invDomain.BatchStatusActive, ReceivedAt: time.Now()},
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{WarehouseID: "wh-1", BatchID: strPtrAlloc("batch-1"), QuantityDelta: 2},
	}}
	uc := appinv.NewAllocateSaleStockUseCase(newMockGormDB(t), movementRepo, batchRepo, &stubOutboxRepo{})

	movements, err := uc.Execute(context.Background(), appinv.AllocateSaleStockInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeSale, Quantity: 100, OccurredAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInsufficientStock))
	require.Nil(t, movements)
	require.Empty(t, movementRepo.created)
}

func TestAllocateSaleStock_reconcileShortfallWritesAdjustmentThenSucceeds(t *testing.T) {
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{WarehouseID: "wh-1", BatchID: nil, QuantityDelta: 1},
	}}
	uc := appinv.NewAllocateSaleStockUseCase(newMockGormDB(t), movementRepo, &stubBatchRepo{}, &stubOutboxRepo{})

	movements, err := uc.Execute(context.Background(), appinv.AllocateSaleStockInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeSale, Quantity: 5, OccurredAt: time.Now(),
		ReconcileShortfall: true,
	})

	require.NoError(t, err)
	// One sale movement against what was genuinely on the general ledger
	// (-1), one against the reconciled remainder (-4) -- the adjustment
	// itself is written but not included in the returned sale movements.
	require.Len(t, movements, 2)
	require.Nil(t, movements[0].BatchID)
	require.Equal(t, int64(-1), movements[0].QuantityDelta)
	require.Nil(t, movements[1].BatchID)
	require.Equal(t, int64(-4), movements[1].QuantityDelta)

	require.Len(t, movementRepo.created, 3)
	require.Equal(t, invDomain.MovementTypeAdjustment, movementRepo.created[0].Type)
	require.Equal(t, int64(4), movementRepo.created[0].QuantityDelta, "shortfall = needed 5 - the 1 already on the ledger")
	require.Equal(t, invDomain.MovementTypeSale, movementRepo.created[1].Type)
	require.Equal(t, invDomain.MovementTypeSale, movementRepo.created[2].Type)
}

func TestAllocateSaleStock_withoutReconcileShortfall_stillFailsOnTrueInsufficiency(t *testing.T) {
	uc := appinv.NewAllocateSaleStockUseCase(newMockGormDB(t), &stubStockMovementRepo{}, &stubBatchRepo{}, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.AllocateSaleStockInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeSale, Quantity: 5, OccurredAt: time.Now(),
		ReconcileShortfall: false,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInsufficientStock))
}

func TestAllocateSaleStock_ignoresBatchesInADifferentWarehouse(t *testing.T) {
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{
		{ID: "batch-other-wh", WarehouseID: "wh-2", Status: invDomain.BatchStatusActive, ReceivedAt: time.Now()},
	}}
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{WarehouseID: "wh-2", BatchID: strPtrAlloc("batch-other-wh"), QuantityDelta: 50},
	}}
	uc := appinv.NewAllocateSaleStockUseCase(newMockGormDB(t), movementRepo, batchRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.AllocateSaleStockInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeSale, Quantity: 3, OccurredAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInsufficientStock))
}
