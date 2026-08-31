package shipping_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/domain/webhooks"
)

func myParcelHookPayload(shipmentID int64, status int, barcode string) []byte {
	return []byte(`{"data":{"hooks":[{"shipment_id":` + strconv.FormatInt(shipmentID, 10) +
		`,"status":` + strconv.Itoa(status) + `,"barcode":"` + barcode + `"}]}}`)
}

func TestHandleShipmentWebhook_shippedStatus_marksOrderFulfilled(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555}
	shipmentRepo := &fakeShipmentRepo{
		findByMyParcelShipmentIDFn: func(_ context.Context, id int64) (*shippingDomain.Shipment, error) {
			require.Equal(t, int64(555), id)
			return shipment, nil
		},
		updateStatusFn: func(_ context.Context, id string, status int, statusMain, _ string) error {
			assert.Equal(t, "s-1", id)
			assert.Equal(t, "shipped", statusMain)
			return nil
		},
	}
	var appended *shippingDomain.ShipmentStatusEvent
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(_ context.Context, e *shippingDomain.ShipmentStatusEvent) error {
		appended = e
		return nil
	}}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{getTrackTraceFn: func(_ context.Context, id int64) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{StatusCode: 2, StatusMain: "shipped"}, nil
	}}
	var fulfilledOrderID string
	var fulfilledFields map[string]any
	salesOrderRepo := &fakeSalesOrderRepo{updateFn: func(_ context.Context, id string, fields map[string]any) error {
		fulfilledOrderID, fulfilledFields = id, fields
		return nil
	}}

	uc := appshipping.NewHandleShipmentWebhookUseCase(newMockGormDB(t, 2), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, nil)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelHookPayload(555, 2, "3SMYPA1")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	require.NotNil(t, appended)
	assert.Equal(t, shippingDomain.StatusEventSourceWebhook, appended.Source)
	assert.Equal(t, "order-1", fulfilledOrderID)
	assert.Equal(t, true, fulfilledFields["fulfilled"])
}

func TestHandleShipmentWebhook_shippedStatus_movesOrderToShipped(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555}
	shipmentRepo := &fakeShipmentRepo{
		findByMyParcelShipmentIDFn: func(_ context.Context, _ int64) (*shippingDomain.Shipment, error) { return shipment, nil },
		updateStatusFn:             func(_ context.Context, _ string, _ int, _ string, _ string) error { return nil },
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(_ context.Context, _ *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{getTrackTraceFn: func(_ context.Context, _ int64) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{StatusCode: 2, StatusMain: "shipped"}, nil
	}}
	order := &omsDomain.SalesOrder{ID: "order-1", Source: omsDomain.SalesOrderSourcePOS, Status: omsDomain.SalesOrderStatusProcessing}
	salesOrderRepo := &fakeSalesOrderRepo{
		updateFn:   func(context.Context, string, map[string]any) error { return nil },
		findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil },
	}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewHandleShipmentWebhookUseCase(newMockGormDB(t, 3), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, updater)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelHookPayload(555, 2, "3SMYPA1")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	require.Len(t, updater.calls, 1)
	assert.Equal(t, "order-1", updater.calls[0].id)
	assert.Equal(t, omsDomain.SalesOrderStatusShipped, updater.calls[0].status)
}

func TestHandleShipmentWebhook_shippedStatus_wooOrderAlsoMovesToShipped(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555}
	shipmentRepo := &fakeShipmentRepo{
		findByMyParcelShipmentIDFn: func(_ context.Context, _ int64) (*shippingDomain.Shipment, error) { return shipment, nil },
		updateStatusFn:             func(_ context.Context, _ string, _ int, _ string, _ string) error { return nil },
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(_ context.Context, _ *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{getTrackTraceFn: func(_ context.Context, _ int64) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{StatusCode: 2, StatusMain: "shipped"}, nil
	}}
	order := &omsDomain.SalesOrder{ID: "order-1", Source: omsDomain.SalesOrderSourceWooCommerce, Status: omsDomain.SalesOrderStatusProcessing}
	salesOrderRepo := &fakeSalesOrderRepo{
		updateFn:   func(context.Context, string, map[string]any) error { return nil },
		findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil },
	}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewHandleShipmentWebhookUseCase(newMockGormDB(t, 3), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, updater)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelHookPayload(555, 2, "3SMYPA1")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	require.Len(t, updater.calls, 1)
	assert.Equal(t, omsDomain.SalesOrderStatusShipped, updater.calls[0].status, "internal status is uniformly Shipped -- Woo's lack of a native shipped status is handled at the channel-push boundary, not here")
}

func TestHandleShipmentWebhook_shippedStatus_alreadyPastProcessing_doesNotReapply(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555}
	shipmentRepo := &fakeShipmentRepo{
		findByMyParcelShipmentIDFn: func(_ context.Context, _ int64) (*shippingDomain.Shipment, error) { return shipment, nil },
		updateStatusFn:             func(_ context.Context, _ string, _ int, _ string, _ string) error { return nil },
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(_ context.Context, _ *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{getTrackTraceFn: func(_ context.Context, _ int64) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{StatusCode: 2, StatusMain: "shipped"}, nil
	}}
	// Already OnHold via some other path (e.g. staff manually placed it on
	// hold) -- a shipped webhook arriving after that must not move it.
	order := &omsDomain.SalesOrder{ID: "order-1", Source: omsDomain.SalesOrderSourcePOS, Status: omsDomain.SalesOrderStatusOnHold}
	salesOrderRepo := &fakeSalesOrderRepo{
		updateFn:   func(context.Context, string, map[string]any) error { return nil },
		findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil },
	}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewHandleShipmentWebhookUseCase(newMockGormDB(t, 3), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, updater)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelHookPayload(555, 2, "3SMYPA1")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	assert.Empty(t, updater.calls, "already past Pending/Processing -- must not move status")
}

func TestHandleShipmentWebhook_deliveredStatus_completesOrder(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555}
	shipmentRepo := &fakeShipmentRepo{
		findByMyParcelShipmentIDFn: func(_ context.Context, id int64) (*shippingDomain.Shipment, error) { return shipment, nil },
		updateStatusFn:             func(_ context.Context, _ string, _ int, _ string, _ string) error { return nil },
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(_ context.Context, _ *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{getTrackTraceFn: func(_ context.Context, _ int64) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{StatusCode: 7, StatusMain: "delivered"}, nil
	}}
	salesOrderRepo := &fakeSalesOrderRepo{updateFn: func(context.Context, string, map[string]any) error { return nil }}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewHandleShipmentWebhookUseCase(newMockGormDB(t, 2), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, updater)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelHookPayload(555, 7, "3SMYPA1")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	require.Len(t, updater.calls, 1)
	assert.Equal(t, "order-1", updater.calls[0].id)
	assert.Equal(t, omsDomain.SalesOrderStatusCompleted, updater.calls[0].status)
}

func TestHandleShipmentWebhook_nonShippedStatus_doesNotTouchOrder(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555}
	shipmentRepo := &fakeShipmentRepo{
		findByMyParcelShipmentIDFn: func(_ context.Context, _ int64) (*shippingDomain.Shipment, error) { return shipment, nil },
		updateStatusFn:             func(_ context.Context, _ string, _ int, _ string, _ string) error { return nil },
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(_ context.Context, _ *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{getTrackTraceFn: func(_ context.Context, _ int64) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{StatusCode: 1, StatusMain: "registered"}, nil
	}}
	orderTouched := false
	salesOrderRepo := &fakeSalesOrderRepo{updateFn: func(context.Context, string, map[string]any) error {
		orderTouched = true
		return nil
	}}

	uc := appshipping.NewHandleShipmentWebhookUseCase(newMockGormDB(t, 2), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, nil)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelHookPayload(555, 1, "3SMYPA1")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	assert.False(t, orderTouched, "only the 'shipped' transition touches the order in v1")
}

func TestHandleShipmentWebhook_unknownShipmentID_isSkippedNotAnError(t *testing.T) {
	shipmentRepo := &fakeShipmentRepo{
		findByMyParcelShipmentIDFn: func(_ context.Context, _ int64) (*shippingDomain.Shipment, error) {
			return nil, shippingDomain.ErrShipmentNotFound
		},
	}
	uc := appshipping.NewHandleShipmentWebhookUseCase(newMockGormDB(t, 1), shipmentRepo, &fakeShipmentStatusEventRepo{},
		&mocks.ConnectionRepo{}, &fakeSalesOrderRepo{}, &fakeMyParcelClientFactory{}, testEncSecret, nil)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelHookPayload(999, 2, "x")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err, "a webhook for a shipment_id this tenant doesn't know about must not fail/retry forever")
}
