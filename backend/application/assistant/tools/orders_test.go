package tools_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	appsearch "github.com/adirkuhn/commerceos/application/search"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

const orderUUID = "8f14e45f-ceea-467a-9b2d-0d2c1b7a6f11"

type stubGetOrder struct {
	out   *appoms.GetSalesOrderResult
	err   error
	gotID string
}

func (s *stubGetOrder) Execute(_ context.Context, id string) (*appoms.GetSalesOrderResult, error) {
	s.gotID = id
	return s.out, s.err
}

type stubListOrders struct {
	out *appoms.ListSalesOrdersResult
	err error
	got appoms.ListSalesOrdersInput
}

func (s *stubListOrders) Execute(_ context.Context, in appoms.ListSalesOrdersInput) (*appoms.ListSalesOrdersResult, error) {
	s.got = in
	return s.out, s.err
}

func shippedOrder() *appoms.GetSalesOrderResult {
	paid := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	return &appoms.GetSalesOrderResult{
		SalesOrder: omsDomain.SalesOrder{
			ID: orderUUID, Number: "CO-12", Status: omsDomain.SalesOrderStatusShipped, CurrencyCode: "EUR",
			TotalMinorUnits: 2598, ShippingTotalMinorUnits: 495, TaxTotalMinorUnits: 214,
			Billing:            omsDomain.Address{FirstName: "Ana", LastName: "Souza", Email: "ana@x.test", City: "Utrecht", Country: "NL"},
			Shipping:           omsDomain.Address{FirstName: "Ana", LastName: "Souza", City: "Utrecht", Country: "NL"},
			PaymentMethodTitle: "iDEAL", TransactionID: "tx-secret-ish", DatePaid: &paid,
			ShippingMethodTitle: "PostNL", Fulfilled: true, Packed: true, OccurredAt: paid,
		},
		LineItems: []omsDomain.SalesOrderLineItem{{Name: "Doce de leite", SKUCode: "7897754200014", Quantity: 2, PriceMinorUnits: 649, TotalMinorUnits: 1298}},
	}
}

func TestGetSalesOrderTool_byID(t *testing.T) {
	get := &stubGetOrder{out: shippedOrder()}
	tool := tools.NewGetSalesOrderTool(get, &stubSearch{})
	assertStrictDefinition(t, tool, "get_sales_order")

	got := call(t, tool, `{"order":"`+orderUUID+`"}`)

	assert.Equal(t, orderUUID, get.gotID)
	assert.Equal(t, "CO-12", got["number"])
	assert.Equal(t, "shipped", got["status"])
	assert.EqualValues(t, 2598, got["total_minor_units"])
	assert.Equal(t, "EUR", got["currency_code"])
	assert.Equal(t, "Ana Souza", got["customer"].(map[string]any)["name"])
	assert.Equal(t, true, got["fulfilled"])
	assert.Equal(t, "PostNL", got["shipping_method"])
	assert.NotContains(t, got, "transaction_id")
	lines := got["lines"].([]any)
	require.Len(t, lines, 1)
	assert.Equal(t, map[string]any{"name": "Doce de leite", "sku_code": "7897754200014", "quantity": float64(2),
		"unit_price_minor_units": float64(649), "total_minor_units": float64(1298)}, lines[0])
}

func TestGetSalesOrderTool_byNumberUsesExactSearchMatch(t *testing.T) {
	get := &stubGetOrder{out: shippedOrder()}
	search := &stubSearch{out: &appsearch.GlobalSearchResult{SalesOrders: []omsDomain.SalesOrder{
		{ID: "other", Number: "CO-120"}, {ID: orderUUID, Number: "CO-12"},
	}}}

	got := call(t, tools.NewGetSalesOrderTool(get, search), `{"order":"#co-12"}`)

	assert.Equal(t, "co-12", search.got, "the # is dropped; search itself is case-insensitive")
	assert.Equal(t, orderUUID, get.gotID)
	assert.Equal(t, "CO-12", got["number"])
}

func TestGetSalesOrderTool_unknownNumberSuggestsCandidates(t *testing.T) {
	search := &stubSearch{out: &appsearch.GlobalSearchResult{SalesOrders: []omsDomain.SalesOrder{{ID: "o-9", Number: "CO-120"}}}}

	err := callErr(t, tools.NewGetSalesOrderTool(&stubGetOrder{}, search), `{"order":"CO-12"}`)

	assert.ErrorContains(t, err, "no order numbered")
	assert.ErrorContains(t, err, "CO-120")
}

func TestGetSalesOrderTool_errors(t *testing.T) {
	assert.ErrorIs(t, callErr(t, tools.NewGetSalesOrderTool(&stubGetOrder{}, &stubSearch{}), `{}`), tools.ErrInvalidArguments)
	err := callErr(t, tools.NewGetSalesOrderTool(&stubGetOrder{err: omsDomain.ErrSalesOrderNotFound}, &stubSearch{}), `{"order":"`+orderUUID+`"}`)
	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}

func TestListSalesOrdersTool(t *testing.T) {
	list := &stubListOrders{out: &appoms.ListSalesOrdersResult{Total: 42, SalesOrders: []omsDomain.SalesOrder{shippedOrder().SalesOrder}}}
	tool := tools.NewListSalesOrdersTool(list)
	assertStrictDefinition(t, tool, "list_sales_orders")

	got := call(t, tool, `{"status":"shipped","customer_id":"c-1","source":"bol","limit":50}`)

	assert.Equal(t, []omsDomain.SalesOrderStatus{omsDomain.SalesOrderStatusShipped}, list.got.Status)
	assert.Equal(t, []omsDomain.SalesOrderSource{omsDomain.SalesOrderSourceBol}, list.got.Source)
	assert.Equal(t, ptr("c-1"), list.got.CustomerID)
	assert.Equal(t, 20, list.got.PerPage, "limit is capped")
	assert.Equal(t, omsDomain.SalesOrderSortOccurredAt, list.got.SortBy)
	assert.Equal(t, omsDomain.SortDesc, list.got.SortDir)
	assert.EqualValues(t, 42, got["total"])
	assert.Equal(t, "CO-12", got["sales_orders"].([]any)[0].(map[string]any)["number"])
}

func TestListSalesOrdersTool_defaultsAndErrors(t *testing.T) {
	list := &stubListOrders{out: &appoms.ListSalesOrdersResult{}}
	call(t, tools.NewListSalesOrdersTool(list), `{}`)
	assert.Equal(t, 10, list.got.PerPage)
	assert.Nil(t, list.got.Status)

	assert.ErrorIs(t, callErr(t, tools.NewListSalesOrdersTool(list), `{"status":"lost"}`), tools.ErrInvalidArguments)
	assert.ErrorContains(t, callErr(t, tools.NewListSalesOrdersTool(&stubListOrders{err: errors.New("db down")}), `{}`), "db down")
}

type stubOrigins struct {
	out    map[string]integrationsDomain.SalesOrderOrigin
	err    error
	gotIDs []string
}

func (s *stubOrigins) Execute(_ context.Context, ids []string) (map[string]integrationsDomain.SalesOrderOrigin, error) {
	s.gotIDs = ids
	return s.out, s.err
}

func TestGetSalesOrderTool_includesTheOrdersOrigin(t *testing.T) {
	conn := "conn-woo"
	origins := &stubOrigins{out: map[string]integrationsDomain.SalesOrderOrigin{orderUUID: {
		SalesOrderID: orderUUID, Origin: integrationsDomain.OriginWooCommerce, ConnectionID: &conn, ExternalOrderID: "27270", ExternalKey: "wc_order_secret",
	}}}
	tool := tools.NewGetSalesOrderTool(&stubGetOrder{out: shippedOrder()}, &stubSearch{}).WithOrigins(origins)

	got := call(t, tool, `{"order":"`+orderUUID+`"}`)

	assert.Equal(t, []string{orderUUID}, origins.gotIDs)
	assert.Equal(t, map[string]any{"source": "woocommerce", "connection_id": "conn-woo", "external_order_id": "27270"}, got["origin"],
		"the shop's order key is a secret-ish reference and stays out")
}

func TestGetSalesOrderTool_orderWithoutOriginOmitsIt(t *testing.T) {
	tool := tools.NewGetSalesOrderTool(&stubGetOrder{out: shippedOrder()}, &stubSearch{}).WithOrigins(&stubOrigins{})

	got := call(t, tool, `{"order":"`+orderUUID+`"}`)

	assert.NotContains(t, got, "origin")
}

func TestGetSalesOrderTool_originLookupErrorFails(t *testing.T) {
	tool := tools.NewGetSalesOrderTool(&stubGetOrder{out: shippedOrder()}, &stubSearch{}).WithOrigins(&stubOrigins{err: errors.New("db down")})

	assert.ErrorContains(t, callErr(t, tool, `{"order":"`+orderUUID+`"}`), "db down")
}
