package channels_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

type customersHarness struct {
	connRepo     *stubConnectionRepo
	wooFactory   *stubWooClientFactory
	wooClient    *stubWooClient
	customerRepo *stubOrdersCustomerRepo
}

func newCustomersHarness(customers map[int][]integrationsDomain.WooCustomer) *customersHarness {
	wooClient := &stubWooClient{customersByPage: customers}
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "consumer-secret")
	if err != nil {
		panic(err)
	}
	return &customersHarness{
		connRepo: &stubConnectionRepo{conn: &integrationsDomain.Connection{
			ID: "conn-1", Provider: integrationsDomain.ProviderWooCommerce,
			StoreURL: "https://mystore.com", ConsumerKey: "ck", ConsumerSecretEncrypted: encrypted,
		}},
		wooFactory:   &stubWooClientFactory{client: wooClient},
		wooClient:    wooClient,
		customerRepo: &stubOrdersCustomerRepo{byWooID: map[int64]customersDomain.Customer{}, byEmail: map[string]customersDomain.Customer{}},
	}
}

func (h *customersHarness) newImportUseCase(t *testing.T, n int) *appchannels.ImportWooCustomersUseCase {
	t.Helper()
	return appchannels.NewImportWooCustomersUseCase(newMockGormDB(t, n), h.connRepo, h.wooFactory, testEncryptionSecret, h.customerRepo, &eventrecordtest.Outbox{})
}

func simpleWooCustomer() integrationsDomain.WooCustomer {
	return integrationsDomain.WooCustomer{
		ID: 42, Email: "jane@example.com", FirstName: "Jane", LastName: "Doe",
		Username: "janedoe", Role: "customer", IsPayingCustomer: true, AvatarURL: "https://example.com/avatar.png",
		Billing:      integrationsDomain.WooAddress{FirstName: "Jane", LastName: "Doe", Address1: "123 Main St", City: "Springfield", Country: "US", Email: "jane@example.com"},
		Shipping:     integrationsDomain.WooAddress{FirstName: "Jane", LastName: "Doe", Address1: "456 Oak Ave", City: "Springfield", Country: "US"},
		DateCreated:  "2026-01-01T10:00:00",
		DateModified: "2026-01-01T10:00:00",
	}
}

func TestImportWooCustomers_createsANewCustomer(t *testing.T) {
	h := newCustomersHarness(map[int][]integrationsDomain.WooCustomer{1: {simpleWooCustomer()}})
	uc := h.newImportUseCase(t, 3)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooCustomersInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 0, result.Skipped)
	assert.Empty(t, result.Errors)
	require.Len(t, h.customerRepo.created, 1)
	created := h.customerRepo.created[0]
	require.NotNil(t, created.WooCustomerID)
	assert.Equal(t, int64(42), *created.WooCustomerID)
	assert.Equal(t, "jane@example.com", created.Email)
	assert.Equal(t, "Jane", created.FirstName)
	assert.Equal(t, "janedoe", created.Username)
	assert.True(t, created.IsPayingCustomer)
	assert.Equal(t, "123 Main St", created.Billing.Address1)
	assert.Equal(t, "jane@example.com", created.Billing.Email)
	assert.Equal(t, "456 Oak Ave", created.Shipping.Address1)
	assert.Empty(t, created.Shipping.Email, "shipping has no email field, same convention as orders/oms Address")
}

func TestImportWooCustomers_skipsACustomerAlreadyLinkedByWooCustomerID(t *testing.T) {
	h := newCustomersHarness(map[int][]integrationsDomain.WooCustomer{1: {simpleWooCustomer()}})
	h.customerRepo.byWooID[42] = customersDomain.Customer{ID: "customer-1", Email: "jane@example.com"}
	uc := h.newImportUseCase(t, 2)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooCustomersInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 1, result.Skipped)
	assert.Empty(t, h.customerRepo.created)
}

func TestImportWooCustomers_pagesUntilAnEmptyPage(t *testing.T) {
	h := newCustomersHarness(map[int][]integrationsDomain.WooCustomer{
		1: {simpleWooCustomer()},
		2: {{ID: 43, Email: "second@example.com"}},
	})
	uc := h.newImportUseCase(t, 5)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooCustomersInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	assert.Equal(t, 2, result.Created)
	assert.Equal(t, 3, h.wooClient.customersCalls, "pages 1, 2, then 3 (empty) to detect the end")
}

func TestImportWooCustomers_recordsAPerCustomerErrorAndContinuesTheRun(t *testing.T) {
	h := newCustomersHarness(map[int][]integrationsDomain.WooCustomer{
		1: {simpleWooCustomer()},
	})
	h.customerRepo.createErr = assert.AnError
	uc := h.newImportUseCase(t, 3)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooCustomersInput{ConnectionID: "conn-1"})

	require.NoError(t, err, "a single customer's failure must not abort the whole import")
	assert.Equal(t, 0, result.Created)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "42")
}

func TestImportWooCustomers_propagatesAWooRequestFailure(t *testing.T) {
	h := newCustomersHarness(nil)
	h.wooClient.customersErr = assert.AnError
	uc := h.newImportUseCase(t, 1)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooCustomersInput{ConnectionID: "conn-1"})

	require.Error(t, err)
	assert.ErrorIs(t, err, integrationsDomain.ErrWooRequestFailed)
}

func TestImportWooCustomers_propagatesConnectionNotFound(t *testing.T) {
	h := newCustomersHarness(nil)
	h.connRepo.conn = nil
	h.connRepo.findErr = integrationsDomain.ErrConnectionNotFound
	uc := h.newImportUseCase(t, 1)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooCustomersInput{ConnectionID: "conn-1"})

	require.Error(t, err)
	assert.ErrorIs(t, err, integrationsDomain.ErrConnectionNotFound)
}
