package procurement_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestGetPurchaseOrder_returnsHeaderAndLineItems(t *testing.T) {
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1")}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-item-1", Quantity: 10}}}
	uc := appprocurement.NewGetPurchaseOrderUseCase(newMockGormDB(t), poRepo, lineItemRepo, &stubDocumentRepo{}, &stubSKURepo{}, &stubVariantRepo{}, &stubProductRepo{})

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
	uc := appprocurement.NewGetPurchaseOrderUseCase(newMockGormDB(t), poRepo, &stubLineItemRepo{}, documentRepo, &stubSKURepo{}, &stubVariantRepo{}, &stubProductRepo{})

	detail, err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	require.Len(t, detail.Documents, 1)
	require.Equal(t, "https://uploads.example.com/tenant-1/abc.pdf", detail.Documents[0].FileURL)
}

// TestGetPurchaseOrder_linkedLineItemGetsProductNameAndSKUCode is the fix for
// the review table showing a linked item's raw sku_id UUID instead of
// something a human can read -- GetPurchaseOrderUseCase now resolves
// sku_id -> SKU -> Variant -> Product and attaches ProductName/SKUCode.
func TestGetPurchaseOrder_linkedLineItemGetsProductNameAndSKUCode(t *testing.T) {
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-item-1", SKUID: strPtr("sku-1")}}}
	skuRepo := &stubSKURepo{findByIDs: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1", VariantID: "variant-1", SKUCode: "WID-001"}}}
	variantRepo := &stubVariantRepo{byID: map[string]*pmsDomain.Variant{"variant-1": {ID: "variant-1", ProductID: "product-1"}}}
	productRepo := &stubProductRepo{byID: map[string]*pmsDomain.Product{"product-1": {ID: "product-1", Name: "Acme Widget"}}}
	uc := appprocurement.NewGetPurchaseOrderUseCase(newMockGormDB(t), poRepo, lineItemRepo, &stubDocumentRepo{}, skuRepo, variantRepo, productRepo)

	detail, err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	require.Len(t, detail.LineItems, 1)
	require.NotNil(t, detail.LineItems[0].ProductName)
	require.NotNil(t, detail.LineItems[0].SKUCode)
	require.Equal(t, "Acme Widget", *detail.LineItems[0].ProductName)
	require.Equal(t, "WID-001", *detail.LineItems[0].SKUCode)
}

// TestGetPurchaseOrder_unlinkedLineItemHasNilProductDisplayFields is the
// baseline: a still-unmatched item (SKUID nil, awaiting review) must not
// trigger any SKU/Variant/Product lookup, and its display fields stay nil.
func TestGetPurchaseOrder_unlinkedLineItemHasNilProductDisplayFields(t *testing.T) {
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-item-1", SKUID: nil}}}
	uc := appprocurement.NewGetPurchaseOrderUseCase(newMockGormDB(t), poRepo, lineItemRepo, &stubDocumentRepo{}, &stubSKURepo{}, &stubVariantRepo{}, &stubProductRepo{})

	detail, err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	require.Len(t, detail.LineItems, 1)
	require.Nil(t, detail.LineItems[0].ProductName)
	require.Nil(t, detail.LineItems[0].SKUCode)
}

// TestGetPurchaseOrder_computesTotalFromLineItems is the "PO total on the
// edit screen" feature: quantity*unit_cost summed across every line item.
// unit_cost_minor_units is always stored non-negative (openapi
// minimum:0) -- a discount/credit_note row's own value is a positive
// amount, and the total calc itself must flip the sign for those types
// (see LineItemType.IsReduction), not rely on a negative stored value.
func TestGetPurchaseOrder_computesTotalFromLineItems(t *testing.T) {
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "li-1", Quantity: 10, UnitCostMinorUnits: 500},                                                   // product, 5000
		{ID: "li-2", Quantity: 1, UnitCostMinorUnits: 1500, LineType: procurementDomain.LineItemTypeShipping}, // adds 1500
		{ID: "li-3", Quantity: 1, UnitCostMinorUnits: 1000, LineType: procurementDomain.LineItemTypeDiscount}, // subtracts 1000
	}}
	uc := appprocurement.NewGetPurchaseOrderUseCase(newMockGormDB(t), poRepo, lineItemRepo, &stubDocumentRepo{}, &stubSKURepo{}, &stubVariantRepo{}, &stubProductRepo{})

	detail, err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	require.Equal(t, int64(5500), detail.PurchaseOrder.TotalMinorUnits, "5000 + 1500 - 1000")
}

func TestGetPurchaseOrder_creditNoteAlsoSubtractsFromTotal(t *testing.T) {
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "li-1", Quantity: 10, UnitCostMinorUnits: 500},
		{ID: "li-2", Quantity: 1, UnitCostMinorUnits: 750, LineType: procurementDomain.LineItemTypeCreditNote},
	}}
	uc := appprocurement.NewGetPurchaseOrderUseCase(newMockGormDB(t), poRepo, lineItemRepo, &stubDocumentRepo{}, &stubSKURepo{}, &stubVariantRepo{}, &stubProductRepo{})

	detail, err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	require.Equal(t, int64(4250), detail.PurchaseOrder.TotalMinorUnits, "5000 - 750")
}

func TestGetPurchaseOrder_propagatesNotFound(t *testing.T) {
	poRepo := &stubPurchaseOrderRepo{findErr: procurementDomain.ErrPurchaseOrderNotFound}
	lineItemRepo := &stubLineItemRepo{}
	uc := appprocurement.NewGetPurchaseOrderUseCase(newMockGormDB(t), poRepo, lineItemRepo, &stubDocumentRepo{}, &stubSKURepo{}, &stubVariantRepo{}, &stubProductRepo{})

	_, err := uc.Execute(context.Background(), "missing")

	require.ErrorIs(t, err, procurementDomain.ErrPurchaseOrderNotFound)
}
