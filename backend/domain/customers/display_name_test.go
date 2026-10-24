package customers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
)

func TestCustomerDisplayName(t *testing.T) {
	assert.Equal(t, "Ana Souza", customersDomain.Customer{FirstName: " Ana ", LastName: "Souza", Email: "a@x.nl"}.DisplayName())
	assert.Equal(t, "Restaurante Sabor", customersDomain.Customer{Billing: customersDomain.Address{Company: "Restaurante Sabor"}, Email: "a@x.nl"}.DisplayName())
	assert.Equal(t, "a@x.nl", customersDomain.Customer{Email: "a@x.nl"}.DisplayName())
}

func TestCustomerIdentifiable(t *testing.T) {
	assert.True(t, customersDomain.Customer{FirstName: "Ana"}.Identifiable())
	assert.True(t, customersDomain.Customer{Billing: customersDomain.Address{Company: "Sabor B.V."}}.Identifiable())
	assert.True(t, customersDomain.Customer{Email: "a@x.nl"}.Identifiable())
	assert.True(t, customersDomain.Customer{PhoneE164: "31612345678"}.Identifiable())
	assert.True(t, customersDomain.Customer{Billing: customersDomain.Address{Phone: "06 1234"}}.Identifiable())
	assert.False(t, customersDomain.Customer{LastName: "  "}.Identifiable())
}
