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
	uc := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

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
	uc := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

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
	uc := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appcustomers.UpdateCustomerInput{ID: "missing"})

	require.ErrorIs(t, err, customersDomain.ErrCustomerNotFound)
}

func TestUpdateCustomer_needsSomethingToNameIt(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c-1", Email: "a@x.nl"}}

	_, err := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{}).
		Execute(context.Background(), appcustomers.UpdateCustomerInput{ID: "c-1"})

	assert.ErrorIs(t, err, customersDomain.ErrCustomerUnidentified)
}

func TestUpdateCustomer_changesCompanyFields(t *testing.T) {
	repo := &stubCustomerRepo{findResult: &customersDomain.Customer{ID: "c-1", FirstName: "Ana", ClientNumber: "7"}}
	company, name, kvk, vat, number := customersDomain.TypeCompany, "Sabor B.V.", "34073293", "NL001B01", "3286"

	c, err := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{}).
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

	c, err := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{}).
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

	_, err := appcustomers.NewUpdateCustomerUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{}).
		Execute(context.Background(), appcustomers.UpdateCustomerInput{ID: "c-1", Email: "a@x.nl", CompanyName: &empty})

	assert.ErrorIs(t, err, customersDomain.ErrCompanyNameRequired)
}
