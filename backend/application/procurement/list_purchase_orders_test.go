package procurement_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestListPurchaseOrders_returnsRepoResult(t *testing.T) {
	repo := &stubPurchaseOrderRepo{purchaseOrders: []procurementDomain.PurchaseOrder{{ID: "po-1"}}}
	uc := appprocurement.NewListPurchaseOrdersUseCase(newMockGormDB(t), repo)

	pos, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, pos, 1)
	require.Equal(t, "po-1", pos[0].ID)
}
