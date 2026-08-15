package inventory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestExpireBatch_marksBatchExpired(t *testing.T) {
	batchRepo := &stubBatchRepo{}
	uc := appinv.NewExpireBatchUseCase(newMockGormDB(t), batchRepo)

	err := uc.Execute(context.Background(), "batch-1")

	require.NoError(t, err)
	require.Equal(t, []string{"batch-1"}, batchRepo.markExpired)
}

func TestExpireBatch_propagatesRepoError(t *testing.T) {
	batchRepo := &stubBatchRepo{markExpiredErr: invDomain.ErrBatchNotFound}
	uc := appinv.NewExpireBatchUseCase(newMockGormDB(t), batchRepo)

	err := uc.Execute(context.Background(), "batch-1")

	require.Error(t, err)
}
