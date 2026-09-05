package webhookproviders_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/infrastructure/webhookproviders"
)

func TestEvolutionAPIVerifier_matchingToken_verifies(t *testing.T) {
	v := webhookproviders.NewEvolutionAPIVerifier()

	err := v.Verify([]byte(`{}`), map[string]string{"__query_token": "secret-abc"}, "secret-abc")

	require.NoError(t, err)
}

func TestEvolutionAPIVerifier_wrongToken_rejects(t *testing.T) {
	v := webhookproviders.NewEvolutionAPIVerifier()

	err := v.Verify([]byte(`{}`), map[string]string{"__query_token": "wrong"}, "secret-abc")

	require.ErrorIs(t, err, webhooks.ErrInvalidSignature)
}

func TestEvolutionAPIVerifier_missingToken_rejects(t *testing.T) {
	v := webhookproviders.NewEvolutionAPIVerifier()

	err := v.Verify([]byte(`{}`), map[string]string{}, "secret-abc")

	require.ErrorIs(t, err, webhooks.ErrInvalidSignature)
}

func TestEvolutionAPIVerifier_emptySecret_rejects(t *testing.T) {
	v := webhookproviders.NewEvolutionAPIVerifier()

	err := v.Verify([]byte(`{}`), map[string]string{"__query_token": "anything"}, "")

	require.ErrorIs(t, err, webhooks.ErrInvalidSignature, "an unconfigured secret must never match by coincidence")
}

func TestEvolutionAPIVerifier_EventID_alwaysEmpty(t *testing.T) {
	v := webhookproviders.NewEvolutionAPIVerifier()

	assert.Equal(t, "", v.EventID(map[string]string{}, []byte(`{}`)))
}
