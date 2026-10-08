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

type recordingBolClientFactory struct {
	client                 integrationsDomain.BolClient
	clientID, clientSecret string
}

func (f *recordingBolClientFactory) NewClient(clientID, clientSecret string) (integrationsDomain.BolClient, error) {
	f.clientID, f.clientSecret = clientID, clientSecret
	return f.client, nil
}

func TestBolConnectionClients_buildsClientFromDecryptedCredentials(t *testing.T) {
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "bol-secret")
	require.NoError(t, err)
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{
		ID: "conn-1", Provider: integrationsDomain.ProviderBol, ConsumerKey: "bol-client-id", ConsumerSecretEncrypted: encrypted,
	}}
	bol := &fakeBolClient{}
	factory := &recordingBolClientFactory{client: bol}

	client, err := appchannels.NewBolConnectionClients(newMockGormDB(t, 1), connRepo, factory, testEncryptionSecret).
		ClientFor(context.Background(), "conn-1")

	require.NoError(t, err)
	assert.Same(t, bol, client)
	assert.Equal(t, "bol-client-id", factory.clientID)
	assert.Equal(t, "bol-secret", factory.clientSecret)
}

func TestBolConnectionClients_notABolConnection_isRejected(t *testing.T) {
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: "conn-1", Provider: integrationsDomain.ProviderWooCommerce}}

	_, err := appchannels.NewBolConnectionClients(newMockGormDB(t, 1), connRepo, &recordingBolClientFactory{}, testEncryptionSecret).
		ClientFor(context.Background(), "conn-1")

	assert.ErrorIs(t, err, integrationsDomain.ErrProviderNotSupported)
}
