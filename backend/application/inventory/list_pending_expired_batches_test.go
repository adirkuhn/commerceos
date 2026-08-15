package inventory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestListPendingExpiredBatches_returnsRepoResult(t *testing.T) {
	expiry := time.Now().Add(-10 * 24 * time.Hour)
	batchRepo := &stubBatchRepo{pendingReview: []invDomain.ExpiredBatchReview{
		{BatchID: "batch-1", SKUID: "sku-1", SKUCode: "SKU-1", ProductName: "Widget", ExpiryDate: expiry, Quantity: 4},
	}}
	uc := appinv.NewListPendingExpiredBatchesUseCase(newMockGormDB(t), batchRepo)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "batch-1", result[0].BatchID)
}

func TestListPendingExpiredBatches_propagatesRepoError(t *testing.T) {
	batchRepo := &stubBatchRepo{pendingReviewErr: invDomain.ErrBatchNotFound}
	uc := appinv.NewListPendingExpiredBatchesUseCase(newMockGormDB(t), batchRepo)

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}
