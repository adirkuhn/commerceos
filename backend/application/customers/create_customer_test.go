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

func TestCreateCustomer_setsFields(t *testing.T) {
	repo := &stubCustomerRepo{}
	uc := appcustomers.NewCreateCustomerUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	wooID := int64(42)
	c, err := uc.Execute(context.Background(), appcustomers.CreateCustomerInput{
		WooCustomerID: &wooID, Email: "jane@example.com", FirstName: "Jane", LastName: "Doe",
		Username: "janedoe", Role: "customer",
		Billing:  customersDomain.Address{FirstName: "Jane", LastName: "Doe", Address1: "123 Main St", City: "Springfield", Country: "US"},
		Shipping: customersDomain.Address{FirstName: "Jane", LastName: "Doe", Address1: "123 Main St", City: "Springfield", Country: "US"},
	})

	require.NoError(t, err)
	require.Equal(t, "customer-1", c.ID)
	require.Equal(t, "jane@example.com", c.Email)
	require.Equal(t, int64(42), *c.WooCustomerID)
	require.Equal(t, "123 Main St", c.Billing.Address1)
	require.True(t, repo.createCalled)
}

func TestCreateCustomer_setsCustomerNote(t *testing.T) {
	repo := &stubCustomerRepo{}
	uc := appcustomers.NewCreateCustomerUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	c, err := uc.Execute(context.Background(), appcustomers.CreateCustomerInput{
		Email: "jane@example.com", CustomerNote: "VIP, call before delivery",
	})

	require.NoError(t, err)
	require.Equal(t, "VIP, call before delivery", c.CustomerNote)
}

func TestCreateCustomer_propagatesRepoError(t *testing.T) {
	repo := &stubCustomerRepo{createErr: customersDomain.ErrCustomerNotFound}
	uc := appcustomers.NewCreateCustomerUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appcustomers.CreateCustomerInput{Email: "jane@example.com"})

	require.Error(t, err)
}

func TestCreateCustomer_withoutEmailIsFine(t *testing.T) {
	repo := &stubCustomerRepo{}
	c, err := appcustomers.NewCreateCustomerUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{}).
		Execute(context.Background(), appcustomers.CreateCustomerInput{FirstName: "Ana", Billing: customersDomain.Address{Company: "Sabor"}})

	require.NoError(t, err)
	assert.Equal(t, "Ana", c.FirstName)
}

func TestCreateCustomer_needsSomethingToNameIt(t *testing.T) {
	_, err := appcustomers.NewCreateCustomerUseCase(nil, &stubCustomerRepo{}, &eventrecordtest.Outbox{}).
		Execute(context.Background(), appcustomers.CreateCustomerInput{})

	assert.ErrorIs(t, err, customersDomain.ErrCustomerUnidentified)
}

func TestCreateCustomer_aCompanyNeedsOnlyItsCompanyName(t *testing.T) {
	repo := &stubCustomerRepo{}
	outbox := &eventrecordtest.Outbox{}
	c, err := appcustomers.NewCreateCustomerUseCase(newMockGormDB(t), repo, outbox).
		Execute(context.Background(), appcustomers.CreateCustomerInput{
			Type: customersDomain.TypeCompany, CompanyName: "Claus Hoofddorp B.V.", KVKNumber: "34073293", VATNumber: "NL001B01", ClientNumber: "3286",
		})

	require.NoError(t, err)
	assert.Equal(t, customersDomain.TypeCompany, c.Type)
	assert.Equal(t, "Claus Hoofddorp B.V.", c.CompanyName)
	assert.Equal(t, "34073293", c.KVKNumber)
	assert.Equal(t, "NL001B01", c.VATNumber)
	assert.Equal(t, "3286", c.ClientNumber)
}

func TestCreateCustomer_aCompanyWithoutCompanyNameIsRejected(t *testing.T) {
	_, err := appcustomers.NewCreateCustomerUseCase(nil, &stubCustomerRepo{}, &eventrecordtest.Outbox{}).
		Execute(context.Background(), appcustomers.CreateCustomerInput{Type: customersDomain.TypeCompany, FirstName: "Kylian"})

	assert.ErrorIs(t, err, customersDomain.ErrCompanyNameRequired)
}
