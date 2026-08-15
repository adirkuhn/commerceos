package lm2license

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewHTTPPublicKeyFetcher_SkipsTLSVerifyForDevLocalhost reproduces a
// real production bug: fetching lm2's public key from a local dev instance
// (self-signed cert via a local CA nothing else trusts) failed TLS
// verification using the default client, which silently surfaced as
// "invalid or inactive license" for every POS request — the same
// .dev.localhost carve-out the POS desktop's own license.Client already
// applies (see akantio_pos/src/license/client.go's identical check).
func TestNewHTTPPublicKeyFetcher_SkipsTLSVerifyForDevLocalhost(t *testing.T) {
	f := NewHTTPPublicKeyFetcher("https://api-lm2.dev.localhost/api/v1/pos/public-keys/current")

	transport, ok := f.httpClient.Transport.(*http.Transport)
	require.True(t, ok, "expected a custom *http.Transport for a .dev.localhost URL")
	require.NotNil(t, transport.TLSClientConfig)
	assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
}

// TestNewHTTPPublicKeyFetcher_DoesNotSkipTLSVerifyForProductionHost guards
// the other direction — a real production URL must never get the insecure
// carve-out.
func TestNewHTTPPublicKeyFetcher_DoesNotSkipTLSVerifyForProductionHost(t *testing.T) {
	f := NewHTTPPublicKeyFetcher("https://lm2.example.com/api/public-key")

	assert.Nil(t, f.httpClient.Transport, "a production URL must use the default (verifying) transport")
}

func TestNewHTTPPublicKeyFetcher_MalformedURL_DoesNotPanicAndStaysSecure(t *testing.T) {
	f := NewHTTPPublicKeyFetcher("://not a url")

	assert.Nil(t, f.httpClient.Transport)
}

func TestExtractPublicKey_acceptsABarePEMBody(t *testing.T) {
	pem := "-----BEGIN PUBLIC KEY-----\nabc123\n-----END PUBLIC KEY-----"

	got, err := extractPublicKey([]byte(pem))

	require.NoError(t, err)
	assert.Equal(t, pem, got)
}

// TestExtractPublicKey_acceptsLM2sActualDataEnvelope is lm2's real response
// shape for GET public-keys/current — confirmed against the POS desktop's
// own already-working license.Client.GetPublicKey (akantio_pos/src/license/client.go),
// which has been calling this exact endpoint successfully. The original
// bare {"public_key": ...} guess this package shipped with (see the other
// test below) was wrong and caused every /pos/* request to fail
// verification with "invalid or inactive license" until this was found.
func TestExtractPublicKey_acceptsLM2sActualDataEnvelope(t *testing.T) {
	got, err := extractPublicKey([]byte(`{"data": {"public_key": "-----BEGIN PUBLIC KEY-----\nabc123\n-----END PUBLIC KEY-----"}}`))

	require.NoError(t, err)
	assert.Equal(t, "-----BEGIN PUBLIC KEY-----\nabc123\n-----END PUBLIC KEY-----", got)
}

// TestExtractPublicKey_acceptsAFlatJSONEnvelope keeps the original guessed
// shape working as a fallback — zero-cost, and harmless if lm2 (or a
// future endpoint) ever responds without the data wrapper.
func TestExtractPublicKey_acceptsAFlatJSONEnvelope(t *testing.T) {
	got, err := extractPublicKey([]byte(`{"public_key": "-----BEGIN PUBLIC KEY-----\nabc123\n-----END PUBLIC KEY-----"}`))

	require.NoError(t, err)
	assert.Equal(t, "-----BEGIN PUBLIC KEY-----\nabc123\n-----END PUBLIC KEY-----", got)
}

func TestExtractPublicKey_rejectsUnrecognizedFormat(t *testing.T) {
	_, err := extractPublicKey([]byte("garbage"))

	assert.Error(t, err)
}
