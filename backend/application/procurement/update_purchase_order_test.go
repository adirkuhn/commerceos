package procurement_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestUpdatePurchaseOrder_updatesFields(t *testing.T) {
	repo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", Status: procurementDomain.PurchaseOrderStatusDraft}}
	uc := appprocurement.NewUpdatePurchaseOrderUseCase(newMockGormDB(t), repo, &stubLineItemRepo{}, &stubAliasRepo{})

	po, err := uc.Execute(context.Background(), appprocurement.UpdatePurchaseOrderInput{
		ID: "po-1", SupplierID: strPtr("supplier-2"), WarehouseID: strPtr("wh-2"), Status: procurementDomain.PurchaseOrderStatusSent, CurrencyCode: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, "supplier-2", *po.SupplierID)
	require.Equal(t, procurementDomain.PurchaseOrderStatusSent, po.Status)
}

func TestUpdatePurchaseOrder_relinksAliasAutoMatchedLineItemsWhenSupplierChanges(t *testing.T) {
	skuOld := "sku-old-guess"
	skuManual := "sku-manual"
	repo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{
		ID: "po-1", Status: procurementDomain.PurchaseOrderStatusDraft, SupplierID: strPtr("supplier-old"),
	}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-alias", RawSKU: "PT-1", SKUID: &skuOld, SKUAutoLinkedFromAlias: true},
		{ID: "line-manual", RawSKU: "PT-1", SKUID: &skuManual, SKUAutoLinkedFromAlias: false},
	}}
	aliasRepo := &stubAliasRepo{byKey: map[string]*procurementDomain.SupplierProductAlias{
		"supplier-new|pt-1": {SupplierID: "supplier-new", RawKey: "pt-1", SKUID: "sku-new-guess"},
	}}
	uc := appprocurement.NewUpdatePurchaseOrderUseCase(newMockGormDB(t), repo, lineItemRepo, aliasRepo)

	_, err := uc.Execute(context.Background(), appprocurement.UpdatePurchaseOrderInput{
		ID: "po-1", SupplierID: strPtr("supplier-new"), Status: procurementDomain.PurchaseOrderStatusDraft, CurrencyCode: "USD",
	})

	require.NoError(t, err)
	require.Len(t, lineItemRepo.updated, 1, "only the alias-auto-linked item is eligible -- the manually-linked one must never be touched")
	assert.Equal(t, "line-alias", lineItemRepo.updated[0].ID)
	require.NotNil(t, lineItemRepo.updated[0].SKUID)
	assert.Equal(t, "sku-new-guess", *lineItemRepo.updated[0].SKUID)
	assert.True(t, lineItemRepo.updated[0].SKUAutoLinkedFromAlias)
}

func TestUpdatePurchaseOrder_clearsAliasAutoMatchedLineItemWhenNewSupplierHasNoAlias(t *testing.T) {
	skuOld := "sku-old-guess"
	repo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{
		ID: "po-1", Status: procurementDomain.PurchaseOrderStatusDraft, SupplierID: strPtr("supplier-old"),
	}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-alias", RawSKU: "PT-1", SKUID: &skuOld, SKUAutoLinkedFromAlias: true},
	}}
	uc := appprocurement.NewUpdatePurchaseOrderUseCase(newMockGormDB(t), repo, lineItemRepo, &stubAliasRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.UpdatePurchaseOrderInput{
		ID: "po-1", SupplierID: strPtr("supplier-new"), Status: procurementDomain.PurchaseOrderStatusDraft, CurrencyCode: "USD",
	})

	require.NoError(t, err)
	require.Len(t, lineItemRepo.updated, 1)
	assert.Nil(t, lineItemRepo.updated[0].SKUID, "old supplier's guess must not survive a supplier change with no matching alias")
	assert.False(t, lineItemRepo.updated[0].SKUAutoLinkedFromAlias)
}

func TestUpdatePurchaseOrder_doesNotResyncAliasesWhenSupplierUnchanged(t *testing.T) {
	repo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{
		ID: "po-1", Status: procurementDomain.PurchaseOrderStatusDraft, SupplierID: strPtr("supplier-1"),
	}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{}}
	uc := appprocurement.NewUpdatePurchaseOrderUseCase(newMockGormDB(t), repo, lineItemRepo, &stubAliasRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.UpdatePurchaseOrderInput{
		ID: "po-1", SupplierID: strPtr("supplier-1"), Status: procurementDomain.PurchaseOrderStatusSent, CurrencyCode: "USD",
	})

	require.NoError(t, err)
	assert.Empty(t, lineItemRepo.updated)
}

func TestUpdatePurchaseOrder_propagatesNotFound(t *testing.T) {
	repo := &stubPurchaseOrderRepo{findErr: procurementDomain.ErrPurchaseOrderNotFound}
	uc := appprocurement.NewUpdatePurchaseOrderUseCase(newMockGormDB(t), repo, &stubLineItemRepo{}, &stubAliasRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.UpdatePurchaseOrderInput{ID: "missing"})

	require.ErrorIs(t, err, procurementDomain.ErrPurchaseOrderNotFound)
}
