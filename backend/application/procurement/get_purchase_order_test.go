package procurement_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestGetPurchaseOrder_returnsHeaderAndLineItems(t *testing.T) {
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1")}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-item-1", Quantity: 10}}}
	uc := appprocurement.NewGetPurchaseOrderUseCase(newMockGormDB(t), poRepo, lineItemRepo)

	detail, err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	require.Equal(t, "supplier-1", *detail.PurchaseOrder.SupplierID)
	require.Len(t, detail.LineItems, 1)
	require.Equal(t, int64(10), detail.LineItems[0].Quantity)
}

func TestGetPurchaseOrder_propagatesNotFound(t *testing.T) {
	poRepo := &stubPurchaseOrderRepo{findErr: procurementDomain.ErrPurchaseOrderNotFound}
	lineItemRepo := &stubLineItemRepo{}
	uc := appprocurement.NewGetPurchaseOrderUseCase(newMockGormDB(t), poRepo, lineItemRepo)

	_, err := uc.Execute(context.Background(), "missing")

	require.ErrorIs(t, err, procurementDomain.ErrPurchaseOrderNotFound)
}
