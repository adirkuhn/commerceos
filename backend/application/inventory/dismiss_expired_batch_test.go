package inventory_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestDismissExpiredBatch_marksReviewed(t *testing.T) {
	batchRepo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1"}}
	uc := appinv.NewDismissExpiredBatchUseCase(newMockGormDB(t), batchRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "batch-1")

	require.NoError(t, err)
	require.Equal(t, []string{"batch-1"}, batchRepo.markReviewed)
}

func TestDismissExpiredBatch_propagatesRepoError(t *testing.T) {
	batchRepo := &stubBatchRepo{markReviewedErr: invDomain.ErrBatchNotFound}
	uc := appinv.NewDismissExpiredBatchUseCase(newMockGormDB(t), batchRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "batch-1")

	require.Error(t, err)
}
