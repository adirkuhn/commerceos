package tools_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
)

type stubSuppliers struct {
	out []suppliersDomain.Supplier
	err error
}

func (s stubSuppliers) Execute(context.Context) ([]suppliersDomain.Supplier, error) {
	return s.out, s.err
}

type stubPOList struct {
	out []procurementDomain.PurchaseOrder
	err error
}

func (s stubPOList) Execute(context.Context) ([]procurementDomain.PurchaseOrder, error) {
	return s.out, s.err
}

func someSuppliers() stubSuppliers {
	lead := 5
	return stubSuppliers{out: []suppliersDomain.Supplier{
		{ID: "s-1", Name: "Bauducco Foods", Email: "orders@bauducco.example", Phone: "+55 11", Country: "BR", LeadTimeDays: &lead,
			Status: suppliersDomain.SupplierStatusActive, Notes: "pallets only"},
		{ID: "s-2", Name: "Mercado Atacado", Email: "inkoop@mercado.example", Country: "NL", Status: suppliersDomain.SupplierStatusInactive},
	}}
}

func TestSearchSuppliers(t *testing.T) {
	tool := tools.NewSearchSuppliersTool(someSuppliers())
	assertStrictDefinition(t, tool, "search_suppliers")

	out := call(t, tool, `{"query":"BAUDUCCO"}`)
	list := out["suppliers"].([]any)
	require.Len(t, list, 1)
	s := list[0].(map[string]any)
	assert.Equal(t, "s-1", s["id"])
	assert.Equal(t, "orders@bauducco.example", s["email"])
	assert.EqualValues(t, 5, s["lead_time_days"])

	all := call(t, tool, `{}`)
	assert.Len(t, all["suppliers"], 2)
	assert.EqualValues(t, 2, all["total"])

	one := call(t, tool, `{"limit":1}`)
	assert.Len(t, one["suppliers"], 1)
	assert.EqualValues(t, 2, one["total"])
}

func TestSearchSuppliers_errors(t *testing.T) {
	assert.ErrorIs(t, callErr(t, tools.NewSearchSuppliersTool(someSuppliers()), `{"nope":1}`), tools.ErrInvalidArguments)
	err := callErr(t, tools.NewSearchSuppliersTool(stubSuppliers{err: errors.New("db down")}), `{}`)
	assert.ErrorContains(t, err, "db down")
}

func TestGetSupplier_withRecentPurchaseOrders(t *testing.T) {
	sid := "s-1"
	other := "s-2"
	base := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	var pos []procurementDomain.PurchaseOrder
	for i := 0; i < 7; i++ {
		pos = append(pos, procurementDomain.PurchaseOrder{ID: "po-" + string(rune('a'+i)), SupplierID: &sid,
			Status: procurementDomain.PurchaseOrderStatusSent, CurrencyCode: "EUR", TotalMinorUnits: int64(100 * i), CreatedAt: base.AddDate(0, 0, i)})
	}
	pos = append(pos, procurementDomain.PurchaseOrder{ID: "po-other", SupplierID: &other, CreatedAt: base})
	tool := tools.NewGetSupplierTool(someSuppliers(), stubPOList{out: pos})
	assertStrictDefinition(t, tool, "get_supplier")

	out := call(t, tool, `{"supplier_id":"s-1"}`)

	assert.Equal(t, "Bauducco Foods", out["name"])
	assert.Equal(t, "pallets only", out["notes"])
	recent := out["recent_purchase_orders"].([]any)
	require.Len(t, recent, 5)
	assert.Equal(t, "po-g", recent[0].(map[string]any)["id"], "newest first")
	assert.EqualValues(t, 600, recent[0].(map[string]any)["total_minor_units"])
	assert.EqualValues(t, 7, out["purchase_order_count"])
}

func TestGetSupplier_errors(t *testing.T) {
	tool := tools.NewGetSupplierTool(someSuppliers(), stubPOList{})
	assert.ErrorIs(t, callErr(t, tool, `{}`), tools.ErrInvalidArguments)
	assert.ErrorContains(t, callErr(t, tool, `{"supplier_id":"s-9"}`), "not found")
	assert.ErrorContains(t, callErr(t, tools.NewGetSupplierTool(someSuppliers(), stubPOList{err: errors.New("po boom")}), `{"supplier_id":"s-1"}`), "po boom")
}
