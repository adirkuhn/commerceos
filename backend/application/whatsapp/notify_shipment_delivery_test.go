package whatsapp_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appwhatsapp "github.com/adirkuhn/commerceos/application/whatsapp"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

func newDeliveryTestUseCase(t *testing.T, client *fakeEvolutionAPIClient) *appwhatsapp.NotifyShipmentDeliveryUseCase {
	t.Helper()
	return appwhatsapp.NewNotifyShipmentDeliveryUseCase(
		newMockGormDB(t, 1),
		&fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{
			"order-1": {ID: "order-1", Number: "1001", CustomerID: strPtr("cust-1")},
		}},
		&fakeCustomerRepo{byID: map[string]customersDomain.Customer{
			"cust-1": {ID: "cust-1", FirstName: "Jane", PhoneE164: "31612345678"},
		}},
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
	)
}

func TestNotifyShipmentDeliveryUseCase_deliveredToRecipient_sendsPlainDeliveredMessage(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := newDeliveryTestUseCase(t, client)

	err := uc.Execute(context.Background(), "order-1", 7)

	require.NoError(t, err)
	assert.Equal(t, "31612345678", client.sentTo)
	assert.Contains(t, client.sentBody, "delivered!")
	assert.NotContains(t, client.sentBody, "collect point")
}

func TestNotifyShipmentDeliveryUseCase_pickedUpByRecipient_alsoPlainDeliveredMessage(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := newDeliveryTestUseCase(t, client)

	err := uc.Execute(context.Background(), "order-1", 9)

	require.NoError(t, err)
	assert.NotContains(t, client.sentBody, "collect point")
}

func TestNotifyShipmentDeliveryUseCase_readyForPickup_sendsCollectPointMessage(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := newDeliveryTestUseCase(t, client)

	err := uc.Execute(context.Background(), "order-1", 8)

	require.NoError(t, err)
	assert.Contains(t, client.sentBody, "collect point")
}

func TestNotifyShipmentDeliveryUseCase_atAgreedLocation_sendsCollectPointMessage(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := newDeliveryTestUseCase(t, client)

	err := uc.Execute(context.Background(), "order-1", 19)

	require.NoError(t, err)
	assert.Contains(t, client.sentBody, "collect point")
}

func TestNotifyShipmentDeliveryUseCase_guestOrder_sendsNothing(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewNotifyShipmentDeliveryUseCase(
		newMockGormDB(t, 1),
		&fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{
			"order-1": {ID: "order-1", Number: "1001", CustomerID: nil},
		}},
		&fakeCustomerRepo{},
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
	)

	err := uc.Execute(context.Background(), "order-1", 7)

	require.NoError(t, err)
	assert.Empty(t, client.sentTo)
}
