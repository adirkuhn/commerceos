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
	uc := appinv.NewUpdateBatchUseCase(newMockGormDB(t), batchRepo)
	expiry := time.Now().AddDate(0, 1, 0)

	batch, err := uc.Execute(context.Background(), appinv.UpdateBatchInput{
		ID: "batch-1", WarehouseID: "wh-new", LotNumber: "LOT-2", ExpiryDate: &expiry,
	})

	require.NoError(t, err)
	require.Equal(t, "wh-new", batch.WarehouseID)
	require.Equal(t, "LOT-2", batch.LotNumber)
	require.Equal(t, "sku-1", batch.SKUID, "SKUID must be unchanged -- not editable via update")
}

func TestUpdateBatch_propagatesNotFound(t *testing.T) {
	batchRepo := &stubBatchRepo{findErr: invDomain.ErrBatchNotFound}
	uc := appinv.NewUpdateBatchUseCase(newMockGormDB(t), batchRepo)

	_, err := uc.Execute(context.Background(), appinv.UpdateBatchInput{ID: "missing", WarehouseID: "wh-1"})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrBatchNotFound))
}
