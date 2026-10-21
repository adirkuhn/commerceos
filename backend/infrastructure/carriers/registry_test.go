package carriers_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/carriers"
)

type stubClient struct {
	integrationsDomain.CarrierClient
	name string
}

type stubFactory struct {
	name     string
	gotCreds integrationsDomain.CarrierCredentials
	err      error
}

func (f *stubFactory) NewClient(creds integrationsDomain.CarrierCredentials) (integrationsDomain.CarrierClient, error) {
	f.gotCreds = creds
	if f.err != nil {
		return nil, f.err
	}
	return stubClient{name: f.name}, nil
}

func TestRegistry_dispatchesToTheConnectionsProvider(t *testing.T) {
	myparcel := &stubFactory{name: "myparcel"}
	postnl := &stubFactory{name: "postnl"}
	registry := carriers.NewRegistry(map[integrationsDomain.Provider]integrationsDomain.CarrierClientFactory{
		integrationsDomain.ProviderMyParcel: myparcel,
		"postnl":                            postnl,
	})

	client, err := registry.NewClient(integrationsDomain.CarrierCredentials{Provider: "postnl", APIKey: "k"})

	require.NoError(t, err)
	assert.Equal(t, "postnl", client.(stubClient).name)
	assert.Equal(t, "k", postnl.gotCreds.APIKey)
	assert.Empty(t, myparcel.gotCreds.APIKey, "the other provider's factory must not be called")
}

func TestRegistry_unknownProviderIsNotSupported(t *testing.T) {
	registry := carriers.NewRegistry(map[integrationsDomain.Provider]integrationsDomain.CarrierClientFactory{
		integrationsDomain.ProviderMyParcel: &stubFactory{},
	})

	_, err := registry.NewClient(integrationsDomain.CarrierCredentials{Provider: integrationsDomain.ProviderStripe})

	assert.ErrorIs(t, err, integrationsDomain.ErrProviderNotSupported)
}

func TestRegistry_propagatesFactoryError(t *testing.T) {
	boom := errors.New("boom")
	registry := carriers.NewRegistry(map[integrationsDomain.Provider]integrationsDomain.CarrierClientFactory{
		integrationsDomain.ProviderMyParcel: &stubFactory{err: boom},
	})

	_, err := registry.NewClient(integrationsDomain.CarrierCredentials{Provider: integrationsDomain.ProviderMyParcel})

	assert.ErrorIs(t, err, boom)
}

func TestRegistry_supportsReportsRegisteredProviders(t *testing.T) {
	registry := carriers.NewRegistry(map[integrationsDomain.Provider]integrationsDomain.CarrierClientFactory{
		integrationsDomain.ProviderMyParcel: &stubFactory{},
	})

	assert.True(t, registry.Supports(integrationsDomain.ProviderMyParcel))
	assert.False(t, registry.Supports(integrationsDomain.ProviderBol))
}
