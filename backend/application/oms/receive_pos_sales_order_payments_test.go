package oms_test

import (
	"context"
	"testing"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const paidOrderTotal int64 = 2000

type paymentsFixture struct {
	salesOrderRepo *stubSalesOrderRepo
	paymentRepo    *stubSalesOrderPaymentRepo
	uc             *appoms.ReceivePOSSalesOrderUseCase
}

func newPaymentsFixture(t *testing.T, salesOrderRepo *stubSalesOrderRepo, paymentRepo *stubSalesOrderPaymentRepo) paymentsFixture {
	t.Helper()
	db := newMockGormDBN(t, 1) // order create; the adhoc line moves no stock
	origins := &stubOrigins{}
	if salesOrderRepo.findByIDResult != nil {
		origins = receivedPOSSale(salesOrderRepo.findByIDResult.ID, "pos-order-1")
	}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, salesOrderRepo, origins, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, paymentRepo, &stubSKURepo{}, &stubRecordMovement{}, &eventrecordtest.Outbox{})
	return paymentsFixture{salesOrderRepo: salesOrderRepo, paymentRepo: paymentRepo, uc: uc}
}

func newPaymentsFixtureWithFreshOrder(t *testing.T) paymentsFixture {
	return newPaymentsFixture(t, &stubSalesOrderRepo{}, &stubSalesOrderPaymentRepo{})
}

func paidOrderInput(payments omsDomain.SalesOrderPayments) appoms.ReceivePOSSalesOrderInput {
	return appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "EUR", OccurredAt: time.Now(),
		TotalMinorUnits: paidOrderTotal, Payments: payments,
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{Name: "Gift box", Quantity: 1, PriceMinorUnits: paidOrderTotal, TotalMinorUnits: paidOrderTotal}},
	}
}

func cashPlusCard() omsDomain.SalesOrderPayments {
	return omsDomain.SalesOrderPayments{
		{Method: "Cash", AmountMinorUnits: 500},
		{Method: "Revolut", AmountMinorUnits: 1500, TransactionID: "rev-1"},
	}
}

func singleCardPayment() omsDomain.SalesOrderPayments {
	return omsDomain.SalesOrderPayments{{Method: "Revolut", AmountMinorUnits: paidOrderTotal, TransactionID: "rev-1"}}
}

func TestReceivePOSSalesOrder_rejectsPaymentsThatDoNotAddUpToTheTotal(t *testing.T) {
	f := newPaymentsFixture(t, &stubSalesOrderRepo{}, &stubSalesOrderPaymentRepo{})
	payments := omsDomain.SalesOrderPayments{{Method: "Cash", AmountMinorUnits: 500}, {Method: "Revolut", AmountMinorUnits: 1000}}

	_, err := f.uc.Execute(context.Background(), paidOrderInput(payments))

	assert.ErrorIs(t, err, omsDomain.ErrPaymentsTotalMismatch)
}

func TestReceivePOSSalesOrder_storesEveryPaymentPartInOrder(t *testing.T) {
	f := newPaymentsFixtureWithFreshOrder(t)

	_, err := f.uc.Execute(context.Background(), paidOrderInput(cashPlusCard()))

	require.NoError(t, err)
	assert.Equal(t, []omsDomain.SalesOrderPayments{cashPlusCard()}, f.paymentRepo.created)
}

func TestReceivePOSSalesOrder_returnsThePaymentsOnTheOrder(t *testing.T) {
	f := newPaymentsFixtureWithFreshOrder(t)

	order, err := f.uc.Execute(context.Background(), paidOrderInput(cashPlusCard()))

	require.NoError(t, err)
	assert.Equal(t, cashPlusCard(), order.Payments)
}

func TestReceivePOSSalesOrder_derivesSplitMethodFromSeveralPayments(t *testing.T) {
	f := newPaymentsFixtureWithFreshOrder(t)

	_, err := f.uc.Execute(context.Background(), paidOrderInput(cashPlusCard()))

	require.NoError(t, err)
	assert.Equal(t, "split", f.salesOrderRepo.created[0].PaymentMethod)
}

func TestReceivePOSSalesOrder_derivesMethodFromTheOnlyPayment(t *testing.T) {
	f := newPaymentsFixtureWithFreshOrder(t)

	_, err := f.uc.Execute(context.Background(), paidOrderInput(singleCardPayment()))

	require.NoError(t, err)
	assert.Equal(t, "Revolut", f.salesOrderRepo.created[0].PaymentMethod)
}

func TestReceivePOSSalesOrder_derivesTransactionIDFromTheOnlyPayment(t *testing.T) {
	f := newPaymentsFixtureWithFreshOrder(t)

	_, err := f.uc.Execute(context.Background(), paidOrderInput(singleCardPayment()))

	require.NoError(t, err)
	assert.Equal(t, "rev-1", f.salesOrderRepo.created[0].TransactionID)
}

func TestReceivePOSSalesOrder_keepsAnExplicitPaymentMethod(t *testing.T) {
	f := newPaymentsFixtureWithFreshOrder(t)
	in := paidOrderInput(cashPlusCard())
	in.PaymentMethod = "pos_split"

	_, err := f.uc.Execute(context.Background(), in)

	require.NoError(t, err)
	assert.Equal(t, "pos_split", f.salesOrderRepo.created[0].PaymentMethod)
}

func TestReceivePOSSalesOrder_keepsAnExplicitTransactionID(t *testing.T) {
	f := newPaymentsFixtureWithFreshOrder(t)
	in := paidOrderInput(singleCardPayment())
	in.TransactionID = "order-level-ref"

	_, err := f.uc.Execute(context.Background(), in)

	require.NoError(t, err)
	assert.Equal(t, "order-level-ref", f.salesOrderRepo.created[0].TransactionID)
}

func TestReceivePOSSalesOrder_withoutPaymentsKeepsThePaymentFieldsAsSent(t *testing.T) {
	f := newPaymentsFixtureWithFreshOrder(t)
	in := paidOrderInput(nil)
	in.PaymentMethod, in.TransactionID = "Cash", "till-7"

	_, err := f.uc.Execute(context.Background(), in)

	require.NoError(t, err)
	assert.Equal(t, []string{"Cash", "till-7"}, []string{f.salesOrderRepo.created[0].PaymentMethod, f.salesOrderRepo.created[0].TransactionID})
}

func TestReceivePOSSalesOrder_withoutPaymentsStoresNoPaymentParts(t *testing.T) {
	f := newPaymentsFixtureWithFreshOrder(t)

	_, err := f.uc.Execute(context.Background(), paidOrderInput(nil))

	require.NoError(t, err)
	assert.Empty(t, f.paymentRepo.created)
}

func TestReceivePOSSalesOrder_replayDoesNotStoreThePaymentsAgain(t *testing.T) {
	existing := &omsDomain.SalesOrder{ID: "sales-order-existing", ExternalOrderID: "pos-order-1"}
	f := newPaymentsFixture(t, &stubSalesOrderRepo{findByIDResult: existing}, &stubSalesOrderPaymentRepo{byOrder: cashPlusCard()})

	_, err := f.uc.Execute(context.Background(), paidOrderInput(cashPlusCard()))

	require.NoError(t, err)
	assert.Empty(t, f.paymentRepo.created)
}

func TestReceivePOSSalesOrder_replayReturnsTheStoredPayments(t *testing.T) {
	existing := &omsDomain.SalesOrder{ID: "sales-order-existing", ExternalOrderID: "pos-order-1"}
	f := newPaymentsFixture(t, &stubSalesOrderRepo{findByIDResult: existing}, &stubSalesOrderPaymentRepo{byOrder: cashPlusCard()})

	order, err := f.uc.Execute(context.Background(), paidOrderInput(cashPlusCard()))

	require.NoError(t, err)
	assert.Equal(t, cashPlusCard(), order.Payments)
}

func TestReceivePOSSalesOrder_propagatesAPaymentStoreError(t *testing.T) {
	f := newPaymentsFixture(t, &stubSalesOrderRepo{}, &stubSalesOrderPaymentRepo{createErr: assert.AnError})

	_, err := f.uc.Execute(context.Background(), paidOrderInput(cashPlusCard()))

	assert.ErrorIs(t, err, assert.AnError)
}
