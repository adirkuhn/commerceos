package procurement_test

import (
	"context"
	"errors"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	suppliersapp "github.com/adirkuhn/commerceos/application/suppliers"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCreateSupplierForAssignUC struct {
	out      *suppliersDomain.Supplier
	err      error
	gotInput suppliersapp.CreateSupplierInput
}

func (s *stubCreateSupplierForAssignUC) Execute(_ context.Context, in suppliersapp.CreateSupplierInput) (*suppliersDomain.Supplier, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &suppliersDomain.Supplier{ID: "new-supplier-1", Name: in.Name, Status: in.Status}, nil
}

func TestCreateSupplierAndAssignToPurchaseOrder_createsAndAssigns(t *testing.T) {
	db := newMockGormDB(t)
	createSupplier := &stubCreateSupplierForAssignUC{}
	po := &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: nil, Status: procurementDomain.PurchaseOrderStatusNeedsReview}
	poRepo := &stubPurchaseOrderRepo{findResult: po}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{}}
	uc := appprocurement.NewCreateSupplierAndAssignToPurchaseOrderUseCase(db, createSupplier, poRepo, lineItemRepo)

	result, err := uc.Execute(context.Background(), appprocurement.CreateSupplierAndAssignToPurchaseOrderInput{
		PurchaseOrderID: "po-1", Name: "New Supplier Co",
	})

	require.NoError(t, err)
	assert.Equal(t, "New Supplier Co", createSupplier.gotInput.Name)
	assert.Equal(t, suppliersDomain.SupplierStatusActive, createSupplier.gotInput.Status)
	require.NotNil(t, result.SupplierID)
	assert.Equal(t, "new-supplier-1", *result.SupplierID)
}

func TestCreateSupplierAndAssignToPurchaseOrder_revertsToDraftWhenAllLineItemsAlreadyLinked(t *testing.T) {
	db := newMockGormDB(t)
	createSupplier := &stubCreateSupplierForAssignUC{}
	po := &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: nil, Status: procurementDomain.PurchaseOrderStatusNeedsReview}
	poRepo := &stubPurchaseOrderRepo{findResult: po}
	skuID := "sku-1"
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-item-1", SKUID: &skuID}}}
	uc := appprocurement.NewCreateSupplierAndAssignToPurchaseOrderUseCase(db, createSupplier, poRepo, lineItemRepo)

	result, err := uc.Execute(context.Background(), appprocurement.CreateSupplierAndAssignToPurchaseOrderInput{
		PurchaseOrderID: "po-1", Name: "New Supplier Co",
	})

	require.NoError(t, err)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusDraft, result.Status)
}

func TestCreateSupplierAndAssignToPurchaseOrder_staysNeedsReviewWhenLineItemsStillUnmatched(t *testing.T) {
	db := newMockGormDB(t)
	createSupplier := &stubCreateSupplierForAssignUC{}
	po := &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: nil, Status: procurementDomain.PurchaseOrderStatusNeedsReview}
	poRepo := &stubPurchaseOrderRepo{findResult: po}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-item-1", SKUID: nil}}}
	uc := appprocurement.NewCreateSupplierAndAssignToPurchaseOrderUseCase(db, createSupplier, poRepo, lineItemRepo)

	result, err := uc.Execute(context.Background(), appprocurement.CreateSupplierAndAssignToPurchaseOrderInput{
		PurchaseOrderID: "po-1", Name: "New Supplier Co",
	})

	require.NoError(t, err)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusNeedsReview, result.Status, "still missing a matched line item")
}

func TestCreateSupplierAndAssignToPurchaseOrder_propagatesCreateSupplierError(t *testing.T) {
	db := newMockGormDB(t)
	createSupplier := &stubCreateSupplierForAssignUC{err: errors.New("duplicate supplier name")}
	uc := appprocurement.NewCreateSupplierAndAssignToPurchaseOrderUseCase(db, createSupplier, &stubPurchaseOrderRepo{}, &stubLineItemRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.CreateSupplierAndAssignToPurchaseOrderInput{PurchaseOrderID: "po-1", Name: "X"})

	require.Error(t, err)
}

func TestCreateSupplierAndAssignToPurchaseOrder_propagatesPurchaseOrderNotFound(t *testing.T) {
	db := newMockGormDB(t)
	createSupplier := &stubCreateSupplierForAssignUC{}
	poRepo := &stubPurchaseOrderRepo{findErr: procurementDomain.ErrPurchaseOrderNotFound}
	uc := appprocurement.NewCreateSupplierAndAssignToPurchaseOrderUseCase(db, createSupplier, poRepo, &stubLineItemRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.CreateSupplierAndAssignToPurchaseOrderInput{PurchaseOrderID: "missing", Name: "X"})

	assert.ErrorIs(t, err, procurementDomain.ErrPurchaseOrderNotFound)
}
