package inventory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestExtendBatchExpiry_extendsToNewDate(t *testing.T) {
	batchRepo := &stubBatchRepo{}
	uc := appinv.NewExtendBatchExpiryUseCase(newMockGormDB(t), batchRepo)
	newExpiry := time.Now().Add(60 * 24 * time.Hour)

	err := uc.Execute(context.Background(), "batch-1", newExpiry)

	require.NoError(t, err)
	require.Equal(t, "batch-1", batchRepo.extendedID)
	require.True(t, newExpiry.Equal(batchRepo.extendedExpiryDate))
}

func TestExtendBatchExpiry_propagatesRepoError(t *testing.T) {
	batchRepo := &stubBatchRepo{extendExpiryErr: invDomain.ErrBatchNotFound}
	uc := appinv.NewExtendBatchExpiryUseCase(newMockGormDB(t), batchRepo)

	err := uc.Execute(context.Background(), "batch-1", time.Now())

	require.Error(t, err)
}
