package customers_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/customers"
)

func addr(line string) customers.ShippingAddress {
	return customers.ShippingAddress{Address: customers.Address{Address1: line, City: "Utrecht", Country: "NL"}}
}

func TestShippingAddress_validate(t *testing.T) {
	assert.NoError(t, addr("Main 1").Validate())
	assert.ErrorIs(t, addr("  ").Validate(), customers.ErrShippingAddressLineRequired)

	long := addr("Main 1")
	long.Label = strings.Repeat("x", customers.MaxShippingLabelLength+1)
	assert.ErrorIs(t, long.Validate(), customers.ErrShippingLabelTooLong)
}

func TestShippingBook_firstAddressBecomesTheDefault(t *testing.T) {
	var book customers.ShippingBook

	first, err := book.Add(addr("Main 1"), false)
	require.NoError(t, err)
	assert.True(t, first.IsDefault)
	assert.Equal(t, 0, first.Position)

	book = customers.ShippingBook{first}
	second, err := book.Add(addr("Dock 2"), false)
	require.NoError(t, err)
	assert.False(t, second.IsDefault)
	assert.Equal(t, 1, second.Position)
}

func TestShippingBook_addAsDefault(t *testing.T) {
	book := customers.ShippingBook{withID(addr("Main 1"), "a", true, 0)}

	added, err := book.Add(addr("Dock 2"), true)

	require.NoError(t, err)
	assert.True(t, added.IsDefault, "the caller then moves the default to it")
}

func TestShippingBook_addValidatesAndCaps(t *testing.T) {
	var book customers.ShippingBook
	_, err := book.Add(addr(""), false)
	assert.ErrorIs(t, err, customers.ErrShippingAddressLineRequired)

	for i := 0; i < customers.MaxShippingAddresses; i++ {
		book = append(book, withID(addr("Main"), fmt.Sprint(i), i == 0, i))
	}
	_, err = book.Add(addr("One too many"), false)
	assert.ErrorIs(t, err, customers.ErrTooManyShippingAddresses)
}

func TestShippingBook_find(t *testing.T) {
	book := customers.ShippingBook{withID(addr("Main 1"), "a", true, 0)}

	found, err := book.Find("a")
	require.NoError(t, err)
	assert.Equal(t, "Main 1", found.Address.Address1)

	_, err = book.Find("nope")
	assert.ErrorIs(t, err, customers.ErrShippingAddressNotFound)
}

func TestShippingBook_removingTheDefaultPromotesTheNextOne(t *testing.T) {
	book := customers.ShippingBook{
		withID(addr("Main 1"), "a", true, 0),
		withID(addr("Dock 2"), "b", false, 1),
		withID(addr("Shed 3"), "c", false, 2),
	}

	next, err := book.NextDefaultAfterRemoving("a")
	require.NoError(t, err)
	assert.Equal(t, "b", next)

	next, err = book.NextDefaultAfterRemoving("c")
	require.NoError(t, err)
	assert.Empty(t, next, "removing another one keeps the default")

	next, err = customers.ShippingBook{withID(addr("Main 1"), "a", true, 0)}.NextDefaultAfterRemoving("a")
	require.NoError(t, err)
	assert.Empty(t, next, "the last one leaves no default")

	_, err = book.NextDefaultAfterRemoving("nope")
	assert.ErrorIs(t, err, customers.ErrShippingAddressNotFound)
}

func TestShippingBook_default(t *testing.T) {
	book := customers.ShippingBook{withID(addr("Main 1"), "a", false, 0), withID(addr("Dock 2"), "b", true, 1)}
	assert.Equal(t, "b", book.Default().ID)
	assert.Nil(t, customers.ShippingBook{}.Default())
}

func TestCustomer_effectiveShipping(t *testing.T) {
	billing := customers.Address{Address1: "Bill 1", City: "Delft", Email: "a@b.nl", Phone: "06"}
	def := withID(addr("Main 1"), "a", true, 0)
	def.Address.Email = "ignored@x.nl"

	c := customers.Customer{Billing: billing}
	assert.Equal(t, "Main 1", c.EffectiveShipping(&def).Address1)
	assert.Empty(t, c.EffectiveShipping(&def).Email, "shipping carries no email")
	assert.Equal(t, customers.Address{}, c.EffectiveShipping(nil))

	c.ShipToBilling = true
	got := c.EffectiveShipping(&def)
	assert.Equal(t, "Bill 1", got.Address1, "ships to the billing address")
	assert.Equal(t, "06", got.Phone)
	assert.Empty(t, got.Email)
}

func TestAddress_isEmpty(t *testing.T) {
	assert.True(t, customers.Address{}.IsEmpty())
	assert.True(t, customers.Address{Email: "only@email.nl"}.IsEmpty(), "an email alone is no address")
	assert.False(t, customers.Address{City: "Delft"}.IsEmpty())
}

func withID(a customers.ShippingAddress, id string, isDefault bool, position int) customers.ShippingAddress {
	a.ID, a.IsDefault, a.Position = id, isDefault, position
	return a
}
