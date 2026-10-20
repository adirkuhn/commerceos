package oms_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

func getSalesOrderDetail(t *testing.T, order omsDomain.SalesOrder) genoms.SalesOrderDetail {
	t.Helper()
	get := &stubGetSalesOrder{out: &appoms.GetSalesOrderResult{SalesOrder: order}}
	w := doRequest(newTestRouter(&stubListSalesOrders{}, get, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}), "GET", "/api/v1/sales-orders/so-1", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp genoms.SalesOrderDetailResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return *resp.Data
}

func TestGetSalesOrder_includesEveryPayment(t *testing.T) {
	order := omsDomain.SalesOrder{ID: "so-1", Payments: omsDomain.SalesOrderPayments{
		{Method: "Cash", AmountMinorUnits: 500},
		{Method: "Revolut", AmountMinorUnits: 1500, TransactionID: "rev-1"},
	}}

	detail := getSalesOrderDetail(t, order)

	assert.Equal(t, []genoms.SalesOrderPayment{
		{Method: "Cash", AmountMinorUnits: 500, TransactionId: ""},
		{Method: "Revolut", AmountMinorUnits: 1500, TransactionId: "rev-1"},
	}, detail.Payments)
}

func TestGetSalesOrder_listsNoPaymentsForAnOrderWithout(t *testing.T) {
	detail := getSalesOrderDetail(t, omsDomain.SalesOrder{ID: "so-1"})

	assert.Equal(t, []genoms.SalesOrderPayment{}, detail.Payments)
}
