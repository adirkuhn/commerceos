package inventory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
)

func TestCreateBatch_setsFields(t *testing.T) {
	batchRepo := &stubBatchRepo{}
	uc := appinv.NewCreateBatchUseCase(newMockGormDB(t), batchRepo)

	expiry := time.Now().Add(30 * 24 * time.Hour)
	b, err := uc.Execute(context.Background(), appinv.CreateBatchInput{
		SKUID: "sku-1", WarehouseID: "wh-1", LotNumber: "LOT-1", ExpiryDate: &expiry,
	})

	require.NoError(t, err)
	require.Equal(t, "sku-1", b.SKUID)
	require.Equal(t, "LOT-1", b.LotNumber)
	require.True(t, batchRepo.createCalled)
}
