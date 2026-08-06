package procurement_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestUpdatePurchaseOrder_updatesFields(t *testing.T) {
	repo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", Status: procurementDomain.PurchaseOrderStatusDraft}}
	uc := appprocurement.NewUpdatePurchaseOrderUseCase(newMockGormDB(t), repo)

	po, err := uc.Execute(context.Background(), appprocurement.UpdatePurchaseOrderInput{
		ID: "po-1", SupplierID: strPtr("supplier-2"), WarehouseID: strPtr("wh-2"), Status: procurementDomain.PurchaseOrderStatusSent, CurrencyCode: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, "supplier-2", *po.SupplierID)
	require.Equal(t, procurementDomain.PurchaseOrderStatusSent, po.Status)
}

func TestUpdatePurchaseOrder_propagatesNotFound(t *testing.T) {
	repo := &stubPurchaseOrderRepo{findErr: procurementDomain.ErrPurchaseOrderNotFound}
	uc := appprocurement.NewUpdatePurchaseOrderUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appprocurement.UpdatePurchaseOrderInput{ID: "missing"})

	require.ErrorIs(t, err, procurementDomain.ErrPurchaseOrderNotFound)
}
