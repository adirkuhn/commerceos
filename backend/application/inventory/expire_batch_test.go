package inventory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestExpireBatch_marksBatchExpired(t *testing.T) {
	batchRepo := &stubBatchRepo{}
	uc := appinv.NewExpireBatchUseCase(newMockGormDB(t), batchRepo, &stubOutboxRepo{})

	err := uc.Execute(context.Background(), "batch-1", "sku-1")

	require.NoError(t, err)
	require.Equal(t, []string{"batch-1"}, batchRepo.markExpired)
}

func TestExpireBatch_propagatesRepoError(t *testing.T) {
	batchRepo := &stubBatchRepo{markExpiredErr: invDomain.ErrBatchNotFound}
	uc := appinv.NewExpireBatchUseCase(newMockGormDB(t), batchRepo, &stubOutboxRepo{})

	err := uc.Execute(context.Background(), "batch-1", "sku-1")

	require.Error(t, err)
}

// TestExpireBatch_enqueuesStockChanged is the regression test for the bug
// where an auto-expired batch never reached sales channels: MarkExpired
// alone doesn't touch StockMovement, so nothing told channels to re-pull
// the now-lower Syncable quantity until some unrelated event happened to
// touch the same SKU. Same enqueueStockChanged pattern as every other
// stock-affecting use case (see update_batch.go).
func TestExpireBatch_enqueuesStockChanged(t *testing.T) {
	batchRepo := &stubBatchRepo{}
	outboxRepo := &stubOutboxRepo{}
	uc := appinv.NewExpireBatchUseCase(newMockGormDB(t), batchRepo, outboxRepo)

	err := uc.Execute(context.Background(), "batch-1", "sku-1")

	require.NoError(t, err)
	require.Equal(t, []string{"inventory.stock.changed"}, outboxRepo.calls)
}

// TestExpireBatch_outboxFailurePropagates makes sure a failed outbox write
// fails the whole call -- the event must commit atomically with the status
// flip (same tx), not best-effort.
func TestExpireBatch_outboxFailurePropagates(t *testing.T) {
	batchRepo := &stubBatchRepo{}
	outboxRepo := &stubOutboxRepo{err: errors.New("outbox down")}
	uc := appinv.NewExpireBatchUseCase(newMockGormDB(t), batchRepo, outboxRepo)

	err := uc.Execute(context.Background(), "batch-1", "sku-1")

	require.Error(t, err)
}
