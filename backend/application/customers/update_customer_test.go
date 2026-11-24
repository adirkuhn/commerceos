package customers_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appcustomers "github.com/adirkuhn/commerceos/application/customers"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
)

func TestUpdateCustomer_updatesFields(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c1", Email: "old@example.com"}}
	uc := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, &memShippingRepo{}, &eventrecordtest.Outbox{})

	c, err := uc.Execute(context.Background(), appcustomers.UpdateCustomerInput{
		ID: "c1", Email: "new@example.com", FirstName: "Jane", LastName: "Doe",
	})

	require.NoError(t, err)
	require.Equal(t, "new@example.com", c.Email)
	require.Len(t, repo.updated, 1)
	require.Equal(t, "new@example.com", repo.updated[0].Email)
}

func TestUpdateCustomer_updatesCustomerNote(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c1", Email: "jane@example.com", CustomerNote: "old note"}}
	uc := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, &memShippingRepo{}, &eventrecordtest.Outbox{})

	c, err := uc.Execute(context.Background(), appcustomers.UpdateCustomerInput{
		ID: "c1", Email: "jane@example.com", CustomerNote: "Call before delivery",
	})

	require.NoError(t, err)
	require.Equal(t, "Call before delivery", c.CustomerNote)
	require.Len(t, repo.updated, 1)
	require.Equal(t, "Call before delivery", repo.updated[0].CustomerNote)
}

func TestUpdateCustomer_propagatesNotFound(t *testing.T) {
	repo := &stubCustomerRepo{findErr: customersDomain.ErrCustomerNotFound}
	uc := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, &memShippingRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appcustomers.UpdateCustomerInput{ID: "missing"})

	require.ErrorIs(t, err, customersDomain.ErrCustomerNotFound)
}

func TestUpdateCustomer_needsSomethingToNameIt(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c-1", Email: "a@x.nl"}}

	_, err := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, &memShippingRepo{}, &eventrecordtest.Outbox{}).
		Execute(context.Background(), appcustomers.UpdateCustomerInput{ID: "c-1"})

	assert.ErrorIs(t, err, customersDomain.ErrCustomerUnidentified)
}

func TestUpdateCustomer_changesCompanyFields(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c-1", FirstName: "Ana", ClientNumber: "7"}}
	company, name, kvk, vat, number := customersDomain.TypeCompany, "Sabor B.V.", "34073293", "NL001B01", "3286"

	c, err := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, &memShippingRepo{}, &eventrecordtest.Outbox{}).
		Execute(context.Background(), appcustomers.UpdateCustomerInput{ID: "c-1", FirstName: "Ana",
			Type: &company, CompanyName: &name, KVKNumber: &kvk, VATNumber: &vat, ClientNumber: &number})

	require.NoError(t, err)
	assert.Equal(t, customersDomain.TypeCompany, c.Type)
	assert.Equal(t, "Sabor B.V.", c.CompanyName)
	assert.Equal(t, "34073293", c.KVKNumber)
	assert.Equal(t, "NL001B01", c.VATNumber)
	assert.Equal(t, "3286", c.ClientNumber)
}

func TestUpdateCustomer_keepsCompanyFieldsNotSent(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c-1", Type: customersDomain.TypeCompany, CompanyName: "Sabor B.V.", ClientNumber: "7", KVKNumber: "1"}}

	c, err := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, &memShippingRepo{}, &eventrecordtest.Outbox{}).
		Execute(context.Background(), appcustomers.UpdateCustomerInput{ID: "c-1", Email: "info@sabor.nl"})

	require.NoError(t, err)
	assert.Equal(t, customersDomain.TypeCompany, c.Type)
	assert.Equal(t, "Sabor B.V.", c.CompanyName)
	assert.Equal(t, "7", c.ClientNumber)
	assert.Equal(t, "1", c.KVKNumber)
}

func TestUpdateCustomer_aCompanyCannotLoseItsName(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c-1", Type: customersDomain.TypeCompany, CompanyName: "Sabor B.V.", Email: "a@x.nl"}}
	empty := ""

	_, err := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, &memShippingRepo{}, &eventrecordtest.Outbox{}).
		Execute(context.Background(), appcustomers.UpdateCustomerInput{ID: "c-1", Email: "a@x.nl", CompanyName: &empty})

	assert.ErrorIs(t, err, customersDomain.ErrCompanyNameRequired)
}

func addressPtr(a customersDomain.Address) *customersDomain.Address { return &a }

func TestUpdateCustomer_leavingShippingOutKeepsTheBook(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c1", Email: "a@x.nl", Shipping: customersDomain.Address{Address1: "Main 1"}}}
	book := &memShippingRepo{}
	book.seed("c1", "Main 1", true)

	c, err := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, book, &eventrecordtest.Outbox{}).
		Execute(context.Background(), appcustomers.UpdateCustomerInput{ID: "c1", Email: "a@x.nl"})

	require.NoError(t, err)
	assert.Equal(t, "Main 1", c.Shipping.Address1)
	assert.Len(t, book.rows, 1)
}

func TestUpdateCustomer_shippingSentBecomesTheDefaultAddress(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c1", Email: "a@x.nl"}}
	book := &memShippingRepo{}
	outbox := &eventrecordtest.Outbox{}

	c, err := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, book, outbox).
		Execute(context.Background(), appcustomers.UpdateCustomerInput{ID: "c1", Email: "a@x.nl", Shipping: addressPtr(customersDomain.Address{Address1: "Dock 2"})})

	require.NoError(t, err)
	assert.Equal(t, "Dock 2", c.Shipping.Address1)
	require.Len(t, book.rows, 1)
	assert.True(t, book.rows[0].IsDefault)
	assert.Equal(t, "Dock 2", book.rows[0].Address.Address1)
	assert.Contains(t, outbox.Subjects(), "customers.shipping_address.added")
}

func TestUpdateCustomer_shippingSentEditsTheDefaultAddress(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c1", Email: "a@x.nl", Shipping: customersDomain.Address{Address1: "Main 1"}}}
	book := &memShippingRepo{}
	book.seed("c1", "Main 1", true)
	book.seed("c1", "Dock 2", false)
	outbox := &eventrecordtest.Outbox{}

	c, err := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, book, outbox).
		Execute(context.Background(), appcustomers.UpdateCustomerInput{ID: "c1", Email: "a@x.nl", Shipping: addressPtr(customersDomain.Address{Address1: "Main 1", City: "Delft"})})

	require.NoError(t, err)
	assert.Equal(t, "Delft", c.Shipping.City)
	require.Len(t, book.rows, 2)
	assert.Equal(t, "Delft", book.rows[0].Address.City)
	assert.Contains(t, outbox.Subjects(), "customers.shipping_address.updated")
}

func TestUpdateCustomer_shipToBillingCopiesBillingAndFollowsIt(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c1", Email: "a@x.nl", Shipping: customersDomain.Address{Address1: "Main 1"}}}
	book := &memShippingRepo{}
	book.seed("c1", "Main 1", true)
	on := true

	c, err := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, book, &eventrecordtest.Outbox{}).
		Execute(context.Background(), appcustomers.UpdateCustomerInput{
			ID: "c1", Email: "a@x.nl", ShipToBilling: &on,
			Billing:  customersDomain.Address{Address1: "Bill 1", Email: "bill@x.nl"},
			Shipping: addressPtr(customersDomain.Address{Address1: "Ignored 9"}),
		})

	require.NoError(t, err)
	assert.True(t, c.ShipToBilling)
	assert.Equal(t, "Bill 1", c.Shipping.Address1)
	assert.Empty(t, c.Shipping.Email)
	assert.Equal(t, "Main 1", book.rows[0].Address.Address1, "the book is left alone while shipping to billing")
}

func TestUpdateCustomer_shipToBillingOffGoesBackToTheDefault(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c1", Email: "a@x.nl", ShipToBilling: true, Billing: customersDomain.Address{Address1: "Bill 1"}}}
	book := &memShippingRepo{}
	book.seed("c1", "Main 1", true)
	off := false

	c, err := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, book, &eventrecordtest.Outbox{}).
		Execute(context.Background(), appcustomers.UpdateCustomerInput{ID: "c1", Email: "a@x.nl", ShipToBilling: &off, Billing: customersDomain.Address{Address1: "Bill 1"}})

	require.NoError(t, err)
	assert.False(t, c.ShipToBilling)
	assert.Equal(t, "Main 1", c.Shipping.Address1)
}
