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

func myParcelHookEvent(id, eventID string, shipmentID int64, status int) webhooksDomain.WebhookEvent {
	return webhooksDomain.WebhookEvent{
		ID: id, Provider: "myparcel", Topic: "shipment_status_change", EventID: eventID,
		Payload: myParcelHookPayload(shipmentID, status, "3SMYPA1"), Status: webhooksDomain.WebhookEventStatusReceived,
	}
}

func TestBackfillShipmentWebhookDispatch_appliesChangedStatusAndLinksToOrder(t *testing.T) {
	eventRepo := &fakeWebhookEventRepoForBackfill{events: []webhooksDomain.WebhookEvent{
		myParcelHookEvent("evt-row-1", "evt-1", 555, 2),
	}}
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555, Status: 1, StatusMain: "registered"}
	shipmentRepo := &fakeShipmentRepo{
		findByMyParcelShipmentIDFn: func(_ context.Context, id int64) (*shippingDomain.Shipment, error) { return shipment, nil },
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

	uc := appshipping.NewBackfillShipmentWebhookDispatchUseCase(newMockGormDB(t, 10), eventRepo, shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, nil)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, result.Checked)
	assert.Equal(t, 1, result.Applied)
	assert.Equal(t, "order-1", fulfilledOrderID, "the backfilled shipped transition must still link to and fulfill its order")
	require.Len(t, eventRepo.markDispatchCalls, 1)
	assert.Equal(t, "evt-row-1", eventRepo.markDispatchCalls[0].id)
	assert.Equal(t, webhooksDomain.WebhookEventStatusProcessed, eventRepo.markDispatchCalls[0].status)
}

func TestBackfillShipmentWebhookDispatch_shippedStatus_movesOrderToShipped(t *testing.T) {
	eventRepo := &fakeWebhookEventRepoForBackfill{events: []webhooksDomain.WebhookEvent{
		myParcelHookEvent("evt-row-1", "evt-1", 555, 2),
	}}
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555, Status: 1, StatusMain: "registered"}
	shipmentRepo := &fakeShipmentRepo{
		findByMyParcelShipmentIDFn: func(_ context.Context, id int64) (*shippingDomain.Shipment, error) { return shipment, nil },
		updateStatusFn:             func(context.Context, string, int, string, string) error { return nil },
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(context.Context, *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(context.Context, string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{getTrackTraceFn: func(context.Context, int64) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{StatusCode: 3, StatusMain: "shipped"}, nil
	}}
	order := &omsDomain.SalesOrder{ID: "order-1", Source: omsDomain.SalesOrderSourceManual, Status: omsDomain.SalesOrderStatusPending}
	salesOrderRepo := &fakeSalesOrderRepo{
		updateFn:   func(context.Context, string, map[string]any) error { return nil },
		findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil },
	}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewBackfillShipmentWebhookDispatchUseCase(newMockGormDB(t, 10), eventRepo, shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, updater)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, result.Applied)
	require.Len(t, updater.calls, 1)
	assert.Equal(t, "order-1", updater.calls[0].id)
	assert.Equal(t, omsDomain.SalesOrderStatusShipped, updater.calls[0].status)
}

func TestBackfillShipmentWebhookDispatch_skipsWhenStatusAlreadyRecorded(t *testing.T) {
	eventRepo := &fakeWebhookEventRepoForBackfill{events: []webhooksDomain.WebhookEvent{
		myParcelHookEvent("evt-row-1", "evt-1", 555, 3),
	}}
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555, Status: 3, StatusMain: "shipped"}
	shipmentRepo := &fakeShipmentRepo{
		findByMyParcelShipmentIDFn: func(context.Context, int64) (*shippingDomain.Shipment, error) { return shipment, nil },
	}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(context.Context, string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{getTrackTraceFn: func(context.Context, int64) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{StatusCode: 3, StatusMain: "shipped"}, nil // already what's recorded
	}}
	appendCalled := false
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(context.Context, *shippingDomain.ShipmentStatusEvent) error {
		appendCalled = true
		return nil
	}}

	uc := appshipping.NewBackfillShipmentWebhookDispatchUseCase(newMockGormDB(t, 10), eventRepo, shipmentRepo, statusEventRepo, connRepo, &fakeSalesOrderRepo{},
		&fakeMyParcelClientFactory{client: client}, testEncSecret, nil)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, result.SkippedDuplicate)
	assert.Equal(t, 0, result.Applied)
	assert.False(t, appendCalled, "a status already matching the shipment's own record must not write a duplicate timeline entry")
	require.Len(t, eventRepo.markDispatchCalls, 1)
	assert.Equal(t, webhooksDomain.WebhookEventStatusProcessed, eventRepo.markDispatchCalls[0].status, "a duplicate is still a known, successfully-checked outcome")
}

func TestBackfillShipmentWebhookDispatch_multipleEventsSameShipment_fetchesTrackTraceOnce(t *testing.T) {
	eventRepo := &fakeWebhookEventRepoForBackfill{events: []webhooksDomain.WebhookEvent{
		myParcelHookEvent("evt-row-1", "evt-1", 555, 2),
		myParcelHookEvent("evt-row-2", "evt-2", 555, 3),
	}}
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555, Status: 1, StatusMain: "registered"}
	shipmentRepo := &fakeShipmentRepo{
		findByMyParcelShipmentIDFn: func(context.Context, int64) (*shippingDomain.Shipment, error) { return shipment, nil },
		updateStatusFn:             func(context.Context, string, int, string, string) error { return nil },
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(context.Context, *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(context.Context, string) (*integrationsDomain.Connection, error) { return conn, nil }}
	trackTraceCalls := 0
	client := &fakeMyParcelClient{getTrackTraceFn: func(context.Context, int64) (integrationsDomain.TrackTraceResult, error) {
		trackTraceCalls++
		return integrationsDomain.TrackTraceResult{StatusCode: 3, StatusMain: "shipped"}, nil
	}}
	salesOrderRepo := &fakeSalesOrderRepo{updateFn: func(context.Context, string, map[string]any) error { return nil }}

	uc := appshipping.NewBackfillShipmentWebhookDispatchUseCase(newMockGormDB(t, 10), eventRepo, shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, nil)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, result.Checked)
	assert.Equal(t, 1, trackTraceCalls, "two webhook_events rows for the same shipment must only fetch its live status once per run")
	assert.Equal(t, 1, result.Applied, "the first row's fetch applies the change")
	assert.Equal(t, 1, result.SkippedDuplicate, "the second row reuses the cached, already-applied outcome instead of appending again")
	require.Len(t, eventRepo.markDispatchCalls, 2, "each row is still marked individually")
}

func TestBackfillShipmentWebhookDispatch_unknownShipment_skipsAndMarksProcessed(t *testing.T) {
	eventRepo := &fakeWebhookEventRepoForBackfill{events: []webhooksDomain.WebhookEvent{
		myParcelHookEvent("evt-row-1", "evt-1", 999, 2),
	}}
	shipmentRepo := &fakeShipmentRepo{
		findByMyParcelShipmentIDFn: func(context.Context, int64) (*shippingDomain.Shipment, error) {
			return nil, shippingDomain.ErrShipmentNotFound
		},
	}
	uc := appshipping.NewBackfillShipmentWebhookDispatchUseCase(newMockGormDB(t, 10), eventRepo, shipmentRepo, &fakeShipmentStatusEventRepo{},
		&mocks.ConnectionRepo{}, &fakeSalesOrderRepo{}, &fakeMyParcelClientFactory{}, testEncSecret, nil)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, result.SkippedUnknownShipment)
	require.Len(t, eventRepo.markDispatchCalls, 1)
	assert.Equal(t, webhooksDomain.WebhookEventStatusProcessed, eventRepo.markDispatchCalls[0].status, "an unknown shipment is the same 'nothing to do' outcome live processing already treats as a non-error")
}

func TestBackfillShipmentWebhookDispatch_providerFetchFails_marksEventFailed(t *testing.T) {
	eventRepo := &fakeWebhookEventRepoForBackfill{events: []webhooksDomain.WebhookEvent{
		myParcelHookEvent("evt-row-1", "evt-1", 555, 2),
	}}
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555}
	shipmentRepo := &fakeShipmentRepo{findByMyParcelShipmentIDFn: func(context.Context, int64) (*shippingDomain.Shipment, error) { return shipment, nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(context.Context, string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{getTrackTraceFn: func(context.Context, int64) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{}, errors.New("myparcel unreachable")
	}}

	uc := appshipping.NewBackfillShipmentWebhookDispatchUseCase(newMockGormDB(t, 10), eventRepo, shipmentRepo, &fakeShipmentStatusEventRepo{},
		connRepo, &fakeSalesOrderRepo{}, &fakeMyParcelClientFactory{client: client}, testEncSecret, nil)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err, "a per-event failure must not fail the whole backfill run")
	assert.Equal(t, 1, result.Failed)
	require.Len(t, eventRepo.markDispatchCalls, 1)
	assert.Equal(t, webhooksDomain.WebhookEventStatusFailed, eventRepo.markDispatchCalls[0].status)
	assert.Contains(t, eventRepo.markDispatchCalls[0].dispatchError, "myparcel unreachable")
}

func TestBackfillShipmentWebhookDispatch_ignoresAlreadyProcessedAndOtherTopics(t *testing.T) {
	eventRepo := &fakeWebhookEventRepoForBackfill{events: []webhooksDomain.WebhookEvent{
		{ID: "evt-row-1", Provider: "myparcel", Topic: "shipment_status_change", EventID: "evt-1", Payload: []byte(`{}`), Status: webhooksDomain.WebhookEventStatusProcessed},
		{ID: "evt-row-2", Provider: "myparcel", Topic: "shop_updated", EventID: "evt-2", Payload: []byte(`{}`), Status: webhooksDomain.WebhookEventStatusReceived},
	}}
	uc := appshipping.NewBackfillShipmentWebhookDispatchUseCase(newMockGormDB(t, 10), eventRepo, &fakeShipmentRepo{}, &fakeShipmentStatusEventRepo{},
		&mocks.ConnectionRepo{}, &fakeSalesOrderRepo{}, &fakeMyParcelClientFactory{}, testEncSecret, nil)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 0, result.Checked, "already-processed rows and non-dispatchable topics are both left untouched")
	assert.Empty(t, eventRepo.markDispatchCalls)
}
