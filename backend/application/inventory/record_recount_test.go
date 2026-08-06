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

func TestRecordRecount_rejectsNegativeCountedQuantity(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewRecordRecountUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.RecordRecountInput{
		SKUID: "sku-1", WarehouseID: "wh-1", CountedQuantity: -1, OccurredAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInvalidQuantity))
	require.Empty(t, movementRepo.created)
}

func TestRecordRecount_createsAdjustmentForBatchScopedDifference(t *testing.T) {
	movementRepo := &stubStockMovementRepo{byBatch: []invDomain.StockMovement{
		{BatchID: strPtr("batch-1"), QuantityDelta: 10, Type: invDomain.MovementTypeReceipt},
	}}
	uc := appinv.NewRecordRecountUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	m, err := uc.Execute(context.Background(), appinv.RecordRecountInput{
		SKUID: "sku-1", WarehouseID: "wh-1", BatchID: strPtr("batch-1"), CountedQuantity: 7, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.NotNil(t, m)
	require.Equal(t, int64(-3), m.QuantityDelta)
	require.Equal(t, invDomain.MovementTypeAdjustment, m.Type)
}

func TestRecordRecount_createsAdjustmentForUnlottedDifference(t *testing.T) {
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", BatchID: nil, QuantityDelta: 5, Type: invDomain.MovementTypeReceipt},
		{SKUID: "sku-1", BatchID: strPtr("other-batch"), QuantityDelta: 100, Type: invDomain.MovementTypeReceipt},
	}}
	uc := appinv.NewRecordRecountUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	m, err := uc.Execute(context.Background(), appinv.RecordRecountInput{
		SKUID: "sku-1", WarehouseID: "wh-1", BatchID: nil, CountedQuantity: 8, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.NotNil(t, m)
	require.Equal(t, int64(3), m.QuantityDelta, "must only compare against un-lotted stock, ignoring the other batch's 100 units")
}

func TestRecordRecount_noMovementWhenCountMatchesCurrent(t *testing.T) {
	movementRepo := &stubStockMovementRepo{byBatch: []invDomain.StockMovement{
		{BatchID: strPtr("batch-1"), QuantityDelta: 10, Type: invDomain.MovementTypeReceipt},
	}}
	outboxRepo := &stubOutboxRepo{}
	uc := appinv.NewRecordRecountUseCase(newMockGormDB(t), movementRepo, outboxRepo)

	m, err := uc.Execute(context.Background(), appinv.RecordRecountInput{
		SKUID: "sku-1", WarehouseID: "wh-1", BatchID: strPtr("batch-1"), CountedQuantity: 10, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Nil(t, m)
	require.Empty(t, movementRepo.created)
	require.Empty(t, outboxRepo.calls, "no movement means nothing to sync")
}

func TestRecordRecount_raisesStockChangedEventWhenAdjusted(t *testing.T) {
	movementRepo := &stubStockMovementRepo{byBatch: []invDomain.StockMovement{
		{BatchID: strPtr("batch-1"), QuantityDelta: 10, Type: invDomain.MovementTypeReceipt},
	}}
	outboxRepo := &stubOutboxRepo{}
	uc := appinv.NewRecordRecountUseCase(newMockGormDB(t), movementRepo, outboxRepo)

	_, err := uc.Execute(context.Background(), appinv.RecordRecountInput{
		SKUID: "sku-1", WarehouseID: "wh-1", BatchID: strPtr("batch-1"), CountedQuantity: 7, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Equal(t, []string{"inventory.stock.changed"}, outboxRepo.calls)
}
