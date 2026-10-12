package oms_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

func TestListSalesOrders_passesShipDueThrough(t *testing.T) {
	list := &stubListSalesOrders{out: &appoms.ListSalesOrdersResult{}}
	w := doRequest(newTestRouter(list, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}),
		"GET", "/api/v1/sales-orders?ship_due=true", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, list.gotInput.ShipDue)
}

func TestListSalesOrders_invalidShipDueIsBadRequest(t *testing.T) {
	list := &stubListSalesOrders{out: &appoms.ListSalesOrdersResult{}}
	w := doRequest(newTestRouter(list, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}),
		"GET", "/api/v1/sales-orders?ship_due=maybe", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, list.gotInput.ShipDue)
}

// Guards toGenSalesOrder against dropping ship_by_date (same failure mode
// TestListSalesOrders_includesFulfilled pins): set -> a plain date, nil ->
// absent.
func TestListSalesOrders_includesShipByDate(t *testing.T) {
	shipBy := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	list := &stubListSalesOrders{out: &appoms.ListSalesOrdersResult{
		SalesOrders: []omsDomain.SalesOrder{
			{ID: "so-1", ExternalOrderID: "bol-1", Status: omsDomain.SalesOrderStatusProcessing, ShipByDate: &shipBy},
			{ID: "so-2", ExternalOrderID: "woo-2", Status: omsDomain.SalesOrderStatusProcessing},
		},
		Total: 2,
	}}
	w := doRequest(newTestRouter(list, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "GET", "/api/v1/sales-orders", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"ship_by_date":"2026-10-14"`)
	assert.Equal(t, 1, strings.Count(w.Body.String(), `"ship_by_date"`))
}
