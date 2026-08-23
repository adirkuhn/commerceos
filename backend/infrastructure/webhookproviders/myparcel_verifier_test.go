package webhookproviders_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/infrastructure/webhookproviders"
)

func TestMyParcelVerifier_matchingToken_verifies(t *testing.T) {
	v := webhookproviders.NewMyParcelVerifier()

	err := v.Verify([]byte(`{}`), map[string]string{"__query_token": "secret-abc"}, "secret-abc")

	require.NoError(t, err)
}

func TestMyParcelVerifier_wrongToken_rejects(t *testing.T) {
	v := webhookproviders.NewMyParcelVerifier()

	err := v.Verify([]byte(`{}`), map[string]string{"__query_token": "wrong"}, "secret-abc")

	require.ErrorIs(t, err, webhooks.ErrInvalidSignature)
}

func TestMyParcelVerifier_missingToken_rejects(t *testing.T) {
	v := webhookproviders.NewMyParcelVerifier()

	err := v.Verify([]byte(`{}`), map[string]string{}, "secret-abc")

	require.ErrorIs(t, err, webhooks.ErrInvalidSignature)
}

func TestMyParcelVerifier_emptySecret_rejects(t *testing.T) {
	v := webhookproviders.NewMyParcelVerifier()

	err := v.Verify([]byte(`{}`), map[string]string{"__query_token": "anything"}, "")

	require.ErrorIs(t, err, webhooks.ErrInvalidSignature, "an unconfigured secret must never match by coincidence")
}

func TestMyParcelVerifier_EventID_alwaysEmpty(t *testing.T) {
	v := webhookproviders.NewMyParcelVerifier()

	assert.Equal(t, "", v.EventID(map[string]string{}, []byte(`{}`)))
}
