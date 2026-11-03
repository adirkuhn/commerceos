package integrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStoreURLHost(t *testing.T) {
	assert.Equal(t, "cantinhobrasileiro.nl", StoreURL("https://cantinhobrasileiro.nl/").Host())
	assert.Equal(t, "shop.example:8080", StoreURL("http://shop.example:8080/wp").Host())
	assert.Equal(t, "not a url", StoreURL("not a url").Host(), "falls back to the raw value")
}
