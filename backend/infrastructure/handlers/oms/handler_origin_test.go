package oms_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// COM-15: the order holds no integration data, the API response still
// carries source, external_order_id and order_key -- from the order's
// origin, so the contract and the frontend are unchanged.
func TestListSalesOrders_responseCarriesEachOrdersOrigin(t *testing.T) {
	conn := "conn-1"
	testOrigins.byOrder = map[string]integrationsDomain.SalesOrderOrigin{
		"so-1": {SalesOrderID: "so-1", Origin: integrationsDomain.OriginWooCommerce, ConnectionID: &conn, ExternalOrderID: "26217", ExternalKey: "wc_order_k"},
		"so-2": {SalesOrderID: "so-2", Origin: integrationsDomain.OriginManual},
	}
	t.Cleanup(func() { testOrigins.byOrder = nil })
	list := &stubListSalesOrders{out: &appoms.ListSalesOrdersResult{
		SalesOrders: []omsDomain.SalesOrder{{ID: "so-1", Number: "26217"}, {ID: "so-2", Number: "CO-1001"}}, Total: 2,
	}}

	w := doRequest(newTestRouter(list, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}),
		"GET", "/api/v1/sales-orders", nil)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Data struct {
			SalesOrders []map[string]any `json:"sales_orders"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data.SalesOrders, 2)
	woo, manual := body.Data.SalesOrders[0], body.Data.SalesOrders[1]
	assert.Equal(t, "woocommerce", woo["source"])
	assert.Equal(t, "26217", woo["external_order_id"])
	assert.Equal(t, "wc_order_k", woo["order_key"])
	assert.Equal(t, "manual", manual["source"])
	assert.Equal(t, "", manual["external_order_id"])
}

func TestListSalesOrders_anOriginLookupFailureIsAnError(t *testing.T) {
	testOrigins.err = errors.New("db down")
	t.Cleanup(func() { testOrigins.err = nil })
	list := &stubListSalesOrders{out: &appoms.ListSalesOrdersResult{SalesOrders: []omsDomain.SalesOrder{{ID: "so-1"}}, Total: 1}}

	w := doRequest(newTestRouter(list, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}),
		"GET", "/api/v1/sales-orders", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
