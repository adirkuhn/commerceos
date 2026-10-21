package shipping_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func validAddress() shippingDomain.ShippingAddress {
	return shippingDomain.ShippingAddress{
		Company: " Cantinho Brasileiro ", Street: "Damrak", Number: "1", NumberSuffix: "A",
		PostalCode: "1012 ab", City: "Amsterdam", CC: "nl", Email: "shop@example.com", Phone: "+31612345678",
	}
}

func TestNewShippingAddress_normalizes(t *testing.T) {
	a, err := shippingDomain.NewShippingAddress(validAddress())

	require.NoError(t, err)
	assert.Equal(t, "Cantinho Brasileiro", a.Company)
	assert.Equal(t, "1012AB", a.PostalCode)
	assert.Equal(t, "NL", a.CC)
}

func TestNewShippingAddress_requiresTheFieldsCarriersPrint(t *testing.T) {
	for name, mutate := range map[string]func(*shippingDomain.ShippingAddress){
		"company":     func(a *shippingDomain.ShippingAddress) { a.Company = "" },
		"street":      func(a *shippingDomain.ShippingAddress) { a.Street = " " },
		"number":      func(a *shippingDomain.ShippingAddress) { a.Number = "" },
		"postal code": func(a *shippingDomain.ShippingAddress) { a.PostalCode = "" },
		"city":        func(a *shippingDomain.ShippingAddress) { a.City = "" },
		"country":     func(a *shippingDomain.ShippingAddress) { a.CC = "Netherlands" },
		"email":       func(a *shippingDomain.ShippingAddress) { a.Email = "not-an-email" },
	} {
		t.Run(name, func(t *testing.T) {
			in := validAddress()
			mutate(&in)

			_, err := shippingDomain.NewShippingAddress(in)

			assert.ErrorIs(t, err, shippingDomain.ErrShippingAddressInvalid)
		})
	}
}

func TestNewShippingAddress_emailIsOptional(t *testing.T) {
	in := validAddress()
	in.Email = ""

	_, err := shippingDomain.NewShippingAddress(in)

	assert.NoError(t, err)
}
