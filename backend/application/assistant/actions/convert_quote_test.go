package actions_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appassistant "github.com/adirkuhn/commerceos/application/assistant"
	"github.com/adirkuhn/commerceos/application/assistant/actions"
	appquotes "github.com/adirkuhn/commerceos/application/quotes"
	"github.com/adirkuhn/commerceos/domain/assistant"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/quotes"
)

type convertFixture struct {
	prices     *stubPrices
	quotes     *stubGetQuote
	warehouses *stubWarehouses
	stock      *stubStock
	convert    *stubConvert
	action     *actions.ConvertQuoteAction
}

func newConvertFixture() *convertFixture {
	f := &convertFixture{
		prices: &stubPrices{out: priceRows()},
		quotes: &stubGetQuote{byID: map[string]*quotes.Quote{
			"q-1": {ID: "q-1", Number: 12, Name: "Ana Bakery", CurrencyCode: "EUR",
				Customer: &quotes.CustomerRef{ID: "c-1", Name: "Ana Bakery", Email: "ana@x.test"},
				Lines: []quotes.Line{
					{ProductID: "p-1", SKUID: "s-1", Quantity: 10, UnitPriceMinorUnits: 787},
					{ProductID: "p-2", SKUID: "s-2", Quantity: 3, UnitPriceMinorUnits: 550},
				}},
			"q-empty": {ID: "q-empty", Number: 13, Name: "Empty"},
			"q-gone": {ID: "q-gone", Number: 14, Name: "Old", CurrencyCode: "EUR",
				Lines: []quotes.Line{{ProductID: "p-9", SKUID: "s-9", Quantity: 2, UnitPriceMinorUnits: 100}}},
		}},
		warehouses: &stubWarehouses{out: []invDomain.Warehouse{{ID: "w-1", Name: "Main"}, {ID: "w-2", Name: "Overflow"}}},
		stock: &stubStock{out: &appquotes.StockCheck{Lines: []appquotes.StockCheckLine{
			{ProductID: "p-1", SKUID: "s-1", Quantity: 10, Available: 4, Short: 6},
			{ProductID: "p-2", SKUID: "s-2", Quantity: 3, Available: 9},
		}}},
		convert: &stubConvert{},
	}
	f.action = actions.NewConvertQuoteAction(f.prices, f.quotes, f.warehouses, f.stock, f.convert)
	return f
}

func TestConvertQuoteAction_definition(t *testing.T) {
	a := newConvertFixture().action

	assert.Equal(t, assistant.KindConvertQuote, a.Kind())
	d := a.Tool()
	assert.Equal(t, "propose_quote_conversion", d.Name)
	assertStrictSchema(t, d, "quote_id")
}

func TestConvertQuoteAction_prepareUsesFirstWarehouseAndShowsShortfall(t *testing.T) {
	f := newConvertFixture()

	payload, preview, err := f.action.Prepare(userCtx(), json.RawMessage(`{"quote_id":"q-1"}`))

	require.NoError(t, err)
	assert.JSONEq(t, `{"quote_id":"q-1","warehouse_id":"w-1"}`, string(payload))
	assert.Equal(t, "q-1", f.stock.gotQuote)
	assert.Equal(t, "w-1", f.stock.gotWhouse)
	assert.JSONEq(t, `{
		"quote_id":"q-1","quote_ref":"Q-0012","name":"Ana Bakery",
		"customer":{"id":"c-1","name":"Ana Bakery","email":"ana@x.test"},
		"line_count":2,"total_minor_units":9520,"currency_code":"EUR","warehouse_id":"w-1",
		"warehouses":[{"id":"w-1","name":"Main"},{"id":"w-2","name":"Overflow"}],
		"lines":[
			{"product_id":"p-1","sku_id":"s-1","product_name":"Farofa","variant_name":"500g","sku_code":"FAR-500",
			 "quantity":10,"unit_price_minor_units":787,"line_total_minor_units":7870},
			{"product_id":"p-2","sku_id":"s-2","product_name":"Doce de leite","variant_name":"","sku_code":"DDL",
			 "quantity":3,"unit_price_minor_units":550,"line_total_minor_units":1650}
		],
		"shortfall":[{"product_id":"p-1","sku_id":"s-1","product_name":"Farofa","variant_name":"500g","sku_code":"FAR-500",
			"requested":10,"available":4,"short":6}]
	}`, string(preview))
	assert.Equal(t, []string{"p-1", "p-2"}, f.prices.got.ProductIDs)
	assert.Nil(t, f.convert.got, "Prepare never converts")
}

func TestConvertQuoteAction_prepareErrors(t *testing.T) {
	cases := map[string]struct {
		args    string
		setup   func(f *convertFixture)
		wantErr error
		wantMsg string
	}{
		"no args":        {args: ``, wantErr: actions.ErrInvalidArguments},
		"empty quote_id": {args: `{"quote_id":""}`, wantErr: actions.ErrInvalidArguments},
		"unknown field":  {args: `{"quote_id":"q-1","warehouse":"x"}`, wantErr: actions.ErrInvalidArguments},
		"unknown quote":  {args: `{"quote_id":"q-9"}`, wantErr: quotes.ErrQuoteNotFound},
		"empty quote":    {args: `{"quote_id":"q-empty"}`, wantErr: actions.ErrEmptyQuote},
		"no warehouse": {args: `{"quote_id":"q-1"}`, wantErr: actions.ErrNoWarehouse,
			setup: func(f *convertFixture) { f.warehouses.out = nil }},
		"warehouses fail": {args: `{"quote_id":"q-1"}`, wantMsg: "db down",
			setup: func(f *convertFixture) { f.warehouses.err = errors.New("db down") }},
		"price list fails": {args: `{"quote_id":"q-1"}`, wantMsg: "prices down",
			setup: func(f *convertFixture) { f.prices.err = errors.New("prices down") }},
		"stock check fails": {args: `{"quote_id":"q-1"}`, wantMsg: "stock down",
			setup: func(f *convertFixture) { f.stock.err = errors.New("stock down") }},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newConvertFixture()
			if tc.setup != nil {
				tc.setup(f)
			}

			_, _, err := f.action.Prepare(userCtx(), json.RawMessage(tc.args))

			require.Error(t, err)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			}
			if tc.wantMsg != "" {
				assert.Contains(t, err.Error(), tc.wantMsg)
			}
		})
	}
}

func TestConvertQuoteAction_executeUsesTheChosenWarehouse(t *testing.T) {
	f := newConvertFixture()
	payload := json.RawMessage(`{"quote_id":"q-1","warehouse_id":"w-1"}`)

	res, err := f.action.Execute(userCtx(), payload, appassistant.ConfirmChoices{})
	require.NoError(t, err)
	assert.Equal(t, appquotes.ConvertQuoteInput{QuoteID: "q-1", WarehouseID: "w-1"}, *f.convert.got)
	assert.Equal(t, appassistant.ActionResult{ID: "so-1", Ref: "CO-31"}, res, "an order has no follow-up")

	_, err = f.action.Execute(userCtx(), payload, appassistant.ConfirmChoices{WarehouseID: "w-2"})
	require.NoError(t, err)
	assert.Equal(t, "w-2", f.convert.got.WarehouseID)
}

func TestConvertQuoteAction_executeErrors(t *testing.T) {
	f := newConvertFixture()
	f.convert.err = quotes.ErrQuoteNotFound

	_, err := f.action.Execute(userCtx(), json.RawMessage(`{"quote_id":"q-1","warehouse_id":"w-1"}`), appassistant.ConfirmChoices{})
	assert.ErrorIs(t, err, quotes.ErrQuoteNotFound)

	_, err = f.action.Execute(userCtx(), json.RawMessage(`[`), appassistant.ConfirmChoices{})
	assert.Error(t, err)
}

func TestConvertQuoteAction_prepareKeepsLinesWhoseSKUIsNoLongerListed(t *testing.T) {
	f := newConvertFixture()
	f.stock.out = &appquotes.StockCheck{Lines: []appquotes.StockCheckLine{{ProductID: "p-9", SKUID: "s-9", Quantity: 2, Short: 2}}}

	_, preview, err := f.action.Prepare(userCtx(), json.RawMessage(`{"quote_id":"q-gone"}`))

	require.NoError(t, err)
	p := decodeJSON(t, preview)
	assert.Equal(t, map[string]any{"product_id": "p-9", "sku_id": "s-9", "product_name": "", "variant_name": "", "sku_code": "",
		"quantity": float64(2), "unit_price_minor_units": float64(100), "line_total_minor_units": float64(200)}, p["lines"].([]any)[0])
	assert.Equal(t, "", p["shortfall"].([]any)[0].(map[string]any)["product_name"])
}
