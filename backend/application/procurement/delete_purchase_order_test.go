package procurement_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestDeletePurchaseOrder_deletesExisting(t *testing.T) {
	repo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	uc := appprocurement.NewDeletePurchaseOrderUseCase(newMockGormDB(t), repo)

	err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	require.Equal(t, []string{"po-1"}, repo.deleted)
}

func TestDeletePurchaseOrder_propagatesNotFound(t *testing.T) {
	repo := &stubPurchaseOrderRepo{findErr: procurementDomain.ErrPurchaseOrderNotFound}
	uc := appprocurement.NewDeletePurchaseOrderUseCase(newMockGormDB(t), repo)

	err := uc.Execute(context.Background(), "missing")

	require.ErrorIs(t, err, procurementDomain.ErrPurchaseOrderNotFound)
}
