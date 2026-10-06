package shipping_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestCancelShipment_myParcelDeletesSuccessfully_marksCancelledLocally(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}

	shipment := shippingDomain.Shipment{ID: "shipment-1", SalesOrderID: "order-1", ConnectionID: "conn-1", ExternalShipmentID: "555"}
	var gotDeleteID string
	client := &fakeCarrierClient{
		cancelShipmentFn: func(_ context.Context, id string) error {
			gotDeleteID = id
			return nil
		},
	}
	var markedID string
	var markedNote string
	shipmentRepo := &fakeShipmentRepo{
		findByIDFn: func(_ context.Context, id string) (*shippingDomain.Shipment, error) { return &shipment, nil },
		markCancelledFn: func(_ context.Context, id string, _ time.Time, note string) error {
			markedID, markedNote = id, note
			return nil
		},
	}

	outbox := &eventrecordtest.Outbox{}
	uc := appshipping.NewCancelShipmentUseCase(newMockGormDB(t, 2), connRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, outbox)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx, appshipping.CancelShipmentInput{ShipmentID: "shipment-1"})

	require.NoError(t, err)
	assert.Equal(t, "555", gotDeleteID)
	assert.True(t, result.DeletedOnMyParcel)
	assert.Contains(t, result.Message, "deleted on MyParcel")
	assert.Equal(t, "shipment-1", markedID)
	assert.Equal(t, result.Message, markedNote)

	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "shipping.shipment.cancelled", e.Subject)
	assert.Equal(t, "shipment-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"sales_order_id": "order-1", "deleted_on_carrier": true, "note": result.Message}, e.Data())
}

func TestCancelShipment_myParcelRejectsDelete_stillCancelsLocallyAndInformsCaller(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}

	shipment := shippingDomain.Shipment{ID: "shipment-1", SalesOrderID: "order-1", ConnectionID: "conn-1", ExternalShipmentID: "555"}
	client := &fakeCarrierClient{
		cancelShipmentFn: func(_ context.Context, _ string) error {
			return &myParcelAPIError{statusCode: 422, body: "Shipment cannot be deleted"}
		},
	}
	var markCancelledCalled bool
	shipmentRepo := &fakeShipmentRepo{
		findByIDFn: func(_ context.Context, id string) (*shippingDomain.Shipment, error) { return &shipment, nil },
		markCancelledFn: func(_ context.Context, _ string, _ time.Time, _ string) error {
			markCancelledCalled = true
			return nil
		},
	}

	uc := appshipping.NewCancelShipmentUseCase(newMockGormDB(t, 2), connRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, &eventrecordtest.Outbox{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx, appshipping.CancelShipmentInput{ShipmentID: "shipment-1"})

	require.NoError(t, err, "MyParcel rejecting the delete must not fail the use case")
	assert.False(t, result.DeletedOnMyParcel)
	assert.Contains(t, result.Message, "could not delete")
	assert.Contains(t, result.Message, "Shipment cannot be deleted")
	assert.True(t, markCancelledCalled, "local cancellation must proceed even when MyParcel rejects the delete")
}

func TestCancelShipment_alreadyCancelled_isIdempotentAndDoesNotCallMyParcelAgain(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}

	cancelledAt := time.Now()
	shipment := shippingDomain.Shipment{
		ID: "shipment-1", SalesOrderID: "order-1", ConnectionID: "conn-1", ExternalShipmentID: "555",
		CancelledAt: &cancelledAt, CancelNote: "Shipment deleted on MyParcel.",
	}
	deleteCalled := false
	client := &fakeCarrierClient{
		cancelShipmentFn: func(_ context.Context, _ string) error {
			deleteCalled = true
			return nil
		},
	}
	shipmentRepo := &fakeShipmentRepo{
		findByIDFn: func(_ context.Context, id string) (*shippingDomain.Shipment, error) { return &shipment, nil },
	}

	uc := appshipping.NewCancelShipmentUseCase(newMockGormDB(t, 1), connRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, &eventrecordtest.Outbox{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx, appshipping.CancelShipmentInput{ShipmentID: "shipment-1"})

	require.NoError(t, err)
	assert.False(t, deleteCalled, "an already-cancelled shipment must not re-call MyParcel")
	assert.Equal(t, "Shipment deleted on MyParcel.", result.Message)
}

func TestCancelShipment_shipmentNotFound_returnsError(t *testing.T) {
	connRepo := &mocks.ConnectionRepo{}
	shipmentRepo := &fakeShipmentRepo{
		findByIDFn: func(_ context.Context, _ string) (*shippingDomain.Shipment, error) {
			return nil, shippingDomain.ErrShipmentNotFound
		},
	}
	uc := appshipping.NewCancelShipmentUseCase(newMockGormDB(t, 1), connRepo, shipmentRepo,
		&fakeCarrierClientFactory{}, testEncSecret, &fakeRecordActivity{}, &eventrecordtest.Outbox{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	_, err := uc.Execute(ctx, appshipping.CancelShipmentInput{ShipmentID: "missing"})

	require.Error(t, err)
	assert.ErrorIs(t, err, shippingDomain.ErrShipmentNotFound)
}

// myParcelAPIError is a minimal stand-in for infrastructure/myparcel.APIError
// -- this test only needs its Error() text, not the real type (which would
// pull an infrastructure import into an application-layer test).
type myParcelAPIError struct {
	statusCode int
	body       string
}

func (e *myParcelAPIError) Error() string {
	return errors.New("myparcel: request failed: " + e.body).Error()
}
