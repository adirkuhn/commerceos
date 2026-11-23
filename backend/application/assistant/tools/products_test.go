package tools_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubGetProduct struct {
	out   *apppms.ProductDetail
	err   error
	gotID string
}

func (s *stubGetProduct) Execute(_ context.Context, id string) (*apppms.ProductDetail, error) {
	s.gotID = id
	return s.out, s.err
}

type stubStock struct {
	out    map[string]appinventory.StockLevel
	err    error
	gotIDs []string
}

func (s *stubStock) Execute(_ context.Context, ids []string) (map[string]appinventory.StockLevel, error) {
	s.gotIDs = ids
	return s.out, s.err
}

func productDetail() *apppms.ProductDetail {
	return &apppms.ProductDetail{
		Product: pmsDomain.Product{ID: "p-1", Name: "Doce de leite", Status: pmsDomain.ProductStatusActive, Tags: []string{"br"}},
		Variants: []apppms.VariantDetail{{
			Variant: pmsDomain.Variant{ID: "v-1", Name: "500g"},
			SKUs: []pmsDomain.SKU{{ID: "s-1", SKUCode: "DDL-500", Barcode: "7897754200014", PriceMinorUnits: 649,
				SalePriceMinorUnits: ptr(int64(549)), CurrencyCode: "EUR", CostPriceMinorUnits: ptr(int64(310)), UnitType: pmsDomain.SKUUnitTypeUnit}},
		}},
	}
}

func TestGetProductTool(t *testing.T) {
	get, stock := &stubGetProduct{out: productDetail()}, &stubStock{out: map[string]appinventory.StockLevel{"s-1": {SKUID: "s-1", Total: 15, Syncable: 12}}}
	tool := tools.NewGetProductTool(get, stock)
	assertStrictDefinition(t, tool, "get_product")

	got := call(t, tool, `{"product_id":"p-1"}`)

	assert.Equal(t, "p-1", get.gotID)
	assert.Equal(t, []string{"s-1"}, stock.gotIDs)
	assert.Equal(t, "Doce de leite", got["name"])
	assert.Equal(t, "active", got["status"])
	skus := got["skus"].([]any)
	require.Len(t, skus, 1)
	sku := skus[0].(map[string]any)
	assert.Equal(t, "500g", sku["variant"])
	assert.Equal(t, "DDL-500", sku["sku_code"])
	assert.EqualValues(t, 649, sku["price_minor_units"])
	assert.EqualValues(t, 549, sku["sale_price_minor_units"])
	assert.EqualValues(t, 310, sku["cost_price_minor_units"])
	assert.EqualValues(t, 15, sku["stock_on_hand"])
	assert.EqualValues(t, 12, sku["stock_sellable_on_channels"])
}

func TestGetProductTool_errors(t *testing.T) {
	assert.ErrorIs(t, callErr(t, tools.NewGetProductTool(&stubGetProduct{}, &stubStock{}), `{}`), tools.ErrInvalidArguments)
	assert.ErrorContains(t, callErr(t, tools.NewGetProductTool(&stubGetProduct{err: errors.New("product not found")}, &stubStock{}), `{"product_id":"x"}`), "product not found")
	assert.ErrorContains(t, callErr(t, tools.NewGetProductTool(&stubGetProduct{out: productDetail()}, &stubStock{err: errors.New("stock down")}), `{"product_id":"p-1"}`), "stock down")
}
