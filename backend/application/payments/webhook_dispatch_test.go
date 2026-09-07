package payments_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	omsapp "github.com/adirkuhn/commerceos/application/oms"
	"github.com/adirkuhn/commerceos/application/payments"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/domain/webhooks"
)

type fakeParser struct {
	event PaymentEventOverride
	err   error
}

// PaymentEventOverride avoids importing payments.PaymentEvent twice under
// two names -- it's the exact same struct, aliased for readability at the
// test call sites below.
type PaymentEventOverride = payments.PaymentEvent

func (f *fakeParser) Parse(_ []byte) (payments.PaymentEvent, error) {
	return f.event, f.err
}

type fakePaymentsSalesOrderRepo struct {
	omsDomain.SalesOrderRepository
	byNumber map[string]*omsDomain.SalesOrder
}

func (f *fakePaymentsSalesOrderRepo) FindByNumber(_ context.Context, number string) (*omsDomain.SalesOrder, error) {
	return f.byNumber[number], nil
}

type fakeSalesOrderUpdater struct {
	calls []struct {
		id string
		in omsapp.UpdateSalesOrderInput
	}
	err error
}

func (f *fakeSalesOrderUpdater) Execute(_ context.Context, id string, in omsapp.UpdateSalesOrderInput) (*omsDomain.SalesOrder, error) {
	f.calls = append(f.calls, struct {
		id string
		in omsapp.UpdateSalesOrderInput
	}{id, in})
	if f.err != nil {
		return nil, f.err
	}
	return &omsDomain.SalesOrder{ID: id}, nil
}

func TestWebhookDispatchHandler_notHandled_isNoOpAndReturnsNil(t *testing.T) {
	parser := &fakeParser{event: payments.PaymentEvent{Handled: false}}
	updater := &fakeSalesOrderUpdater{}
	repo := &fakePaymentsSalesOrderRepo{byNumber: map[string]*omsDomain.SalesOrder{}}
	h := payments.NewWebhookDispatchHandler(newMockGormDB(t), repo, parser, updater, "stripe")

	err := h.Execute(context.Background(), webhooks.Envelope{TenantID: "t1", RawPayload: []byte(`{}`)})

	require.NoError(t, err)
	assert.Empty(t, updater.calls)
}

func TestWebhookDispatchHandler_orderNotFound_isNoOp(t *testing.T) {
	parser := &fakeParser{event: payments.PaymentEvent{Handled: true, OrderNumber: "WA-UNKNOWN", ExternalPaymentID: "ext-1"}}
	updater := &fakeSalesOrderUpdater{}
	repo := &fakePaymentsSalesOrderRepo{byNumber: map[string]*omsDomain.SalesOrder{}}
	h := payments.NewWebhookDispatchHandler(newMockGormDB(t), repo, parser, updater, "stripe")

	err := h.Execute(context.Background(), webhooks.Envelope{TenantID: "t1", RawPayload: []byte(`{}`)})

	require.NoError(t, err)
	assert.Empty(t, updater.calls)
}

func TestWebhookDispatchHandler_alreadyPaid_isIdempotentNoOp(t *testing.T) {
	paidAt := time.Now()
	parser := &fakeParser{event: payments.PaymentEvent{Handled: true, OrderNumber: "WA-PAID", ExternalPaymentID: "ext-1"}}
	updater := &fakeSalesOrderUpdater{}
	repo := &fakePaymentsSalesOrderRepo{byNumber: map[string]*omsDomain.SalesOrder{
		"WA-PAID": {ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing, DatePaid: &paidAt},
	}}
	h := payments.NewWebhookDispatchHandler(newMockGormDB(t), repo, parser, updater, "stripe")

	err := h.Execute(context.Background(), webhooks.Envelope{TenantID: "t1", RawPayload: []byte(`{}`)})

	require.NoError(t, err)
	assert.Empty(t, updater.calls, "a redelivered webhook for an already-paid order must not update it again")
}

func TestWebhookDispatchHandler_pendingOrder_transitionsToProcessing(t *testing.T) {
	parser := &fakeParser{event: payments.PaymentEvent{Handled: true, OrderNumber: "WA-PENDING", ExternalPaymentID: "ext-1"}}
	updater := &fakeSalesOrderUpdater{}
	repo := &fakePaymentsSalesOrderRepo{byNumber: map[string]*omsDomain.SalesOrder{
		"WA-PENDING": {ID: "order-1", Status: omsDomain.SalesOrderStatusPending},
	}}
	h := payments.NewWebhookDispatchHandler(newMockGormDB(t), repo, parser, updater, "stripe")

	err := h.Execute(context.Background(), webhooks.Envelope{TenantID: "t1", RawPayload: []byte(`{}`)})

	require.NoError(t, err)
	require.Len(t, updater.calls, 1)
	assert.Equal(t, "order-1", updater.calls[0].id)
	require.NotNil(t, updater.calls[0].in.Status)
	assert.Equal(t, omsDomain.SalesOrderStatusProcessing, *updater.calls[0].in.Status)
	require.NotNil(t, updater.calls[0].in.TransactionID)
	assert.Equal(t, "ext-1", *updater.calls[0].in.TransactionID)
	assert.NotNil(t, updater.calls[0].in.DatePaid)
}

func TestWebhookDispatchHandler_orderPastPending_neverRegressesStatus(t *testing.T) {
	parser := &fakeParser{event: payments.PaymentEvent{Handled: true, OrderNumber: "WA-SHIPPED", ExternalPaymentID: "ext-1"}}
	updater := &fakeSalesOrderUpdater{}
	repo := &fakePaymentsSalesOrderRepo{byNumber: map[string]*omsDomain.SalesOrder{
		"WA-SHIPPED": {ID: "order-1", Status: omsDomain.SalesOrderStatusShipped},
	}}
	h := payments.NewWebhookDispatchHandler(newMockGormDB(t), repo, parser, updater, "stripe")

	err := h.Execute(context.Background(), webhooks.Envelope{TenantID: "t1", RawPayload: []byte(`{}`)})

	require.NoError(t, err)
	require.Len(t, updater.calls, 1)
	assert.Nil(t, updater.calls[0].in.Status, "an order already past Pending must not have its status touched")
	require.NotNil(t, updater.calls[0].in.TransactionID)
	assert.Equal(t, "ext-1", *updater.calls[0].in.TransactionID)
}

func TestWebhookDispatchHandler_parseError_propagates(t *testing.T) {
	parser := &fakeParser{err: errors.New("boom")}
	updater := &fakeSalesOrderUpdater{}
	repo := &fakePaymentsSalesOrderRepo{byNumber: map[string]*omsDomain.SalesOrder{}}
	h := payments.NewWebhookDispatchHandler(newMockGormDB(t), repo, parser, updater, "stripe")

	err := h.Execute(context.Background(), webhooks.Envelope{TenantID: "t1", RawPayload: []byte(`{}`)})

	require.Error(t, err)
}

func TestWebhookDispatchHandler_updateError_propagates(t *testing.T) {
	parser := &fakeParser{event: payments.PaymentEvent{Handled: true, OrderNumber: "WA-PENDING", ExternalPaymentID: "ext-1"}}
	updater := &fakeSalesOrderUpdater{err: errors.New("db down")}
	repo := &fakePaymentsSalesOrderRepo{byNumber: map[string]*omsDomain.SalesOrder{
		"WA-PENDING": {ID: "order-1", Status: omsDomain.SalesOrderStatusPending},
	}}
	h := payments.NewWebhookDispatchHandler(newMockGormDB(t), repo, parser, updater, "stripe")

	err := h.Execute(context.Background(), webhooks.Envelope{TenantID: "t1", RawPayload: []byte(`{}`)})

	require.Error(t, err)
}
