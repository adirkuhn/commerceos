package tools_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

type stubStockWarehouses struct {
	out []invDomain.Warehouse
	err error
}

func (s *stubStockWarehouses) Execute(context.Context) ([]invDomain.Warehouse, error) { return s.out, s.err }

type stubScopedStock struct {
	qty map[string]int64 // sku|warehouse -> qty
	err error
}

func (s *stubScopedStock) Execute(_ context.Context, skuID, warehouseID string, batchID *string) (int64, error) {
	if batchID != nil {
		return 0, errors.New("batch scope not expected")
	}
	return s.qty[skuID+"|"+warehouseID], s.err
}

func stockWarehouses() *stubStockWarehouses {
	return &stubStockWarehouses{out: []invDomain.Warehouse{{ID: "w-1", Name: "Main"}, {ID: "w-2", Name: "Overflow"}}}
}

func TestGetStockByWarehouseTool_forAProduct(t *testing.T) {
	scoped := &stubScopedStock{qty: map[string]int64{"s-1|w-1": 12, "s-1|w-2": 3}}
	stock := &stubStock{out: map[string]appinventory.StockLevel{"s-1": {SKUID: "s-1", Total: 15, Syncable: 10}}}
	tool := tools.NewGetStockByWarehouseTool(&stubGetProduct{out: productDetail()}, stockWarehouses(), scoped, stock)
	assertStrictDefinition(t, tool, "get_stock_by_warehouse")

	got := call(t, tool, `{"product_id":"p-1"}`)

	assert.Equal(t, "Doce de leite", got["product_name"])
	skus := got["skus"].([]any)
	require.Len(t, skus, 1)
	sku := skus[0].(map[string]any)
	assert.Equal(t, "DDL-500", sku["sku_code"])
	assert.Equal(t, "500g", sku["variant"])
	assert.EqualValues(t, 15, sku["total_on_hand"])
	assert.EqualValues(t, 10, sku["sellable_on_channels"])
	assert.Equal(t, []any{
		map[string]any{"warehouse_id": "w-1", "warehouse": "Main", "on_hand": float64(12)},
		map[string]any{"warehouse_id": "w-2", "warehouse": "Overflow", "on_hand": float64(3)},
	}, sku["warehouses"])
}

func TestGetStockByWarehouseTool_forOneSKU(t *testing.T) {
	scoped := &stubScopedStock{qty: map[string]int64{"s-1|w-1": 4}}
	stock := &stubStock{out: map[string]appinventory.StockLevel{"s-1": {SKUID: "s-1", Total: 4, Syncable: 4}}}
	product := &stubGetProduct{out: productDetail()}
	tool := tools.NewGetStockByWarehouseTool(product, stockWarehouses(), scoped, stock)

	got := call(t, tool, `{"product_id":"p-1","sku_id":"s-1"}`)
	assert.Len(t, got["skus"], 1)

	got = call(t, tool, `{"sku_id":"s-1"}`)
	sku := got["skus"].([]any)[0].(map[string]any)
	assert.Equal(t, "s-1", sku["sku_id"])
	assert.NotContains(t, got, "product_name", "only the SKU id is known without the product")
	assert.Equal(t, []string{"s-1"}, stock.gotIDs)
}

func TestGetStockByWarehouseTool_errors(t *testing.T) {
	ok := func() (*stubGetProduct, *stubStockWarehouses, *stubScopedStock, *stubStock) {
		return &stubGetProduct{out: productDetail()}, stockWarehouses(), &stubScopedStock{}, &stubStock{}
	}
	p, w, s, l := ok()
	assert.ErrorIs(t, callErr(t, tools.NewGetStockByWarehouseTool(p, w, s, l), `{}`), tools.ErrInvalidArguments)
	assert.ErrorIs(t, callErr(t, tools.NewGetStockByWarehouseTool(p, w, s, l), `{"product_id":"p-1","sku_id":"s-9"}`), tools.ErrInvalidArguments)

	p, w, s, l = ok()
	p.err = errors.New("product down")
	assert.ErrorContains(t, callErr(t, tools.NewGetStockByWarehouseTool(p, w, s, l), `{"product_id":"p-1"}`), "product down")

	p, w, s, l = ok()
	w.err = errors.New("warehouses down")
	assert.ErrorContains(t, callErr(t, tools.NewGetStockByWarehouseTool(p, w, s, l), `{"product_id":"p-1"}`), "warehouses down")

	p, w, s, l = ok()
	s.err = errors.New("movements down")
	assert.ErrorContains(t, callErr(t, tools.NewGetStockByWarehouseTool(p, w, s, l), `{"product_id":"p-1"}`), "movements down")

	p, w, s, l = ok()
	l.err = errors.New("levels down")
	assert.ErrorContains(t, callErr(t, tools.NewGetStockByWarehouseTool(p, w, s, l), `{"product_id":"p-1"}`), "levels down")
}
