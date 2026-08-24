package pms_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestListProductHistory_returnsRepoResult(t *testing.T) {
	historyRepo := &stubProductHistoryRepo{listResult: []pmsDomain.ProductHistory{
		{ID: "h1", EntityType: pmsDomain.ProductHistoryEntityProduct, EntityID: "p1", Field: pmsDomain.ProductHistoryFieldName, OldValue: "Old", NewValue: "New", ChangedAt: time.Now()},
	}}
	uc := apppms.NewListProductHistoryUseCase(newMockGormDB(t), historyRepo)

	result, err := uc.Execute(context.Background(), pmsDomain.ProductHistoryEntityProduct, "p1")

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "New", result[0].NewValue)
}

func TestListProductHistory_propagatesRepoError(t *testing.T) {
	historyRepo := &stubProductHistoryRepo{listErr: context.DeadlineExceeded}
	uc := apppms.NewListProductHistoryUseCase(newMockGormDB(t), historyRepo)

	_, err := uc.Execute(context.Background(), pmsDomain.ProductHistoryEntityProduct, "p1")

	require.Error(t, err)
}
