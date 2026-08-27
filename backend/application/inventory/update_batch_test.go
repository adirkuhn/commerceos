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

func TestUpdateBatch_updatesWarehouseLotAndExpiry(t *testing.T) {
	batchRepo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1", SKUID: "sku-1", WarehouseID: "wh-old"}}
	uc := appinv.NewUpdateBatchUseCase(newMockGormDB(t), batchRepo, &stubOutboxRepo{})
	expiry := time.Now().AddDate(0, 1, 0)

	batch, err := uc.Execute(context.Background(), appinv.UpdateBatchInput{
		ID: "batch-1", WarehouseID: "wh-new", LotNumber: "LOT-2", ExpiryDate: &expiry,
	})

	require.NoError(t, err)
	require.Equal(t, "wh-new", batch.WarehouseID)
	require.Equal(t, "LOT-2", batch.LotNumber)
	require.Equal(t, "sku-1", batch.SKUID, "SKUID must be unchanged -- not editable via update")
}

// TestUpdateBatch_raisesStockChangedEvent proves a lot/expiry correction
// (no quantity change) still reaches every channel the SKU is linked to --
// SyncStockToChannelsUseCase pushes lot_number/expiry_date to Woo alongside
// stock_quantity, so without this event a batch correction silently never
// syncs until an unrelated stock movement happens to touch the same batch.
func TestUpdateBatch_raisesStockChangedEvent(t *testing.T) {
	batchRepo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1", SKUID: "sku-1", WarehouseID: "wh-old"}}
	outboxRepo := &stubOutboxRepo{}
	uc := appinv.NewUpdateBatchUseCase(newMockGormDB(t), batchRepo, outboxRepo)

	_, err := uc.Execute(context.Background(), appinv.UpdateBatchInput{ID: "batch-1", WarehouseID: "wh-new", LotNumber: "LOT-2"})

	require.NoError(t, err)
	require.Equal(t, []string{"inventory.stock.changed"}, outboxRepo.calls)
}

func TestUpdateBatch_propagatesNotFound(t *testing.T) {
	batchRepo := &stubBatchRepo{findErr: invDomain.ErrBatchNotFound}
	uc := appinv.NewUpdateBatchUseCase(newMockGormDB(t), batchRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.UpdateBatchInput{ID: "missing", WarehouseID: "wh-1"})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrBatchNotFound))
}
