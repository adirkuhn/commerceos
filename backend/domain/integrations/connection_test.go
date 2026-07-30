package integrations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/integrations"
)

func TestStoreURL_Validate_valid(t *testing.T) {
	cases := []string{
		"https://mystore.com",
		"https://mystore.com/",
		"http://localhost:8080",
	}
	for _, c := range cases {
		assert.NoError(t, integrations.StoreURL(c).Validate(), "url: %s", c)
	}
}

func TestStoreURL_Validate_invalid(t *testing.T) {
	cases := []string{
		"",
		"not-a-url",
		"ftp://mystore.com",
		"mystore.com",
		"   ",
	}
	for _, c := range cases {
		assert.ErrorIs(t, integrations.StoreURL(c).Validate(), integrations.ErrStoreURLInvalid, "url: %q", c)
	}
}
