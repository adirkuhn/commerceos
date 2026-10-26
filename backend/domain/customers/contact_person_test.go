package customers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
)

func TestContactPerson_Name(t *testing.T) {
	assert.Equal(t, "Kylian Klockenbrink", customersDomain.ContactPerson{FirstName: " Kylian", LastName: "Klockenbrink "}.Name())
	assert.Equal(t, "Kylian", customersDomain.ContactPerson{FirstName: "Kylian"}.Name())
}

func TestContactPerson_Validate(t *testing.T) {
	assert.NoError(t, customersDomain.ContactPerson{LastName: "Klockenbrink"}.Validate(), "email and phone are optional")
	assert.ErrorIs(t, customersDomain.ContactPerson{Email: "k@claus.nl"}.Validate(), customersDomain.ErrContactPersonNameRequired)
}
