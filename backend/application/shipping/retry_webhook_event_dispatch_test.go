package shipping_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	webhooksDomain "github.com/adirkuhn/commerceos/domain/webhooks"
)

func TestRetryWebhookEventDispatch_reprocessesAFailedEventAndReturnsItsUpdatedRow(t *testing.T) {
	eventRepo := &fakeWebhookEventRepoForBackfill{events: []webhooksDomain.WebhookEvent{
		{ID: "evt-row-1", Provider: "myparcel", Topic: "shipment_status_change", EventID: "evt-1",
			Payload: myParcelHookPayload(555, 2, "3SMYPA1"), Status: webhooksDomain.WebhookEventStatusFailed, DispatchError: "myparcel unreachable"},
	}}
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555, Status: 1, StatusMain: "registered"}
	shipmentRepo := &fakeShipmentRepo{
		findByMyParcelShipmentIDFn: func(context.Context, int64) (*shippingDomain.Shipment, error) { return shipment, nil },
		updateStatusFn:             func(context.Context, string, int, string, string) error { return nil },
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(context.Context, *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(context.Context, string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{getTrackTraceFn: func(context.Context, int64) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{StatusCode: 3, StatusMain: "shipped"}, nil
	}}
	var fulfilledOrderID string
	salesOrderRepo := &fakeSalesOrderRepo{updateFn: func(_ context.Context, id string, _ map[string]any) error {
		fulfilledOrderID = id
		return nil
	}}

	uc := appshipping.NewRetryWebhookEventDispatchUseCase(newMockGormDB(t, 10), eventRepo, shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, nil, nil)

	updated, err := uc.Execute(context.Background(), "evt-row-1")

	require.NoError(t, err)
	assert.Equal(t, webhooksDomain.WebhookEventStatusProcessed, updated.Status, "the retry succeeded, so the returned row must reflect the new outcome")
	assert.Empty(t, updated.DispatchError)
	assert.Equal(t, "order-1", fulfilledOrderID)
}

func TestRetryWebhookEventDispatch_stillFailingReturnsTheFailedRowNotAnError(t *testing.T) {
	eventRepo := &fakeWebhookEventRepoForBackfill{events: []webhooksDomain.WebhookEvent{
		{ID: "evt-row-1", Provider: "myparcel", Topic: "shipment_status_change", EventID: "evt-1",
			Payload: myParcelHookPayload(555, 2, "3SMYPA1"), Status: webhooksDomain.WebhookEventStatusFailed},
	}}
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555}
	shipmentRepo := &fakeShipmentRepo{findByMyParcelShipmentIDFn: func(context.Context, int64) (*shippingDomain.Shipment, error) { return shipment, nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(context.Context, string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{getTrackTraceFn: func(context.Context, int64) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{}, errors.New("myparcel still unreachable")
	}}

	uc := appshipping.NewRetryWebhookEventDispatchUseCase(newMockGormDB(t, 10), eventRepo, shipmentRepo, &fakeShipmentStatusEventRepo{},
		connRepo, &fakeSalesOrderRepo{}, &fakeMyParcelClientFactory{client: client}, testEncSecret, nil, nil)

	updated, err := uc.Execute(context.Background(), "evt-row-1")

	require.NoError(t, err, "a retry that fails again is a normal outcome, not a fault in the retry action itself")
	assert.Equal(t, webhooksDomain.WebhookEventStatusFailed, updated.Status)
	assert.Contains(t, updated.DispatchError, "myparcel still unreachable")
}

func TestRetryWebhookEventDispatch_unknownID_returnsNotFound(t *testing.T) {
	eventRepo := &fakeWebhookEventRepoForBackfill{}
	uc := appshipping.NewRetryWebhookEventDispatchUseCase(newMockGormDB(t, 1), eventRepo, &fakeShipmentRepo{}, &fakeShipmentStatusEventRepo{},
		&mocks.ConnectionRepo{}, &fakeSalesOrderRepo{}, &fakeMyParcelClientFactory{}, testEncSecret, nil, nil)

	_, err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
	assert.ErrorIs(t, err, webhooksDomain.ErrWebhookEventNotFound)
}

func TestRetryWebhookEventDispatch_nonDispatchableTopic_returnsTypedError(t *testing.T) {
	eventRepo := &fakeWebhookEventRepoForBackfill{events: []webhooksDomain.WebhookEvent{
		{ID: "evt-row-1", Provider: "woocommerce", Topic: "order.updated", EventID: "evt-1", Payload: []byte(`{}`), Status: webhooksDomain.WebhookEventStatusReceived},
	}}
	uc := appshipping.NewRetryWebhookEventDispatchUseCase(newMockGormDB(t, 1), eventRepo, &fakeShipmentRepo{}, &fakeShipmentStatusEventRepo{},
		&mocks.ConnectionRepo{}, &fakeSalesOrderRepo{}, &fakeMyParcelClientFactory{}, testEncSecret, nil, nil)

	_, err := uc.Execute(context.Background(), "evt-row-1")

	require.Error(t, err)
	assert.ErrorIs(t, err, webhooksDomain.ErrWebhookEventNotDispatchable)
	assert.Empty(t, eventRepo.markDispatchCalls, "a non-dispatchable event must never even be attempted")
}

func TestRetryWebhookEventDispatch_orderStatusChangeTopic_reprocessesAndMarksProcessed(t *testing.T) {
	eventRepo := &fakeWebhookEventRepoForBackfill{events: []webhooksDomain.WebhookEvent{
		{ID: "evt-row-1", Provider: "myparcel", Topic: "order_status_change", EventID: "evt-1",
			Payload: myParcelOrderHookPayload("mp-uuid-1", "processed"), Status: webhooksDomain.WebhookEventStatusFailed},
	}}
	fulfilment := &shippingDomain.MyParcelFulfilmentOrder{SalesOrderID: "order-1", MyParcelUUID: "mp-uuid-1"}
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{findByMyParcelUUIDFn: func(context.Context, string) (*shippingDomain.MyParcelFulfilmentOrder, error) { return fulfilment, nil }}
	order := &omsDomain.SalesOrder{ID: "order-1", Source: omsDomain.SalesOrderSourcePOS, Status: omsDomain.SalesOrderStatusPending}
	salesOrderRepo := &fakeSalesOrderRepo{findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil }}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewRetryWebhookEventDispatchUseCase(newMockGormDB(t, 10), eventRepo, &fakeShipmentRepo{}, &fakeShipmentStatusEventRepo{},
		&mocks.ConnectionRepo{}, salesOrderRepo, &fakeMyParcelClientFactory{}, testEncSecret, fulfilmentRepo, updater)

	updated, err := uc.Execute(context.Background(), "evt-row-1")

	require.NoError(t, err)
	assert.Equal(t, webhooksDomain.WebhookEventStatusProcessed, updated.Status)
	require.Len(t, updater.calls, 1)
	assert.Equal(t, omsDomain.SalesOrderStatusShipped, updater.calls[0].status)
}

func TestRetryWebhookEventDispatch_orderStatusChangeTopic_unrecognizedStatus_marksFailedNotAnError(t *testing.T) {
	eventRepo := &fakeWebhookEventRepoForBackfill{events: []webhooksDomain.WebhookEvent{
		{ID: "evt-row-1", Provider: "myparcel", Topic: "order_status_change", EventID: "evt-1",
			Payload: myParcelOrderHookPayload("mp-uuid-1", "invoiced"), Status: webhooksDomain.WebhookEventStatusFailed},
	}}
	fulfilment := &shippingDomain.MyParcelFulfilmentOrder{SalesOrderID: "order-1", MyParcelUUID: "mp-uuid-1"}
	fulfilmentRepo := &fakeMyParcelFulfilmentOrderRepo{findByMyParcelUUIDFn: func(context.Context, string) (*shippingDomain.MyParcelFulfilmentOrder, error) { return fulfilment, nil }}

	uc := appshipping.NewRetryWebhookEventDispatchUseCase(newMockGormDB(t, 10), eventRepo, &fakeShipmentRepo{}, &fakeShipmentStatusEventRepo{},
		&mocks.ConnectionRepo{}, &fakeSalesOrderRepo{}, &fakeMyParcelClientFactory{}, testEncSecret, fulfilmentRepo, &fakeSalesOrderStatusUpdater{})

	updated, err := uc.Execute(context.Background(), "evt-row-1")

	require.NoError(t, err, "a retry that fails again is a normal outcome, not a fault in the retry action itself")
	assert.Equal(t, webhooksDomain.WebhookEventStatusFailed, updated.Status)
	assert.Contains(t, updated.DispatchError, "unrecognized myparcel order status")
}

func TestRetryWebhookEventDispatch_orderStatusChangeTopic_noFulfilmentOrderRepoWired_returnsNotDispatchable(t *testing.T) {
	eventRepo := &fakeWebhookEventRepoForBackfill{events: []webhooksDomain.WebhookEvent{
		{ID: "evt-row-1", Provider: "myparcel", Topic: "order_status_change", EventID: "evt-1", Payload: []byte(`{}`), Status: webhooksDomain.WebhookEventStatusReceived},
	}}
	uc := appshipping.NewRetryWebhookEventDispatchUseCase(newMockGormDB(t, 1), eventRepo, &fakeShipmentRepo{}, &fakeShipmentStatusEventRepo{},
		&mocks.ConnectionRepo{}, &fakeSalesOrderRepo{}, &fakeMyParcelClientFactory{}, testEncSecret, nil, nil)

	_, err := uc.Execute(context.Background(), "evt-row-1")

	require.Error(t, err)
	assert.ErrorIs(t, err, webhooksDomain.ErrWebhookEventNotDispatchable)
}
