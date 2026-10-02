package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

func TestWooConnectionClients_buildsAClientForAWooConnection(t *testing.T) {
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "consumer-secret")
	require.NoError(t, err)
	client := &stubWooClient{}
	clients := appchannels.NewWooConnectionClients(newMockGormDB(t, 1), &stubConnectionRepo{conn: &integrationsDomain.Connection{
		ID: "conn-1", Provider: integrationsDomain.ProviderWooCommerce, ConsumerSecretEncrypted: encrypted,
	}}, &stubWooClientFactory{client: client}, testEncryptionSecret)

	got, err := clients.ClientFor(context.Background(), "conn-1")

	require.NoError(t, err)
	assert.Same(t, client, got)
}

func TestWooConnectionClients_rejectsAConnectionOfAnotherProvider(t *testing.T) {
	clients := appchannels.NewWooConnectionClients(newMockGormDB(t, 1), &stubConnectionRepo{conn: &integrationsDomain.Connection{
		ID: "conn-bol", Provider: integrationsDomain.ProviderBol,
	}}, &stubWooClientFactory{client: &stubWooClient{}}, testEncryptionSecret)

	_, err := clients.ClientFor(context.Background(), "conn-bol")

	assert.ErrorIs(t, err, integrationsDomain.ErrProviderNotSupported)
}

func TestWooConnectionClients_propagatesConnectionNotFound(t *testing.T) {
	clients := appchannels.NewWooConnectionClients(newMockGormDB(t, 1), &stubConnectionRepo{findErr: integrationsDomain.ErrConnectionNotFound},
		&stubWooClientFactory{client: &stubWooClient{}}, testEncryptionSecret)

	_, err := clients.ClientFor(context.Background(), "missing")

	assert.ErrorIs(t, err, integrationsDomain.ErrConnectionNotFound)
}
