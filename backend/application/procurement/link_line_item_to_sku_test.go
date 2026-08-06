package procurement_test

import (
	"context"
	"errors"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkLineItemToSKU_setsSKUIDAndWritesAlias(t *testing.T) {
	db := newMockGormDB(t)
	li := procurementDomain.PurchaseOrderLineItem{ID: "line-item-1", PurchaseOrderID: "po-1", RawSKU: "WID-001", RawDescription: "Widget A"}
	po := procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1"), Status: procurementDomain.PurchaseOrderStatusNeedsReview}
	poRepo := &stubPurchaseOrderRepo{findResult: &po}
	lineItemRepo := &stubLineItemRepo{findResult: &li, byPO: []procurementDomain.PurchaseOrderLineItem{li}}
	aliasRepo := &stubAliasRepo{}
	uc := appprocurement.NewLinkLineItemToSKUUseCase(db, poRepo, lineItemRepo, aliasRepo)

	result, err := uc.Execute(context.Background(), appprocurement.LinkLineItemToSKUInput{LineItemID: "line-item-1", SKUID: "sku-1"})

	require.NoError(t, err)
	require.NotNil(t, result.SKUID)
	assert.Equal(t, "sku-1", *result.SKUID)

	require.Len(t, aliasRepo.upserted, 1)
	assert.Equal(t, "supplier-1", aliasRepo.upserted[0].SupplierID)
	assert.Equal(t, "wid-001", aliasRepo.upserted[0].RawKey)
	assert.Equal(t, "sku-1", aliasRepo.upserted[0].SKUID)
}

func TestLinkLineItemToSKU_revertsPOFromNeedsReviewWhenAllItemsLinked(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-already-linked"
	linkedSKUID := "sku-2"
	li1 := procurementDomain.PurchaseOrderLineItem{ID: "line-item-1", PurchaseOrderID: "po-1", RawDescription: "Widget A"}
	li2 := procurementDomain.PurchaseOrderLineItem{ID: "line-item-2", PurchaseOrderID: "po-1", SKUID: &skuID, RawDescription: "Widget B"}
	po := procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1"), Status: procurementDomain.PurchaseOrderStatusNeedsReview}
	poRepo := &stubPurchaseOrderRepo{findResult: &po}
	// byPO simulates the state ListByPurchaseOrder would see *after* the
	// Update the use case just performed -- the stub repo doesn't mutate
	// byPO in place the way a real DB would reflect its own prior write
	// within the same call.
	li1AfterLink := li1
	li1AfterLink.SKUID = &linkedSKUID
	lineItemRepo := &stubLineItemRepo{findResult: &li1, byPO: []procurementDomain.PurchaseOrderLineItem{li1AfterLink, li2}}
	aliasRepo := &stubAliasRepo{}
	uc := appprocurement.NewLinkLineItemToSKUUseCase(db, poRepo, lineItemRepo, aliasRepo)

	_, err := uc.Execute(context.Background(), appprocurement.LinkLineItemToSKUInput{LineItemID: "line-item-1", SKUID: "sku-2"})

	require.NoError(t, err)
	require.Len(t, poRepo.updated, 1)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusDraft, poRepo.updated[0].Status)
}

func TestLinkLineItemToSKU_leavesPOInNeedsReviewWhenOtherItemsStillUnmatched(t *testing.T) {
	db := newMockGormDB(t)
	li1 := procurementDomain.PurchaseOrderLineItem{ID: "line-item-1", PurchaseOrderID: "po-1", RawDescription: "Widget A"}
	li2 := procurementDomain.PurchaseOrderLineItem{ID: "line-item-2", PurchaseOrderID: "po-1", SKUID: nil, RawDescription: "Widget B"}
	po := procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1"), Status: procurementDomain.PurchaseOrderStatusNeedsReview}
	poRepo := &stubPurchaseOrderRepo{findResult: &po}
	lineItemRepo := &stubLineItemRepo{findResult: &li1, byPO: []procurementDomain.PurchaseOrderLineItem{li1, li2}}
	aliasRepo := &stubAliasRepo{}
	uc := appprocurement.NewLinkLineItemToSKUUseCase(db, poRepo, lineItemRepo, aliasRepo)

	_, err := uc.Execute(context.Background(), appprocurement.LinkLineItemToSKUInput{LineItemID: "line-item-1", SKUID: "sku-2"})

	require.NoError(t, err)
	assert.Empty(t, poRepo.updated, "PO must stay needs_review while another line item is still unmatched")
}

func TestLinkLineItemToSKU_doesNotTouchPOStatusWhenNotInNeedsReview(t *testing.T) {
	db := newMockGormDB(t)
	li := procurementDomain.PurchaseOrderLineItem{ID: "line-item-1", PurchaseOrderID: "po-1", RawDescription: "Widget A"}
	po := procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1"), Status: procurementDomain.PurchaseOrderStatusDraft}
	poRepo := &stubPurchaseOrderRepo{findResult: &po}
	lineItemRepo := &stubLineItemRepo{findResult: &li, byPO: []procurementDomain.PurchaseOrderLineItem{li}}
	aliasRepo := &stubAliasRepo{}
	uc := appprocurement.NewLinkLineItemToSKUUseCase(db, poRepo, lineItemRepo, aliasRepo)

	_, err := uc.Execute(context.Background(), appprocurement.LinkLineItemToSKUInput{LineItemID: "line-item-1", SKUID: "sku-2"})

	require.NoError(t, err)
	assert.Empty(t, poRepo.updated)
}

func TestLinkLineItemToSKU_propagatesLineItemNotFound(t *testing.T) {
	db := newMockGormDB(t)
	poRepo := &stubPurchaseOrderRepo{}
	lineItemRepo := &stubLineItemRepo{findErr: procurementDomain.ErrLineItemNotFound}
	aliasRepo := &stubAliasRepo{}
	uc := appprocurement.NewLinkLineItemToSKUUseCase(db, poRepo, lineItemRepo, aliasRepo)

	_, err := uc.Execute(context.Background(), appprocurement.LinkLineItemToSKUInput{LineItemID: "missing", SKUID: "sku-2"})

	assert.ErrorIs(t, err, procurementDomain.ErrLineItemNotFound)
}

func TestLinkLineItemToSKU_propagatesAliasUpsertError(t *testing.T) {
	db := newMockGormDB(t)
	li := procurementDomain.PurchaseOrderLineItem{ID: "line-item-1", PurchaseOrderID: "po-1", RawDescription: "Widget A"}
	po := procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1"), Status: procurementDomain.PurchaseOrderStatusNeedsReview}
	poRepo := &stubPurchaseOrderRepo{findResult: &po}
	lineItemRepo := &stubLineItemRepo{findResult: &li}
	aliasRepo := &stubAliasRepo{upsertErr: errors.New("db down")}
	uc := appprocurement.NewLinkLineItemToSKUUseCase(db, poRepo, lineItemRepo, aliasRepo)

	_, err := uc.Execute(context.Background(), appprocurement.LinkLineItemToSKUInput{LineItemID: "line-item-1", SKUID: "sku-2"})

	require.Error(t, err)
}
