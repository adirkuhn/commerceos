package inventory_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestDeleteBatch_deletesABatchWithNoMovements(t *testing.T) {
	batchRepo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1"}}
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewDeleteBatchUseCase(newMockGormDB(t), batchRepo, movementRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "batch-1")

	require.NoError(t, err)
	require.Equal(t, []string{"batch-1"}, batchRepo.deleted)
}

func TestDeleteBatch_rejectsABatchWithMovements(t *testing.T) {
	batchRepo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1"}}
	movementRepo := &stubStockMovementRepo{byBatch: []invDomain.StockMovement{{ID: "movement-1", BatchID: strPtr("batch-1")}}}
	uc := appinv.NewDeleteBatchUseCase(newMockGormDB(t), batchRepo, movementRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "batch-1")

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrBatchHasMovements))
	require.Empty(t, batchRepo.deleted)
}

func TestDeleteBatch_propagatesNotFound(t *testing.T) {
	batchRepo := &stubBatchRepo{findErr: invDomain.ErrBatchNotFound}
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewDeleteBatchUseCase(newMockGormDB(t), batchRepo, movementRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrBatchNotFound))
}
