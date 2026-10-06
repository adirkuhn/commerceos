package procurement_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func onlyEvent(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

func TestCreatePurchaseOrder_RecordsCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appprocurement.NewCreatePurchaseOrderUseCase(newMockGormDB(t), &stubPurchaseOrderRepo{}, outbox)

	_, err := uc.Execute(context.Background(), appprocurement.CreatePurchaseOrderInput{SupplierID: strPtr("sup-1"), CurrencyCode: "EUR"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.purchase_order.created", e.Subject)
	assert.Equal(t, "po-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"source": "manual", "supplier_id": "sup-1", "currency_code": "EUR", "status": "draft"}, e.Data())
}

func TestUpdatePurchaseOrder_RecordsChangedFields(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{
		ID: "po-1", SupplierID: strPtr("sup-1"), Status: procurementDomain.PurchaseOrderStatusDraft, CurrencyCode: "EUR",
	}}
	uc := appprocurement.NewUpdatePurchaseOrderUseCase(newMockGormDB(t), poRepo, &stubLineItemRepo{}, &stubAliasRepo{}, outbox)

	_, err := uc.Execute(context.Background(), appprocurement.UpdatePurchaseOrderInput{
		ID: "po-1", SupplierID: strPtr("sup-1"), Status: procurementDomain.PurchaseOrderStatusSent, CurrencyCode: "EUR", Notes: "call first",
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.purchase_order.updated", e.Subject)
	assert.Equal(t, map[string][2]any{"status": {"draft", "sent"}, "notes": {"", "call first"}}, e.Changes())
}

// A new supplier re-matches unreviewed lines against its aliases: each
// re-linked line is its own event.
func TestUpdatePurchaseOrder_SupplierChangeRecordsRelinkedLines(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("sup-1")}}
	lines := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-1", PurchaseOrderID: "po-1", RawSKU: "PAN", LineType: procurementDomain.LineItemTypeProduct},
	}}
	aliases := &stubAliasRepo{byKey: map[string]*procurementDomain.SupplierProductAlias{"sup-2|pan": {SKUID: "sku-9", UnitsPerPurchasedItem: 6}}}
	uc := appprocurement.NewUpdatePurchaseOrderUseCase(newMockGormDB(t), poRepo, lines, aliases, outbox)

	_, err := uc.Execute(context.Background(), appprocurement.UpdatePurchaseOrderInput{ID: "po-1", SupplierID: strPtr("sup-2")})

	require.NoError(t, err)
	require.Equal(t, []string{"procurement.purchase_order.updated", "procurement.purchase_order_line_item.linked"}, outbox.Subjects())
	linked := outbox.Events[1]
	assert.Equal(t, "line-1", linked.Payload["entity_id"])
	assert.Equal(t, map[string]any{"purchase_order_id": "po-1", "source": "supplier_alias"}, linked.Data())
	assert.Equal(t, map[string][2]any{"sku_id": {nil, "sku-9"}, "units_per_purchased_item": {float64(0), float64(6)}}, linked.Changes())
}

func TestDeletePurchaseOrder_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("sup-1"), Status: procurementDomain.PurchaseOrderStatusDraft}}
	uc := appprocurement.NewDeletePurchaseOrderUseCase(newMockGormDB(t), poRepo, &stubDocumentRepo{}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "po-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.purchase_order.deleted", e.Subject)
	assert.Equal(t, map[string]any{"supplier_id": "sup-1", "status": "draft"}, e.Data())
}

func TestAddPurchaseOrderLineItem_RecordsAdded(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appprocurement.NewAddPurchaseOrderLineItemUseCase(newMockGormDB(t), &stubLineItemRepo{}, outbox)

	_, err := uc.Execute(context.Background(), appprocurement.AddPurchaseOrderLineItemInput{
		PurchaseOrderID: "po-1", SKUID: strPtr("sku-1"), Quantity: 3, UnitsPerPurchasedItem: 6, UnitCostMinorUnits: 450,
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.purchase_order_line_item.added", e.Subject)
	assert.Equal(t, "line-item-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"purchase_order_id": "po-1", "sku_id": "sku-1", "quantity": float64(3), "units_per_purchased_item": float64(6),
		"unit_cost_minor_units": float64(450), "line_type": "", "source": "manual",
	}, e.Data())
}

func TestUpdatePurchaseOrderLineItem_RecordsChangedFields(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	lines := &stubLineItemRepo{findResult: &procurementDomain.PurchaseOrderLineItem{
		ID: "line-1", PurchaseOrderID: "po-1", Quantity: 3, UnitsPerPurchasedItem: 1, UnitCostMinorUnits: 450,
	}}
	uc := appprocurement.NewUpdatePurchaseOrderLineItemUseCase(newMockGormDB(t), lines, outbox)

	_, err := uc.Execute(context.Background(), appprocurement.UpdatePurchaseOrderLineItemInput{ID: "line-1", Quantity: 5, UnitCostMinorUnits: 450})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.purchase_order_line_item.updated", e.Subject)
	assert.Equal(t, map[string][2]any{"quantity": {float64(3), float64(5)}}, e.Changes())
	assert.Equal(t, map[string]any{"purchase_order_id": "po-1"}, e.Data())
}

func TestRemovePurchaseOrderLineItem_RecordsRemoved(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	lines := &stubLineItemRepo{findResult: &procurementDomain.PurchaseOrderLineItem{ID: "line-1", PurchaseOrderID: "po-1", SKUID: strPtr("sku-1")}}
	uc := appprocurement.NewRemovePurchaseOrderLineItemUseCase(newMockGormDB(t), lines, outbox)

	require.NoError(t, uc.Execute(context.Background(), "line-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.purchase_order_line_item.removed", e.Subject)
	assert.Equal(t, map[string]any{"purchase_order_id": "po-1", "sku_id": "sku-1"}, e.Data())
}

// Linking the last unlinked line takes the PO out of needs_review: the
// link and the PO's status change are each recorded.
func TestLinkLineItemToSKU_RecordsLinkAndReviewResolution(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{
		ID: "po-1", SupplierID: strPtr("sup-1"), Status: procurementDomain.PurchaseOrderStatusNeedsReview,
	}}
	line := procurementDomain.PurchaseOrderLineItem{ID: "line-1", PurchaseOrderID: "po-1", LineType: procurementDomain.LineItemTypeProduct, UnitsPerPurchasedItem: 1}
	lines := &stubLineItemRepo{findResult: &line, byPO: []procurementDomain.PurchaseOrderLineItem{line}}
	uc := appprocurement.NewLinkLineItemToSKUUseCase(newMockGormDB(t), poRepo, lines, &stubAliasRepo{}, outbox)

	_, err := uc.Execute(context.Background(), appprocurement.LinkLineItemToSKUInput{LineItemID: "line-1", SKUID: "sku-1", UnitsPerPurchasedItem: 6})

	require.NoError(t, err)
	require.Equal(t, []string{"procurement.purchase_order_line_item.linked", "procurement.purchase_order.updated"}, outbox.Subjects())
	assert.Equal(t, map[string]any{"purchase_order_id": "po-1", "source": "manual"}, outbox.Events[0].Data())
	assert.Equal(t, map[string][2]any{"sku_id": {nil, "sku-1"}, "units_per_purchased_item": {float64(1), float64(6)}}, outbox.Events[0].Changes())
	resolved := outbox.Events[1]
	assert.Equal(t, "po-1", resolved.Payload["entity_id"])
	assert.Equal(t, map[string][2]any{"status": {"needs_review", "draft"}}, resolved.Changes())
	assert.Equal(t, map[string]any{"reason": "line_items_resolved"}, resolved.Data())
}

func TestPurchaseOrderUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := func() *eventrecordtest.Outbox { return &eventrecordtest.Outbox{Err: boom} }
	po := func() *stubPurchaseOrderRepo {
		return &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	}
	line := func() *stubLineItemRepo {
		return &stubLineItemRepo{findResult: &procurementDomain.PurchaseOrderLineItem{ID: "line-1", PurchaseOrderID: "po-1"}}
	}
	ctx := context.Background()

	_, err := appprocurement.NewCreatePurchaseOrderUseCase(newMockGormDB(t), po(), failing()).Execute(ctx, appprocurement.CreatePurchaseOrderInput{})
	assert.ErrorIs(t, err, boom, "create")
	_, err = appprocurement.NewUpdatePurchaseOrderUseCase(newMockGormDB(t), po(), line(), &stubAliasRepo{}, failing()).Execute(ctx, appprocurement.UpdatePurchaseOrderInput{ID: "po-1", Notes: "x"})
	assert.ErrorIs(t, err, boom, "update")
	assert.ErrorIs(t, appprocurement.NewDeletePurchaseOrderUseCase(newMockGormDB(t), po(), &stubDocumentRepo{}, failing()).Execute(ctx, "po-1"), boom, "delete")
	_, err = appprocurement.NewAddPurchaseOrderLineItemUseCase(newMockGormDB(t), line(), failing()).Execute(ctx, appprocurement.AddPurchaseOrderLineItemInput{PurchaseOrderID: "po-1", Quantity: 1})
	assert.ErrorIs(t, err, boom, "add line")
	_, err = appprocurement.NewUpdatePurchaseOrderLineItemUseCase(newMockGormDB(t), line(), failing()).Execute(ctx, appprocurement.UpdatePurchaseOrderLineItemInput{ID: "line-1", Quantity: 2})
	assert.ErrorIs(t, err, boom, "update line")
	assert.ErrorIs(t, appprocurement.NewRemovePurchaseOrderLineItemUseCase(newMockGormDB(t), line(), failing()).Execute(ctx, "line-1"), boom, "remove line")
	_, err = appprocurement.NewLinkLineItemToSKUUseCase(newMockGormDB(t), po(), line(), &stubAliasRepo{}, failing()).Execute(ctx, appprocurement.LinkLineItemToSKUInput{LineItemID: "line-1", SKUID: "sku-1"})
	assert.ErrorIs(t, err, boom, "link")
}
