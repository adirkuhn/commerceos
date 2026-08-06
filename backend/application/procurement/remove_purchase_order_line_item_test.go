package procurement_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestRemovePurchaseOrderLineItem_removesExisting(t *testing.T) {
	repo := &stubLineItemRepo{findResult: &procurementDomain.PurchaseOrderLineItem{ID: "line-item-1"}}
	uc := appprocurement.NewRemovePurchaseOrderLineItemUseCase(newMockGormDB(t), repo)

	err := uc.Execute(context.Background(), "line-item-1")

	require.NoError(t, err)
	require.Equal(t, []string{"line-item-1"}, repo.deleted)
}

func TestRemovePurchaseOrderLineItem_propagatesNotFound(t *testing.T) {
	repo := &stubLineItemRepo{findErr: procurementDomain.ErrLineItemNotFound}
	uc := appprocurement.NewRemovePurchaseOrderLineItemUseCase(newMockGormDB(t), repo)

	err := uc.Execute(context.Background(), "missing")

	require.ErrorIs(t, err, procurementDomain.ErrLineItemNotFound)
}
