package customers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
)

func TestCompanyCustomerIsNamedByItsCompany(t *testing.T) {
	c := customersDomain.Customer{Type: customersDomain.TypeCompany, CompanyName: "Claus Hoofddorp B.V.", FirstName: "Kylian"}
	assert.Equal(t, "Claus Hoofddorp B.V.", c.DisplayName())
}

func TestPrivateCustomerIsNamedByThePerson(t *testing.T) {
	c := customersDomain.Customer{Type: customersDomain.TypePrivate, CompanyName: "Old Co", FirstName: "Ana", LastName: "Souza"}
	assert.Equal(t, "Ana Souza", c.DisplayName())
}

func TestValidate(t *testing.T) {
	assert.NoError(t, customersDomain.Customer{Type: customersDomain.TypeCompany, CompanyName: "Sabor B.V."}.Validate(), "a company needs no person or email")
	assert.NoError(t, customersDomain.Customer{Type: customersDomain.TypePrivate, FirstName: "Ana"}.Validate())
	assert.NoError(t, customersDomain.Customer{FirstName: "Ana"}.Validate(), "no type reads as private")
	assert.ErrorIs(t, customersDomain.Customer{Type: customersDomain.TypeCompany, FirstName: "Ana"}.Validate(), customersDomain.ErrCompanyNameRequired)
	assert.ErrorIs(t, customersDomain.Customer{Type: "reseller", FirstName: "Ana"}.Validate(), customersDomain.ErrInvalidCustomerType)
	assert.ErrorIs(t, customersDomain.Customer{Type: customersDomain.TypePrivate}.Validate(), customersDomain.ErrCustomerUnidentified)
}

func TestTypeOrPrivate(t *testing.T) {
	assert.Equal(t, customersDomain.TypePrivate, customersDomain.Customer{}.TypeOrPrivate())
	assert.Equal(t, customersDomain.TypeCompany, customersDomain.Customer{Type: customersDomain.TypeCompany}.TypeOrPrivate())
}
