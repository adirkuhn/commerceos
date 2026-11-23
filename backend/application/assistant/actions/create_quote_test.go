package actions_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appassistant "github.com/adirkuhn/commerceos/application/assistant"
	"github.com/adirkuhn/commerceos/application/assistant/actions"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	"github.com/adirkuhn/commerceos/domain/assistant"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/quotes"
)

func priceRows() *apppms.ListPriceListResult {
	return &apppms.ListPriceListResult{PricesIncludeTax: true, Rows: []apppms.PriceListRow{
		{ProductID: "p-1", ProductName: "Farofa", VariantName: "500g", SKUID: "s-1", SKUCode: "FAR-500", CurrencyCode: "EUR", PriceMinorUnits: 899},
		{ProductID: "p-2", ProductName: "Doce de leite", SKUID: "s-2", SKUCode: "DDL", CurrencyCode: "EUR", PriceMinorUnits: 649, SalePriceMinorUnits: ptr(int64(599))},
		{ProductID: "p-3", ProductName: "Imported", SKUID: "s-3", CurrencyCode: "USD", PriceMinorUnits: 100},
	}}
}

type createFixture struct {
	prices    *stubPrices
	customers *stubCustomers
	create    *stubCreateQuote
	action    *actions.CreateQuoteAction
}

func newCreateFixture() *createFixture {
	f := &createFixture{
		prices: &stubPrices{out: priceRows()},
		customers: &stubCustomers{byID: map[string]*customersDomain.Customer{
			"c-1": {ID: "c-1", CompanyName: "Ana Bakery", Type: customersDomain.TypeCompany, Email: "ana@x.test"},
		}},
		create: &stubCreateQuote{},
	}
	f.action = actions.NewCreateQuoteAction(f.prices, f.customers, f.create, clock)
	return f
}

func TestCreateQuoteAction_definition(t *testing.T) {
	a := newCreateFixture().action

	assert.Equal(t, assistant.KindCreateQuote, a.Kind())
	d := a.Tool()
	assert.Equal(t, "propose_quote", d.Name)
	assertStrictSchema(t, d, "name", "lines")
	assert.Contains(t, d.Description, "confirm")
}

func TestCreateQuoteAction_preparePricesLikeTheQuoteEditor(t *testing.T) {
	f := newCreateFixture()

	payload, preview, err := f.action.Prepare(userCtx(), json.RawMessage(`{
		"name":"Ana Bakery","customer_id":"c-1","discount_percent":12.5,
		"lines":[
			{"product_id":"p-1","sku_id":"s-1","quantity":10},
			{"product_id":"p-2","sku_id":"s-2","quantity":3},
			{"product_id":"p-1","sku_id":"s-1b","quantity":1,"unit_price_minor_units":500}
		]}`))
	require.Error(t, err, "s-1b isn't a SKU of p-1")
	assert.Contains(t, err.Error(), "line 3")

	payload, preview, err = f.action.Prepare(userCtx(), json.RawMessage(`{
		"name":"Ana Bakery","customer_id":"c-1","discount_percent":12.5,
		"lines":[
			{"product_id":"p-1","sku_id":"s-1","quantity":10},
			{"product_id":"p-2","sku_id":"s-2","quantity":3,"unit_price_minor_units":550}
		]}`))
	require.NoError(t, err)
	assert.Equal(t, []string{"p-1", "p-2"}, f.prices.got.ProductIDs)

	// 899 * 0.875 = 786.625 -> 787 (rounded like the editor); the stated
	// price wins over the sale price.
	assert.JSONEq(t, `{"customer_id":"c-1","name":"Ana Bakery","discount_basis_points":1250,"currency_code":"EUR",
		"lines":[{"product_id":"p-1","sku_id":"s-1","quantity":10,"unit_price_minor_units":787},
		         {"product_id":"p-2","sku_id":"s-2","quantity":3,"unit_price_minor_units":550}]}`, string(payload))

	p := decodeJSON(t, preview)
	assert.Equal(t, map[string]any{"id": "c-1", "name": "Ana Bakery", "email": "ana@x.test"}, p["customer"])
	assert.Equal(t, "EUR", p["currency_code"])
	assert.Equal(t, true, p["prices_include_tax"])
	assert.EqualValues(t, 787*10+550*3, p["total_minor_units"])
	lines := p["lines"].([]any)
	require.Len(t, lines, 2)
	assert.Equal(t, map[string]any{"product_id": "p-1", "sku_id": "s-1", "product_name": "Farofa", "variant_name": "500g", "sku_code": "FAR-500",
		"quantity": float64(10), "list_price_minor_units": float64(899), "unit_price_minor_units": float64(787), "line_total_minor_units": float64(7870)}, lines[0])
	assert.EqualValues(t, 599, lines[1].(map[string]any)["list_price_minor_units"], "the running sale price is the list price")
}

func TestCreateQuoteAction_prepareUsesSalePriceWithoutCustomer(t *testing.T) {
	f := newCreateFixture()

	payload, preview, err := f.action.Prepare(userCtx(), json.RawMessage(`{"name":"Walk-in","lines":[{"product_id":"p-2","sku_id":"s-2","quantity":2}]}`))

	require.NoError(t, err)
	assert.JSONEq(t, `{"customer_id":null,"name":"Walk-in","discount_basis_points":0,"currency_code":"EUR",
		"lines":[{"product_id":"p-2","sku_id":"s-2","quantity":2,"unit_price_minor_units":599}]}`, string(payload))
	assert.Nil(t, decodeJSON(t, preview)["customer"])
}

func TestCreateQuoteAction_prepareErrors(t *testing.T) {
	cases := map[string]struct {
		args    string
		wantErr error
		wantMsg string
	}{
		"bad json":         {args: `{"name":`, wantErr: actions.ErrInvalidArguments},
		"unknown field":    {args: `{"name":"x","lines":[],"total":1}`, wantErr: actions.ErrInvalidArguments},
		"no lines":         {args: `{"name":"x","lines":[]}`, wantErr: actions.ErrInvalidArguments},
		"no name":          {args: `{"name":"","lines":[{"product_id":"p-1","sku_id":"s-1","quantity":1}]}`, wantErr: actions.ErrInvalidArguments},
		"zero quantity":    {args: `{"name":"x","lines":[{"product_id":"p-1","sku_id":"s-1","quantity":0}]}`, wantErr: actions.ErrInvalidArguments},
		"discount > 100":   {args: `{"name":"x","discount_percent":120,"lines":[{"product_id":"p-1","sku_id":"s-1","quantity":1}]}`, wantErr: actions.ErrInvalidArguments},
		"negative price":   {args: `{"name":"x","lines":[{"product_id":"p-1","sku_id":"s-1","quantity":1,"unit_price_minor_units":-1}]}`, wantErr: actions.ErrInvalidArguments},
		"mixed currencies": {args: `{"name":"x","lines":[{"product_id":"p-1","sku_id":"s-1","quantity":1},{"product_id":"p-3","sku_id":"s-3","quantity":1}]}`, wantMsg: "one currency"},
		"unknown customer": {args: `{"name":"x","customer_id":"c-9","lines":[{"product_id":"p-1","sku_id":"s-1","quantity":1}]}`, wantErr: customersDomain.ErrCustomerNotFound},
		"duplicate sku":    {args: `{"name":"x","lines":[{"product_id":"p-1","sku_id":"s-1","quantity":1},{"product_id":"p-1","sku_id":"s-1","quantity":2}]}`, wantErr: quotes.ErrDuplicateLine},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newCreateFixture()

			_, _, err := f.action.Prepare(userCtx(), json.RawMessage(tc.args))

			require.Error(t, err)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			}
			if tc.wantMsg != "" {
				assert.Contains(t, err.Error(), tc.wantMsg)
			}
			assert.Nil(t, f.create.got, "Prepare never creates anything")
		})
	}
}

func TestCreateQuoteAction_preparePriceListError(t *testing.T) {
	f := newCreateFixture()
	f.prices.err = errors.New("db down")

	_, _, err := f.action.Prepare(userCtx(), json.RawMessage(`{"name":"x","lines":[{"product_id":"p-1","sku_id":"s-1","quantity":1}]}`))

	assert.ErrorContains(t, err, "db down")
}

func TestCreateQuoteAction_executeCreatesAndOffersConversion(t *testing.T) {
	f := newCreateFixture()
	payload, _, err := f.action.Prepare(userCtx(), json.RawMessage(`{"name":"Ana","customer_id":"c-1","lines":[{"product_id":"p-1","sku_id":"s-1","quantity":2}]}`))
	require.NoError(t, err)

	res, err := f.action.Execute(userCtx(), payload, appassistant.ConfirmChoices{})

	require.NoError(t, err)
	require.NotNil(t, f.create.got)
	assert.Equal(t, "c-1", *f.create.got.CustomerID)
	assert.Equal(t, []quotes.Line{{ProductID: "p-1", SKUID: "s-1", Quantity: 2, UnitPriceMinorUnits: 899}}, f.create.got.Lines)
	assert.Equal(t, "EUR", f.create.got.CurrencyCode)
	assert.Equal(t, "q-1", res.ID)
	assert.Equal(t, "Q-0012", res.Ref)
	require.NotNil(t, res.FollowUp)
	assert.Equal(t, assistant.KindConvertQuote, res.FollowUp.Kind)
	assert.JSONEq(t, `{"quote_id":"q-1"}`, string(res.FollowUp.Args))
}

func TestCreateQuoteAction_executeErrors(t *testing.T) {
	f := newCreateFixture()
	f.create.err = quotes.ErrInvalidCurrency

	_, err := f.action.Execute(userCtx(), json.RawMessage(`{"name":"x","lines":[]}`), appassistant.ConfirmChoices{})
	assert.ErrorIs(t, err, quotes.ErrInvalidCurrency)

	_, err = f.action.Execute(userCtx(), json.RawMessage(`not json`), appassistant.ConfirmChoices{})
	assert.Error(t, err)
}
