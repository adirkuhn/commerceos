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

func strPtr(s string) *string { return &s }

func TestNotifyOrderStatusUseCase_knownCustomerWithPhone_sendsMessage(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewNotifyOrderStatusUseCase(
		newMockGormDB(t, 1),
		&fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{
			"order-1": {ID: "order-1", Number: "1001", Status: omsDomain.SalesOrderStatusShipped, CustomerID: strPtr("cust-1")},
		}},
		&fakeCustomerRepo{byID: map[string]customersDomain.Customer{
			"cust-1": {ID: "cust-1", FirstName: "Jane", PhoneE164: "31612345678"},
		}},
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
	)

	err := uc.Execute(context.Background(), "order-1")

	require.NoError(t, err)
	assert.Equal(t, "31612345678", client.sentTo)
	assert.Contains(t, client.sentBody, "Jane")
	assert.Contains(t, client.sentBody, "1001")
	assert.Contains(t, client.sentBody, "on its way")
}

func TestNotifyOrderStatusUseCase_guestOrder_sendsNothing(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewNotifyOrderStatusUseCase(
		newMockGormDB(t, 1),
		&fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{
			"order-1": {ID: "order-1", Number: "1001", Status: omsDomain.SalesOrderStatusShipped, CustomerID: nil},
		}},
		&fakeCustomerRepo{},
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
	)

	err := uc.Execute(context.Background(), "order-1")

	require.NoError(t, err)
	assert.Empty(t, client.sentTo, "a guest/walk-in order has no customer to notify")
}

func TestNotifyOrderStatusUseCase_customerWithNoPhoneOnFile_sendsNothing(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewNotifyOrderStatusUseCase(
		newMockGormDB(t, 1),
		&fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{
			"order-1": {ID: "order-1", Number: "1001", Status: omsDomain.SalesOrderStatusShipped, CustomerID: strPtr("cust-1")},
		}},
		&fakeCustomerRepo{byID: map[string]customersDomain.Customer{
			"cust-1": {ID: "cust-1", FirstName: "Jane", PhoneE164: ""},
		}},
		&fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
	)

	err := uc.Execute(context.Background(), "order-1")

	require.NoError(t, err)
	assert.Empty(t, client.sentTo, "no verified phone on file means nowhere safe to send")
}

func TestNotifyOrderStatusUseCase_noWhatsAppChannelConfigured_sendsNothing(t *testing.T) {
	client := &fakeEvolutionAPIClient{}
	uc := appwhatsapp.NewNotifyOrderStatusUseCase(
		newMockGormDB(t, 1),
		&fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{
			"order-1": {ID: "order-1", Number: "1001", Status: omsDomain.SalesOrderStatusShipped, CustomerID: strPtr("cust-1")},
		}},
		&fakeCustomerRepo{byID: map[string]customersDomain.Customer{
			"cust-1": {ID: "cust-1", FirstName: "Jane", PhoneE164: "31612345678"},
		}},
		&fakeConnectionRepo{conns: nil},
		&fakeEvolutionAPIClientFactory{client: client},
		testEncryptionSecret,
	)

	err := uc.Execute(context.Background(), "order-1")

	require.NoError(t, err)
	assert.Empty(t, client.sentTo)
}
