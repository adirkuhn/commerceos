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
