package shipping_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

// The local cancellation and its event are one transaction: no event, no
// cancellation.
func TestCancelShipment_FailsWhenTheEventCantBeRecorded(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encryptedFor(t, "api-key")}
	connRepo := &mocks.ConnectionRepo{FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil }}
	shipment := shippingDomain.Shipment{ID: "shipment-1", SalesOrderID: "order-1", ConnectionID: "conn-1", ExternalShipmentID: "555"}
	shipmentRepo := &fakeShipmentRepo{findByIDFn: func(_ context.Context, _ string) (*shippingDomain.Shipment, error) { return &shipment, nil }}
	client := &fakeCarrierClient{cancelShipmentFn: func(_ context.Context, _ string) error { return nil }}
	boom := errors.New("outbox down")

	_, err := appshipping.NewCancelShipmentUseCase(newMockGormDB(t, 2), connRepo, shipmentRepo,
		&fakeCarrierClientFactory{client: client}, testEncSecret, &fakeRecordActivity{}, &eventrecordtest.Outbox{Err: boom}).
		Execute(context.Background(), appshipping.CancelShipmentInput{ShipmentID: "shipment-1"})

	assert.ErrorIs(t, err, boom)
}
