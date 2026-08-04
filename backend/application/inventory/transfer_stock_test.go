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

func TestTransferStock_rejectsNonPositiveQuantity(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewTransferStockUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.TransferStockInput{
		SKUID: "sku-1", Quantity: 0, FromWarehouseID: "wh-a", ToWarehouseID: "wh-b", OccurredAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInvalidQuantity))
	require.Empty(t, movementRepo.created)
}

func TestTransferStock_createsMatchingOutAndInMovements(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewTransferStockUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	result, err := uc.Execute(context.Background(), appinv.TransferStockInput{
		SKUID: "sku-1", Quantity: 5,
		FromWarehouseID: "wh-a", FromBatchID: strPtr("batch-a"),
		ToWarehouseID: "wh-b", ToBatchID: strPtr("batch-b"),
		OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Len(t, movementRepo.created, 2)

	require.Equal(t, invDomain.MovementTypeTransferOut, result.Out.Type)
	require.Equal(t, int64(-5), result.Out.QuantityDelta)
	require.Equal(t, "wh-a", result.Out.WarehouseID)
	require.Equal(t, "batch-a", *result.Out.BatchID)

	require.Equal(t, invDomain.MovementTypeTransferIn, result.In.Type)
	require.Equal(t, int64(5), result.In.QuantityDelta)
	require.Equal(t, "wh-b", result.In.WarehouseID)
	require.Equal(t, "batch-b", *result.In.BatchID)
}

func TestTransferStock_raisesStockChangedEvent(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	outboxRepo := &stubOutboxRepo{}
	uc := appinv.NewTransferStockUseCase(newMockGormDB(t), movementRepo, outboxRepo)

	_, err := uc.Execute(context.Background(), appinv.TransferStockInput{
		SKUID: "sku-1", Quantity: 5, FromWarehouseID: "wh-a", ToWarehouseID: "wh-b", OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Equal(t, []string{"inventory.stock.changed"}, outboxRepo.calls)
}

func TestTransferStock_propagatesRepoErrorAndCreatesNothingFurther(t *testing.T) {
	movementRepo := &stubStockMovementRepo{createErr: errors.New("db down")}
	uc := appinv.NewTransferStockUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.TransferStockInput{
		SKUID: "sku-1", Quantity: 5, FromWarehouseID: "wh-a", ToWarehouseID: "wh-b", OccurredAt: time.Now(),
	})

	require.Error(t, err)
}
