package pos_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genpos "github.com/adirkuhn/commerceos/api/pos"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

func splitPaidOrderBody(payments []genpos.CreatePOSSalesOrderPayment) genpos.CreatePOSSalesOrderRequest {
	return genpos.CreatePOSSalesOrderRequest{
		ExternalOrderId: "pos-1", WarehouseId: "wh-1", CurrencyCode: "EUR", TotalMinorUnits: 2000, OccurredAt: time.Now(),
		LineItems: []genpos.CreatePOSSalesOrderLineItem{{Name: strPtr("Gift box"), Quantity: 1}},
		Payments:  &payments,
	}
}

func cashPlusCardBody() []genpos.CreatePOSSalesOrderPayment {
	return []genpos.CreatePOSSalesOrderPayment{
		{Method: "Cash", AmountMinorUnits: 500},
		{Method: "Revolut", AmountMinorUnits: 1500, TransactionId: strPtr("rev-1")},
	}
}

func TestCreatePOSSalesOrder_passesEveryPaymentToTheUseCase(t *testing.T) {
	s := receivedOrderStubs()

	w := doRequest(newContractTestRouter(t, s), "POST", salesOrdersPath, splitPaidOrderBody(cashPlusCardBody()))

	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, omsDomain.SalesOrderPayments{
		{Method: "Cash", AmountMinorUnits: 500},
		{Method: "Revolut", AmountMinorUnits: 1500, TransactionID: "rev-1"},
	}, s.receiveOrder.gotInput.Payments)
}

func TestCreatePOSSalesOrder_withoutPaymentsPassesNone(t *testing.T) {
	s := receivedOrderStubs()

	w := doRequest(newContractTestRouter(t, s), "POST", salesOrdersPath, weighedOrderBody(350))

	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Empty(t, s.receiveOrder.gotInput.Payments)
}

func TestCreatePOSSalesOrder_contractRejectsAZeroAmountPayment(t *testing.T) {
	body := splitPaidOrderBody([]genpos.CreatePOSSalesOrderPayment{{Method: "Cash", AmountMinorUnits: 0}})

	w := doRequest(newContractTestRouter(t, receivedOrderStubs()), "POST", salesOrdersPath, body)

	assert.Equal(t, 400, w.Code)
}

func TestCreatePOSSalesOrder_contractRejectsAPaymentWithoutAMethod(t *testing.T) {
	body := splitPaidOrderBody([]genpos.CreatePOSSalesOrderPayment{{Method: "", AmountMinorUnits: 2000}})

	w := doRequest(newContractTestRouter(t, receivedOrderStubs()), "POST", salesOrdersPath, body)

	assert.Equal(t, 400, w.Code)
}

func TestCreatePOSSalesOrder_mapsAPaymentsTotalMismatchTo422WithItsCode(t *testing.T) {
	s := &stubs{receiveOrder: stubReceivePOSSalesOrder{err: omsDomain.ErrPaymentsTotalMismatch}}

	w := doRequest(newContractTestRouter(t, s), "POST", salesOrdersPath, splitPaidOrderBody(cashPlusCardBody()))

	require.Equal(t, 422, w.Code)
	var resp genpos.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "PAYMENTS_TOTAL_MISMATCH", *resp.Error.Code)
}

func TestCreatePOSSalesOrder_returnsTheOrdersPayments(t *testing.T) {
	s := receivedOrderStubs()
	s.receiveOrder.out.Payments = omsDomain.SalesOrderPayments{{Method: "Cash", AmountMinorUnits: 500}, {Method: "Revolut", AmountMinorUnits: 1500, TransactionID: "rev-1"}}

	w := doRequest(newContractTestRouter(t, s), "POST", salesOrdersPath, splitPaidOrderBody(cashPlusCardBody()))

	require.Equal(t, 200, w.Code, w.Body.String())
	var resp genpos.POSSalesOrderResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, []genpos.POSSalesOrderPayment{
		{Method: "Cash", AmountMinorUnits: 500, TransactionId: ""},
		{Method: "Revolut", AmountMinorUnits: 1500, TransactionId: "rev-1"},
	}, *resp.Data.Payments)
}
