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
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func TestRefreshShipmentTracking_fetchesFromProviderAndSavesTheResult(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555}
	shipmentRepo := &fakeShipmentRepo{
		findByIDFn: func(_ context.Context, id string) (*shippingDomain.Shipment, error) {
			require.Equal(t, "s-1", id)
			return shipment, nil
		},
		updateStatusFn: func(_ context.Context, id string, status int, statusMain, _, _ string) error {
			assert.Equal(t, "s-1", id)
			assert.Equal(t, 5, status)
			assert.Equal(t, "enroute", statusMain)
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
		require.Equal(t, int64(555), id)
		return integrationsDomain.TrackTraceResult{StatusCode: 5, StatusMain: "enroute"}, nil
	}}
	salesOrderRepo := &fakeSalesOrderRepo{updateFn: func(context.Context, string, map[string]any) error { return nil }}

	uc := appshipping.NewRefreshShipmentTrackingUseCase(newMockGormDB(t, 2), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret)

	event, err := uc.Execute(context.Background(), "s-1")

	require.NoError(t, err)
	require.NotNil(t, appended, "the fetched status must be saved as a new tracking event, not just returned")
	assert.Equal(t, shippingDomain.StatusEventSourceManual, appended.Source)
	assert.Equal(t, 5, appended.Status)
	assert.Equal(t, "enroute", appended.StatusMain)
	assert.Equal(t, *appended, event)
}

func TestRefreshShipmentTracking_noBarcodeYet_fetchesAndSavesIt(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555, Barcode: ""}
	var savedBarcode string
	shipmentRepo := &fakeShipmentRepo{
		findByIDFn: func(_ context.Context, id string) (*shippingDomain.Shipment, error) { return shipment, nil },
		updateStatusFn: func(_ context.Context, _ string, _ int, _, _, barcode string) error {
			savedBarcode = barcode
			return nil
		},
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(_ context.Context, _ *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{
		getTrackTraceFn: func(_ context.Context, _ int64) (integrationsDomain.TrackTraceResult, error) {
			return integrationsDomain.TrackTraceResult{StatusCode: 4, StatusMain: "enroute"}, nil
		},
		getShipmentBarcodeFn: func(_ context.Context, id int64) (string, error) {
			require.Equal(t, int64(555), id)
			return "3SMYPA447656020", nil
		},
	}
	salesOrderRepo := &fakeSalesOrderRepo{updateFn: func(context.Context, string, map[string]any) error { return nil }}

	uc := appshipping.NewRefreshShipmentTrackingUseCase(newMockGormDB(t, 2), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret)

	_, err := uc.Execute(context.Background(), "s-1")

	require.NoError(t, err)
	assert.Equal(t, "3SMYPA447656020", savedBarcode, "GetTrackTrace never carries a barcode -- must fall back to GET /shipments/{id}")
}

func TestRefreshShipmentTracking_alreadyHasBarcode_skipsTheExtraCall(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555, Barcode: "3SMYPA1"}
	shipmentRepo := &fakeShipmentRepo{
		findByIDFn:     func(_ context.Context, id string) (*shippingDomain.Shipment, error) { return shipment, nil },
		updateStatusFn: func(context.Context, string, int, string, string, string) error { return nil },
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(_ context.Context, _ *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	barcodeCalled := false
	client := &fakeMyParcelClient{
		getTrackTraceFn: func(_ context.Context, _ int64) (integrationsDomain.TrackTraceResult, error) {
			return integrationsDomain.TrackTraceResult{StatusCode: 4, StatusMain: "enroute"}, nil
		},
		getShipmentBarcodeFn: func(_ context.Context, _ int64) (string, error) {
			barcodeCalled = true
			return "should-not-be-used", nil
		},
	}
	salesOrderRepo := &fakeSalesOrderRepo{updateFn: func(context.Context, string, map[string]any) error { return nil }}

	uc := appshipping.NewRefreshShipmentTrackingUseCase(newMockGormDB(t, 2), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret)

	_, err := uc.Execute(context.Background(), "s-1")

	require.NoError(t, err)
	assert.False(t, barcodeCalled, "already has a barcode -- no need for the extra GET /shipments/{id} call")
}

// Note: RefreshShipmentTrackingUseCase deliberately does not offer the
// SalesOrder to MarkOrderShippedOnFulfillment/completeOrderOnDelivery --
// unlike the webhook/backfill/reconcile paths, it has no
// SalesOrderStatusUpdater wired (see its struct -- no updateSalesOrder
// field), since this handler's dependency graph doesn't otherwise need
// application/oms.UpdateSalesOrderUseCase's stock-movement machinery. A
// staff-triggered "refresh tracking" click therefore only updates the
// Shipment/ShipmentStatusEvent rows; the order status catches up via the
// next webhook, the 24h ReconcileShipmentsUseCase poll, or
// ReconcileFulfilledOrdersUseCase's 24h sweep.

func TestRefreshShipmentTracking_shipmentNotFound_propagatesError(t *testing.T) {
	shipmentRepo := &fakeShipmentRepo{
		findByIDFn: func(context.Context, string) (*shippingDomain.Shipment, error) { return nil, shippingDomain.ErrShipmentNotFound },
	}
	uc := appshipping.NewRefreshShipmentTrackingUseCase(newMockGormDB(t, 1), shipmentRepo, &fakeShipmentStatusEventRepo{},
		&mocks.ConnectionRepo{}, &fakeSalesOrderRepo{}, &fakeMyParcelClientFactory{}, testEncSecret)

	_, err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
	assert.ErrorIs(t, err, shippingDomain.ErrShipmentNotFound)
}

func TestRefreshShipmentTracking_providerFetchFails_returnsError(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 555}
	shipmentRepo := &fakeShipmentRepo{findByIDFn: func(context.Context, string) (*shippingDomain.Shipment, error) { return shipment, nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(context.Context, string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{getTrackTraceFn: func(context.Context, int64) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{}, errors.New("myparcel unreachable")
	}}

	uc := appshipping.NewRefreshShipmentTrackingUseCase(newMockGormDB(t, 1), shipmentRepo, &fakeShipmentStatusEventRepo{},
		connRepo, &fakeSalesOrderRepo{}, &fakeMyParcelClientFactory{client: client}, testEncSecret)

	_, err := uc.Execute(context.Background(), "s-1")

	require.Error(t, err)
}
