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

func TestListExpiringBatches_returnsExpiringBatches(t *testing.T) {
	expiry := time.Now().Add(48 * time.Hour)
	batchRepo := &stubBatchRepo{expiringSoon: []invDomain.ExpiringBatch{
		{BatchID: "batch-1", SKUID: "sku-1", SKUCode: "SKU-1", ProductName: "Widget", ExpiryDate: expiry, Quantity: 5},
	}}
	uc := appinv.NewListExpiringBatchesUseCase(newMockGormDB(t), batchRepo)

	batches, err := uc.Execute(context.Background(), 30)

	require.NoError(t, err)
	require.Len(t, batches, 1)
	require.Equal(t, "SKU-1", batches[0].SKUCode)
}

func TestListExpiringBatches_propagatesRepoError(t *testing.T) {
	batchRepo := &stubBatchRepo{expiringSoonErr: errors.New("db down")}
	uc := appinv.NewListExpiringBatchesUseCase(newMockGormDB(t), batchRepo)

	_, err := uc.Execute(context.Background(), 30)

	require.Error(t, err)
}
