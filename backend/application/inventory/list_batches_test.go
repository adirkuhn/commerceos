package inventory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestListBatches_returnsBatchesForSKU(t *testing.T) {
	batchRepo := &stubBatchRepo{batches: []invDomain.Batch{{ID: "batch-1", SKUID: "sku-1"}}}
	uc := appinv.NewListBatchesUseCase(newMockGormDB(t), batchRepo)

	batches, err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, batches, 1)
	require.Equal(t, "batch-1", batches[0].ID)
}
