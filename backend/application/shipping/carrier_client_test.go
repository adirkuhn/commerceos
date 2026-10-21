package shipping

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

const helperEncSecret = "test-encryption-secret-32bytes!"

type recordingCarrierFactory struct {
	client   integrationsDomain.CarrierClient
	err      error
	gotCreds integrationsDomain.CarrierCredentials
}

func (f *recordingCarrierFactory) NewClient(creds integrationsDomain.CarrierCredentials) (integrationsDomain.CarrierClient, error) {
	f.gotCreds = creds
	return f.client, f.err
}

type nopCarrierClient struct {
	integrationsDomain.CarrierClient
}

func encryptedKey(t *testing.T, plaintext string) string {
	t.Helper()
	enc, err := crypto.Encrypt(helperEncSecret, plaintext)
	require.NoError(t, err)
	return enc
}

func TestBuildCarrierClient_passesProviderAndDecryptedKeyToFactory(t *testing.T) {
	client := &nopCarrierClient{}
	factory := &recordingCarrierFactory{client: client}
	conn := integrationsDomain.Connection{Provider: integrationsDomain.ProviderMyParcel, ConsumerSecretEncrypted: encryptedKey(t, "secret-key")}

	got, err := buildCarrierClient(factory, helperEncSecret, conn)

	require.NoError(t, err)
	assert.Same(t, client, got)
	assert.Equal(t, integrationsDomain.ProviderMyParcel, factory.gotCreds.Provider)
	assert.Equal(t, "secret-key", factory.gotCreds.APIKey)
}

func TestBuildCarrierClient_undecryptableKeyFails(t *testing.T) {
	conn := integrationsDomain.Connection{Provider: integrationsDomain.ProviderMyParcel, ConsumerSecretEncrypted: "not-encrypted"}

	_, err := buildCarrierClient(&recordingCarrierFactory{}, helperEncSecret, conn)

	assert.ErrorContains(t, err, "decrypt api key")
}

func TestBuildCarrierClient_unsupportedProviderFails(t *testing.T) {
	factory := &recordingCarrierFactory{err: integrationsDomain.ErrProviderNotSupported}
	conn := integrationsDomain.Connection{Provider: integrationsDomain.ProviderBol, ConsumerSecretEncrypted: encryptedKey(t, "k")}

	_, err := buildCarrierClient(factory, helperEncSecret, conn)

	assert.ErrorIs(t, err, integrationsDomain.ErrProviderNotSupported)
}
