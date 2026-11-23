package tools_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

type stubPOGet struct {
	out   *appprocurement.PurchaseOrderDetail
	err   error
	gotID string
}

func (s *stubPOGet) Execute(_ context.Context, id string) (*appprocurement.PurchaseOrderDetail, error) {
	s.gotID = id
	return s.out, s.err
}

type stubWarehouses struct {
	out []invDomain.Warehouse
	err error
}

func (s stubWarehouses) Execute(context.Context) ([]invDomain.Warehouse, error) { return s.out, s.err }

func poFixtures() stubPOList {
	s1, s2 := "s-1", "s-2"
	base := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	exp := base.AddDate(0, 0, 10)
	return stubPOList{out: []procurementDomain.PurchaseOrder{
		{ID: "po-1", SupplierID: &s1, Status: procurementDomain.PurchaseOrderStatusSent, CurrencyCode: "EUR", ExpectedDate: &exp, TotalMinorUnits: 1000, CreatedAt: base},
		{ID: "po-2", SupplierID: &s2, Status: procurementDomain.PurchaseOrderStatusReceived, CurrencyCode: "EUR", TotalMinorUnits: 2000, CreatedAt: base.AddDate(0, 0, 1)},
		{ID: "po-3", Status: procurementDomain.PurchaseOrderStatusNeedsReview, SuggestedSupplierName: "Acme (from PDF)", CreatedAt: base.AddDate(0, 0, 2)},
	}}
}

func TestListPurchaseOrders(t *testing.T) {
	tool := tools.NewListPurchaseOrdersTool(poFixtures(), someSuppliers())
	assertStrictDefinition(t, tool, "list_purchase_orders")

	all := call(t, tool, `{}`)
	list := all["purchase_orders"].([]any)
	require.Len(t, list, 3)
	first := list[0].(map[string]any)
	assert.Equal(t, "po-3", first["id"], "newest first")
	assert.Equal(t, "Acme (from PDF)", first["suggested_supplier_name"])
	assert.Equal(t, "Bauducco Foods", list[2].(map[string]any)["supplier"].(map[string]any)["name"])
	assert.NotEmpty(t, list[2].(map[string]any)["expected_date"])

	bySupplier := call(t, tool, `{"supplier_id":"s-2"}`)
	require.Len(t, bySupplier["purchase_orders"], 1)

	byStatus := call(t, tool, `{"status":"sent"}`)
	require.Len(t, byStatus["purchase_orders"], 1)
	assert.Equal(t, "po-1", byStatus["purchase_orders"].([]any)[0].(map[string]any)["id"])

	assert.ErrorIs(t, callErr(t, tool, `{"status":"lost"}`), tools.ErrInvalidArguments)
	assert.ErrorContains(t, callErr(t, tools.NewListPurchaseOrdersTool(stubPOList{err: errors.New("boom")}, someSuppliers()), `{}`), "boom")
}

func TestGetPurchaseOrder(t *testing.T) {
	sid, wh := "s-1", "wh-1"
	name, code := "Farofa", "FAR-500"
	get := &stubPOGet{out: &appprocurement.PurchaseOrderDetail{
		PurchaseOrder: procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: &sid, WarehouseID: &wh, Status: procurementDomain.PurchaseOrderStatusPartiallyReceived,
			CurrencyCode: "EUR", Notes: "deliver monday", TotalMinorUnits: 7000, CreatedAt: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
		LineItems: []procurementDomain.PurchaseOrderLineItem{
			{ProductName: &name, SKUCode: &code, RawSKU: "100131", RawDescription: "FAROFA PREMIUM", Quantity: 10, UnitsPerPurchasedItem: 12,
				UnitCostMinorUnits: 600, LineType: procurementDomain.LineItemTypeProduct},
			{RawDescription: "Shipping", Quantity: 1, UnitCostMinorUnits: 1000, LineType: procurementDomain.LineItemType("shipping")},
		},
		Documents: []procurementDomain.PurchaseOrderDocument{{}},
	}}
	tool := tools.NewGetPurchaseOrderTool(get, someSuppliers(), stubWarehouses{out: []invDomain.Warehouse{{ID: "wh-1", Name: "Main"}}})
	assertStrictDefinition(t, tool, "get_purchase_order")

	out := call(t, tool, `{"purchase_order_id":"po-1"}`)

	assert.Equal(t, "po-1", get.gotID)
	assert.Equal(t, "partially_received", out["status"])
	assert.Equal(t, "Bauducco Foods", out["supplier"].(map[string]any)["name"])
	assert.Equal(t, "Main", out["warehouse"].(map[string]any)["name"])
	assert.EqualValues(t, 1, out["document_count"])
	lines := out["lines"].([]any)
	require.Len(t, lines, 2)
	l := lines[0].(map[string]any)
	assert.Equal(t, "Farofa", l["product_name"])
	assert.Equal(t, "FAR-500", l["sku_code"])
	assert.EqualValues(t, 12, l["units_per_purchased_item"])
	assert.EqualValues(t, 6000, l["line_total_minor_units"])
	assert.Equal(t, "shipping", lines[1].(map[string]any)["line_type"])

	assert.ErrorIs(t, callErr(t, tool, `{}`), tools.ErrInvalidArguments)
	get.err = errors.New("purchase order not found")
	assert.ErrorContains(t, callErr(t, tool, `{"purchase_order_id":"x"}`), "not found")
}
