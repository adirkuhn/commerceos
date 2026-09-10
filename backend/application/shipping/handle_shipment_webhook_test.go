package shipping_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/domain/webhooks"
)

func myParcelHookPayload(shipmentID int64, status int, barcode string) []byte {
	return []byte(`{"data":{"hooks":[{"shipment_id":` + strconv.FormatInt(shipmentID, 10) +
		`,"status":` + strconv.Itoa(status) + `,"barcode":"` + barcode + `"}]}}`)
}

func TestHandleShipmentWebhook_recordsRealStatusMainVerbatim(t *testing.T) {
	// "handed_to_carrier" is the real status.main MyParcel sends for this
	// transition (confirmed against a live account 2026-09-03) -- not
	// "shipped", which the codebase used to hardcode and never actually
	// matched. applyStatusUpdate must still record whatever string the API
	// sends, unchanged.
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", ExternalShipmentID: "555"}
	shipmentRepo := &fakeShipmentRepo{
		findByExternalShipmentIDFn: func(_ context.Context, id string) (*shippingDomain.Shipment, error) {
			require.Equal(t, "555", id)
			return shipment, nil
		},
		updateStatusFn: func(_ context.Context, id string, status int, statusMain, _, _ string) error {
			assert.Equal(t, "s-1", id)
			assert.Equal(t, "handed_to_carrier", statusMain)
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
	client := &fakeCarrierClient{getTrackTraceFn: func(_ context.Context, id string) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{StatusCode: 3, StatusMain: "handed_to_carrier"}, nil
	}}
	salesOrderRepo := &fakeSalesOrderRepo{updateFn: func(_ context.Context, _ string, _ map[string]any) error { return nil }}

	uc := appshipping.NewHandleShipmentWebhookUseCase(newMockGormDB(t, 2), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &stubOutboxRepo{})

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelHookPayload(555, 3, "3SMYPA1")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	require.NotNil(t, appended)
	assert.Equal(t, shippingDomain.StatusEventSourceWebhook, appended.Source)
	assert.Equal(t, "handed_to_carrier", appended.StatusMain)
}

func TestHandleShipmentWebhook_savesBarcodeFromWebhookPayload(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", ExternalShipmentID: "555", Barcode: ""}
	var savedBarcode string
	shipmentRepo := &fakeShipmentRepo{
		findByExternalShipmentIDFn: func(_ context.Context, _ string) (*shippingDomain.Shipment, error) { return shipment, nil },
		updateStatusFn: func(_ context.Context, _ string, _ int, _ string, _ string, barcode string) error {
			savedBarcode = barcode
			return nil
		},
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(_ context.Context, _ *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeCarrierClient{getTrackTraceFn: func(_ context.Context, _ string) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{StatusCode: 4, StatusMain: "enroute"}, nil
	}}
	salesOrderRepo := &fakeSalesOrderRepo{updateFn: func(context.Context, string, map[string]any) error { return nil }}

	uc := appshipping.NewHandleShipmentWebhookUseCase(newMockGormDB(t, 2), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &stubOutboxRepo{})

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelHookPayload(555, 4, "3SMYPA447656020")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	assert.Equal(t, "3SMYPA447656020", savedBarcode, "the webhook payload's own barcode must reach the shipment write, not be discarded")
}

func TestHandleShipmentWebhook_enqueuesFulfillmentSignal(t *testing.T) {
	// The bug this reproduces: nothing ever told the SalesOrder side that a
	// shipment webhook arrived, other than a direct in-process call this
	// use case used to make itself. Now it just reports what MyParcel said
	// -- ApplyShipmentFulfillmentSignalUseCase (see its own tests) is the
	// one place that turns this into an actual status transition.
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", ExternalShipmentID: "555"}
	shipmentRepo := &fakeShipmentRepo{
		findByExternalShipmentIDFn: func(_ context.Context, _ string) (*shippingDomain.Shipment, error) { return shipment, nil },
		updateStatusFn:             func(_ context.Context, _ string, _ int, _ string, _ string, _ string) error { return nil },
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(_ context.Context, _ *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeCarrierClient{getTrackTraceFn: func(_ context.Context, _ string) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{StatusCode: 3, StatusMain: "handed_to_carrier"}, nil
	}}
	salesOrderRepo := &fakeSalesOrderRepo{updateFn: func(context.Context, string, map[string]any) error { return nil }}
	outboxRepo := &stubOutboxRepo{}

	uc := appshipping.NewHandleShipmentWebhookUseCase(newMockGormDB(t, 2), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, outboxRepo)

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelHookPayload(555, 3, "3SMYPA1")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	require.Len(t, outboxRepo.calls, 1)
	assert.Equal(t, events.ShipmentFulfillmentSignal{}.EventName(), outboxRepo.calls[0].subject)
	var event events.ShipmentFulfillmentSignal
	require.NoError(t, json.Unmarshal(outboxRepo.calls[0].payload, &event))
	assert.Equal(t, "order-1", event.SalesOrderID)
	assert.Equal(t, events.ShipmentFulfillmentSignalKindShipmentStatus, event.Kind)
	assert.Equal(t, "handed_to_carrier", event.StatusMain)
	assert.Equal(t, 3, event.StatusCode)
}

func TestHandleShipmentWebhook_unknownShipmentID_isSkippedNotAnError(t *testing.T) {
	shipmentRepo := &fakeShipmentRepo{
		findByExternalShipmentIDFn: func(_ context.Context, _ string) (*shippingDomain.Shipment, error) {
			return nil, shippingDomain.ErrShipmentNotFound
		},
	}
	uc := appshipping.NewHandleShipmentWebhookUseCase(newMockGormDB(t, 1), shipmentRepo, &fakeShipmentStatusEventRepo{},
		&mocks.ConnectionRepo{}, &fakeSalesOrderRepo{}, &fakeCarrierClientFactory{}, testEncSecret, &stubOutboxRepo{})

	envelope := webhooks.Envelope{TenantID: "tenant-1", Provider: "myparcel", RawPayload: myParcelHookPayload(999, 2, "x")}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err, "a webhook for a shipment_id this tenant doesn't know about must not fail/retry forever")
}
