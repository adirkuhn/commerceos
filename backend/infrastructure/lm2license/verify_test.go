package lm2license_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/infrastructure/lm2license"
)

// testKeyPair signs a license the same way lm2 would (RS256, "payload"
// claim holding the JSON-encoded LicensePayload) and exposes its public
// key as a stub fetcher would return it, so VerifyAndGetPayload can be
// tested end-to-end without a real lm2 server.
type testKeyPair struct {
	private *rsa.PrivateKey
	pem     string
}

func newTestKeyPair(t *testing.T) testKeyPair {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	return testKeyPair{private: key, pem: string(pemBytes)}
}

func (k testKeyPair) sign(t *testing.T, payload lm2license.LicensePayload) string {
	t.Helper()
	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"payload": string(payloadJSON)})
	signed, err := token.SignedString(k.private)
	require.NoError(t, err)
	return signed
}

type stubFetcher struct {
	key string
	err error
}

func (s stubFetcher) GetPublicKey() (string, error) { return s.key, s.err }

func TestVerifyAndGetPayload_validSignatureRoundTripsThePayload(t *testing.T) {
	keys := newTestKeyPair(t)
	payload := lm2license.LicensePayload{
		CompanyID: "tenant-1", LicenseID: "lic-1", Status: "active",
		DeviceBinding: lm2license.DeviceBinding{DeviceID: "device-1", Mode: "pos"},
	}
	license := keys.sign(t, payload)
	svc := lm2license.NewTokenService(stubFetcher{key: keys.pem})

	got, err := svc.VerifyAndGetPayload(license)

	require.NoError(t, err)
	assert.Equal(t, "tenant-1", got.CompanyID)
	assert.Equal(t, "device-1", got.DeviceBinding.DeviceID)
	assert.Equal(t, "active", got.Status)
}

func TestVerifyAndGetPayload_rejectsASignatureFromADifferentKey(t *testing.T) {
	realKeys := newTestKeyPair(t)
	attackerKeys := newTestKeyPair(t)
	license := attackerKeys.sign(t, lm2license.LicensePayload{CompanyID: "tenant-1", Status: "active"})
	// Verifying against the REAL public key -- a license signed by anyone
	// else must not pass.
	svc := lm2license.NewTokenService(stubFetcher{key: realKeys.pem})

	_, err := svc.VerifyAndGetPayload(license)

	assert.ErrorIs(t, err, lm2license.ErrInvalidLicense)
}

func TestVerifyAndGetPayload_rejectsAMalformedLicenseString(t *testing.T) {
	keys := newTestKeyPair(t)
	svc := lm2license.NewTokenService(stubFetcher{key: keys.pem})

	_, err := svc.VerifyAndGetPayload("not-a-jwt")

	assert.ErrorIs(t, err, lm2license.ErrInvalidLicense)
}

func TestVerifyAndGetPayload_propagatesAPublicKeyFetchError(t *testing.T) {
	svc := lm2license.NewTokenService(stubFetcher{err: assert.AnError})

	_, err := svc.VerifyAndGetPayload("irrelevant")

	require.Error(t, err)
}

func TestVerifyAndGetPayload_rejectsAnInvalidPublicKeyPEM(t *testing.T) {
	svc := lm2license.NewTokenService(stubFetcher{key: "not a pem"})

	_, err := svc.VerifyAndGetPayload("irrelevant")

	require.Error(t, err)
}

func TestLicensePayload_IsUsable(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	cases := []struct {
		name    string
		payload lm2license.LicensePayload
		want    bool
	}{
		{"active, no expiry", lm2license.LicensePayload{Status: "active", CompanyID: "t1"}, true},
		{"active, not yet expired", lm2license.LicensePayload{Status: "active", CompanyID: "t1", ExpiresAt: &future}, true},
		{"active, expired", lm2license.LicensePayload{Status: "active", CompanyID: "t1", ExpiresAt: &past}, false},
		{"suspended", lm2license.LicensePayload{Status: "suspended", CompanyID: "t1"}, false},
		{"no company id", lm2license.LicensePayload{Status: "active"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.payload.IsUsable(now))
		})
	}
}
