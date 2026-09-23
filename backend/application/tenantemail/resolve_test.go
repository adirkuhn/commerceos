package tenantemail_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/tenantemail"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

const secret = "0123456789abcdef0123456789abcdef"

type fakeClient struct{ from string }

func (fakeClient) Send(context.Context, messaging.Message) error { return nil }

type recordingFactory struct{ host, port, user, pass, from string }

func (f *recordingFactory) NewClient(host, port, username, password, from string) messaging.Client {
	f.host, f.port, f.user, f.pass, f.from = host, port, username, password, from
	return fakeClient{from: from}
}

func TestResolve_prefersTheTenantsEmailConnection(t *testing.T) {
	enc, err := crypto.Encrypt(secret, "s3cret")
	require.NoError(t, err)
	port, from := "587", "shop@example.nl"
	conns := []integrationsDomain.Connection{
		{Provider: integrationsDomain.ProviderWooCommerce},
		{Provider: integrationsDomain.ProviderEmail, StoreURL: integrationsDomain.StoreURL("smtp.example.nl"), SMTPPort: &port, ConsumerKey: "user", ConsumerSecretEncrypted: enc, EmailFrom: &from},
	}
	factory := &recordingFactory{}

	c, err := tenantemail.Resolve(conns, secret, factory, fakeClient{from: "fallback"})

	require.NoError(t, err)
	assert.Equal(t, fakeClient{from: "shop@example.nl"}, c)
	assert.Equal(t, "smtp.example.nl", factory.host)
	assert.Equal(t, "587", factory.port)
	assert.Equal(t, "s3cret", factory.pass)
}

func TestResolve_fallsBackWithoutAnEmailConnection(t *testing.T) {
	c, err := tenantemail.Resolve(nil, secret, &recordingFactory{}, fakeClient{from: "fallback"})
	require.NoError(t, err)
	assert.Equal(t, fakeClient{from: "fallback"}, c)
}

func TestResolve_undecryptableSecretIsAnError(t *testing.T) {
	conns := []integrationsDomain.Connection{{Provider: integrationsDomain.ProviderEmail, ConsumerSecretEncrypted: "garbage"}}
	_, err := tenantemail.Resolve(conns, secret, &recordingFactory{}, fakeClient{})
	assert.Error(t, err)
}
