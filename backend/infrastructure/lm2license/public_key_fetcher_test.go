package lm2license

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractPublicKey_acceptsABarePEMBody(t *testing.T) {
	pem := "-----BEGIN PUBLIC KEY-----\nabc123\n-----END PUBLIC KEY-----"

	got, err := extractPublicKey([]byte(pem))

	require.NoError(t, err)
	assert.Equal(t, pem, got)
}

func TestExtractPublicKey_acceptsAJSONEnvelope(t *testing.T) {
	got, err := extractPublicKey([]byte(`{"public_key": "-----BEGIN PUBLIC KEY-----\nabc123\n-----END PUBLIC KEY-----"}`))

	require.NoError(t, err)
	assert.Equal(t, "-----BEGIN PUBLIC KEY-----\nabc123\n-----END PUBLIC KEY-----", got)
}

func TestExtractPublicKey_rejectsUnrecognizedFormat(t *testing.T) {
	_, err := extractPublicKey([]byte("garbage"))

	assert.Error(t, err)
}
