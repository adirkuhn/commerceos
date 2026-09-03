package shipping_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func TestReconcileShipments_updatesChangedStatusAndSkipsUnchanged(t *testing.T) {
	stale := []shippingDomain.Shipment{
		{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 1, Status: 1, StatusMain: "registered"},
		{ID: "s-2", ConnectionID: "conn-1", SalesOrderID: "order-2", MyParcelShipmentID: 2, Status: 1, StatusMain: "registered"},
		{ID: "s-3", ConnectionID: "conn-other", SalesOrderID: "order-3", MyParcelShipmentID: 3, Status: 1, StatusMain: "registered"},
	}
	shipmentRepo := &fakeShipmentRepo{
		findStaleNonTerminalFn: func(_ context.Context, _ time.Time, _ int) ([]shippingDomain.Shipment, error) { return stale, nil },
		updateStatusFn:         func(_ context.Context, _ string, _ int, _ string, _ string, _ string) error { return nil },
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(_ context.Context, _ *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}

	client := &fakeMyParcelClient{getTrackTraceFn: func(_ context.Context, id int64) (integrationsDomain.TrackTraceResult, error) {
		if id == 1 {
			return integrationsDomain.TrackTraceResult{StatusCode: 2, StatusMain: "shipped"}, nil // changed
		}
		return integrationsDomain.TrackTraceResult{StatusCode: 1, StatusMain: "registered"}, nil // unchanged
	}}
	salesOrderRepo := &fakeSalesOrderRepo{updateFn: func(context.Context, string, map[string]any) error { return nil }}

	uc := appshipping.NewReconcileShipmentsUseCase(newMockGormDB(t, 2), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, nil)

	result, err := uc.Execute(context.Background(), "conn-1")

	require.NoError(t, err)
	assert.Equal(t, 2, result.Checked, "only conn-1's 2 shipments are checked, conn-other's is filtered out")
	assert.Equal(t, 1, result.Updated, "only the actually-changed shipment counts as updated")
	assert.Empty(t, result.Errors)
}

func TestReconcileShipments_statusUnchangedButBarcodeMissing_backfillsBarcodeOnly(t *testing.T) {
	stale := []shippingDomain.Shipment{
		{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 1, Status: 4, StatusMain: "enroute", Barcode: ""},
	}
	var updateCalls int
	var savedBarcode string
	shipmentRepo := &fakeShipmentRepo{
		findStaleNonTerminalFn: func(_ context.Context, _ time.Time, _ int) ([]shippingDomain.Shipment, error) { return stale, nil },
		updateStatusFn: func(_ context.Context, _ string, _ int, _, _, barcode string) error {
			updateCalls++
			savedBarcode = barcode
			return nil
		},
	}
	appendCalled := false
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(_ context.Context, _ *shippingDomain.ShipmentStatusEvent) error {
		appendCalled = true
		return nil
	}}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{
		getTrackTraceFn: func(_ context.Context, _ int64) (integrationsDomain.TrackTraceResult, error) {
			return integrationsDomain.TrackTraceResult{StatusCode: 4, StatusMain: "enroute"}, nil // same as recorded
		},
		getShipmentBarcodeFn: func(_ context.Context, id int64) (string, error) {
			require.Equal(t, int64(1), id)
			return "3SMYPA447656020", nil
		},
	}
	salesOrderRepo := &fakeSalesOrderRepo{updateFn: func(context.Context, string, map[string]any) error { return nil }}

	uc := appshipping.NewReconcileShipmentsUseCase(newMockGormDB(t, 2), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, nil)

	result, err := uc.Execute(context.Background(), "conn-1")

	require.NoError(t, err)
	assert.Equal(t, 1, updateCalls)
	assert.Equal(t, "3SMYPA447656020", savedBarcode)
	assert.False(t, appendCalled, "status genuinely didn't change -- no duplicate timeline entry")
	assert.Empty(t, result.Errors)
}

func TestReconcileShipments_perShipmentFailure_doesNotAbortTheBatch(t *testing.T) {
	stale := []shippingDomain.Shipment{
		{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 1, StatusMain: "registered"},
		{ID: "s-2", ConnectionID: "conn-1", SalesOrderID: "order-2", MyParcelShipmentID: 2, StatusMain: "registered"},
	}
	shipmentRepo := &fakeShipmentRepo{
		findStaleNonTerminalFn: func(_ context.Context, _ time.Time, _ int) ([]shippingDomain.Shipment, error) { return stale, nil },
		updateStatusFn:         func(_ context.Context, _ string, _ int, _ string, _ string, _ string) error { return nil },
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(_ context.Context, _ *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}

	client := &fakeMyParcelClient{getTrackTraceFn: func(_ context.Context, id int64) (integrationsDomain.TrackTraceResult, error) {
		if id == 1 {
			return integrationsDomain.TrackTraceResult{}, assert.AnError
		}
		return integrationsDomain.TrackTraceResult{StatusCode: 2, StatusMain: "shipped"}, nil
	}}
	salesOrderRepo := &fakeSalesOrderRepo{updateFn: func(context.Context, string, map[string]any) error { return nil }}

	uc := appshipping.NewReconcileShipmentsUseCase(newMockGormDB(t, 2), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, nil)

	result, err := uc.Execute(context.Background(), "conn-1")

	require.NoError(t, err, "a per-shipment failure must not fail the whole run")
	assert.Equal(t, 2, result.Checked)
	assert.Equal(t, 1, result.Updated, "the second shipment still gets processed despite the first one failing")
	require.Len(t, result.Errors, 1)
}

func TestReconcileShipments_fulfilledOrderStillProcessing_movesToShipped(t *testing.T) {
	// Reproduces the real bug: a reconciliation poll (not the live webhook)
	// catches a status change for an already-Fulfilled order still stuck on
	// Processing -- this path used to never call MarkOrderShippedOnFulfillment
	// at all, so the order stayed Processing forever even after the webhook
	// path was fixed to no longer depend on a status.main == "shipped" match.
	stale := []shippingDomain.Shipment{
		{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 1, Status: 1, StatusMain: "registered"},
	}
	shipmentRepo := &fakeShipmentRepo{
		findStaleNonTerminalFn: func(_ context.Context, _ time.Time, _ int) ([]shippingDomain.Shipment, error) { return stale, nil },
		updateStatusFn:         func(_ context.Context, _ string, _ int, _ string, _ string, _ string) error { return nil },
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(_ context.Context, _ *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{getTrackTraceFn: func(_ context.Context, _ int64) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{StatusCode: 3, StatusMain: "handed_to_carrier"}, nil // the real string MyParcel sends -- never "shipped"
	}}
	order := &omsDomain.SalesOrder{ID: "order-1", Source: omsDomain.SalesOrderSourcePOS, Status: omsDomain.SalesOrderStatusProcessing, Fulfilled: true}
	salesOrderRepo := &fakeSalesOrderRepo{
		updateFn:   func(context.Context, string, map[string]any) error { return nil },
		findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil },
	}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewReconcileShipmentsUseCase(newMockGormDB(t, 3), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, updater)

	result, err := uc.Execute(context.Background(), "conn-1")

	require.NoError(t, err)
	assert.Equal(t, 1, result.Updated)
	require.Len(t, updater.calls, 1)
	assert.Equal(t, "order-1", updater.calls[0].id)
	assert.Equal(t, omsDomain.SalesOrderStatusShipped, updater.calls[0].status)
}

func TestReconcileShipments_notYetFulfilled_doesNotMoveOrder(t *testing.T) {
	stale := []shippingDomain.Shipment{
		{ID: "s-1", ConnectionID: "conn-1", SalesOrderID: "order-1", MyParcelShipmentID: 1, Status: 1, StatusMain: "registered"},
	}
	shipmentRepo := &fakeShipmentRepo{
		findStaleNonTerminalFn: func(_ context.Context, _ time.Time, _ int) ([]shippingDomain.Shipment, error) { return stale, nil },
		updateStatusFn:         func(_ context.Context, _ string, _ int, _ string, _ string, _ string) error { return nil },
	}
	statusEventRepo := &fakeShipmentStatusEventRepo{appendFn: func(_ context.Context, _ *shippingDomain.ShipmentStatusEvent) error { return nil }}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	client := &fakeMyParcelClient{getTrackTraceFn: func(_ context.Context, _ int64) (integrationsDomain.TrackTraceResult, error) {
		return integrationsDomain.TrackTraceResult{StatusCode: 3, StatusMain: "handed_to_carrier"}, nil
	}}
	order := &omsDomain.SalesOrder{ID: "order-1", Source: omsDomain.SalesOrderSourcePOS, Status: omsDomain.SalesOrderStatusProcessing, Fulfilled: false}
	salesOrderRepo := &fakeSalesOrderRepo{
		updateFn:   func(context.Context, string, map[string]any) error { return nil },
		findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil },
	}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewReconcileShipmentsUseCase(newMockGormDB(t, 3), shipmentRepo, statusEventRepo, connRepo, salesOrderRepo,
		&fakeMyParcelClientFactory{client: client}, testEncSecret, updater)

	result, err := uc.Execute(context.Background(), "conn-1")

	require.NoError(t, err)
	assert.Equal(t, 1, result.Updated, "the shipment itself still updates -- only the order transition is withheld")
	assert.Empty(t, updater.calls, "not Fulfilled yet -- must not move Status")
}

func TestReconcileShipments_noStaleShipments_returnsEmptyResultWithoutBuildingClient(t *testing.T) {
	shipmentRepo := &fakeShipmentRepo{
		findStaleNonTerminalFn: func(_ context.Context, _ time.Time, _ int) ([]shippingDomain.Shipment, error) { return nil, nil },
	}
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	factory := &fakeMyParcelClientFactory{newClientErr: assert.AnError} // must never be called

	uc := appshipping.NewReconcileShipmentsUseCase(newMockGormDB(t, 1), shipmentRepo, &fakeShipmentStatusEventRepo{}, connRepo, &fakeSalesOrderRepo{},
		factory, testEncSecret, nil)

	result, err := uc.Execute(context.Background(), "conn-1")

	require.NoError(t, err)
	assert.Equal(t, 0, result.Checked)
}
