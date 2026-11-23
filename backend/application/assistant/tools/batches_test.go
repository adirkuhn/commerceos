package tools_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

type stubBatches struct {
	out    []invDomain.Batch
	err    error
	gotSKU string
}

func (s *stubBatches) Execute(_ context.Context, skuID string) ([]invDomain.Batch, error) {
	s.gotSKU = skuID
	return s.out, s.err
}

type stubExpiring struct {
	out     []invDomain.ExpiringBatch
	err     error
	gotDays int
}

func (s *stubExpiring) Execute(_ context.Context, withinDays int) ([]invDomain.ExpiringBatch, error) {
	s.gotDays = withinDays
	return s.out, s.err
}

func TestListBatches(t *testing.T) {
	exp := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	batches := &stubBatches{out: []invDomain.Batch{
		{WarehouseID: "wh-1", IsGeneralLedger: true, Quantity: 4},
		{WarehouseID: "wh-2", IsGeneralLedger: true, Quantity: 0},
		{ID: "b-1", WarehouseID: "wh-1", LotNumber: "PT26004", ExpiryDate: &exp, Quantity: 20, Status: invDomain.BatchStatusActive},
	}}
	tool := tools.NewListBatchesTool(batches, stubWarehouses{out: []invDomain.Warehouse{{ID: "wh-1", Name: "Main"}, {ID: "wh-2", Name: "Overflow"}}})
	assertStrictDefinition(t, tool, "list_batches")

	out := call(t, tool, `{"sku_id":"sku-1"}`)

	assert.Equal(t, "sku-1", batches.gotSKU)
	rows := out["batches"].([]any)
	require.Len(t, rows, 2, "empty un-lotted rows are left out")
	general := rows[0].(map[string]any)
	assert.Equal(t, true, general["unlotted"])
	assert.Equal(t, "Main", general["warehouse"].(map[string]any)["name"])
	lot := rows[1].(map[string]any)
	assert.Equal(t, "PT26004", lot["lot_number"])
	assert.Equal(t, "2027-03-01", lot["expiry_date"])
	assert.EqualValues(t, 20, lot["quantity"])
	assert.Equal(t, "active", lot["status"])
	assert.EqualValues(t, 24, out["total_quantity"])

	assert.ErrorIs(t, callErr(t, tool, `{}`), tools.ErrInvalidArguments)
	batches.err = errors.New("boom")
	assert.ErrorContains(t, callErr(t, tool, `{"sku_id":"sku-1"}`), "boom")
}

func TestListExpiringStock(t *testing.T) {
	soon := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	later := soon.AddDate(0, 0, 5)
	exp := &stubExpiring{out: []invDomain.ExpiringBatch{
		{BatchID: "b-2", SKUCode: "X", ProductName: "Late", WarehouseName: "Main", LotNumber: "L2", ExpiryDate: later, Quantity: 3},
		{BatchID: "b-1", SKUID: "sku-1", SKUCode: "FAR-500", ProductName: "Farofa", WarehouseID: "wh-1", WarehouseName: "Main", LotNumber: "L1", ExpiryDate: soon, Quantity: 7},
	}}
	tool := tools.NewListExpiringStockTool(exp)
	assertStrictDefinition(t, tool, "list_expiring_stock")

	out := call(t, tool, `{}`)
	assert.Equal(t, 30, exp.gotDays, "default window")
	rows := out["batches"].([]any)
	require.Len(t, rows, 2)
	first := rows[0].(map[string]any)
	assert.Equal(t, "Farofa", first["product_name"], "soonest first")
	assert.Equal(t, "2026-12-01", first["expiry_date"])
	assert.Equal(t, "Main", first["warehouse_name"])

	limited := call(t, tool, `{"within_days":7,"limit":1}`)
	assert.Equal(t, 7, exp.gotDays)
	assert.Len(t, limited["batches"], 1)
	assert.EqualValues(t, 2, limited["total"])

	assert.ErrorIs(t, callErr(t, tool, `{"within_days":0}`), tools.ErrInvalidArguments)
	assert.ErrorIs(t, callErr(t, tool, `{"within_days":400}`), tools.ErrInvalidArguments)
}
