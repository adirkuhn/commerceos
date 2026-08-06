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
	uc := appprocurement.NewGetPurchaseOrderUseCase(newMockGormDB(t), poRepo, lineItemRepo, &stubDocumentRepo{})

	detail, err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	require.Equal(t, "supplier-1", *detail.PurchaseOrder.SupplierID)
	require.Len(t, detail.LineItems, 1)
	require.Equal(t, int64(10), detail.LineItems[0].Quantity)
}

func TestGetPurchaseOrder_returnsUploadedDocuments(t *testing.T) {
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	documentRepo := &stubDocumentRepo{byPO: []procurementDomain.PurchaseOrderDocument{
		{ID: "document-1", FileURL: "https://uploads.example.com/tenant-1/abc.pdf", FileName: "invoice.pdf", ContentType: "application/pdf"},
	}}
	uc := appprocurement.NewGetPurchaseOrderUseCase(newMockGormDB(t), poRepo, &stubLineItemRepo{}, documentRepo)

	detail, err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	require.Len(t, detail.Documents, 1)
	require.Equal(t, "https://uploads.example.com/tenant-1/abc.pdf", detail.Documents[0].FileURL)
}

func TestGetPurchaseOrder_propagatesNotFound(t *testing.T) {
	poRepo := &stubPurchaseOrderRepo{findErr: procurementDomain.ErrPurchaseOrderNotFound}
	lineItemRepo := &stubLineItemRepo{}
	uc := appprocurement.NewGetPurchaseOrderUseCase(newMockGormDB(t), poRepo, lineItemRepo, &stubDocumentRepo{})

	_, err := uc.Execute(context.Background(), "missing")

	require.ErrorIs(t, err, procurementDomain.ErrPurchaseOrderNotFound)
}
