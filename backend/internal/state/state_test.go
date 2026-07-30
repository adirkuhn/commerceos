package state_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/internal/state"
)

var testSecret = []byte("test-state-secret")

func TestMintAndParseConnectState_roundTrips(t *testing.T) {
	now := time.Now()
	token, err := state.MintConnectState(testSecret, "tenant-1", "https://mystore.com", now, 10*time.Minute)
	require.NoError(t, err)

	got, err := state.ParseConnectState(testSecret, token, now.Add(1*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, "tenant-1", got.TenantID)
	assert.Equal(t, "https://mystore.com", got.StoreURL)
}

func TestParseConnectState_rejectsExpired(t *testing.T) {
	now := time.Now()
	token, err := state.MintConnectState(testSecret, "tenant-1", "https://mystore.com", now, 10*time.Minute)
	require.NoError(t, err)

	_, err = state.ParseConnectState(testSecret, token, now.Add(11*time.Minute))
	assert.Error(t, err)
}

func TestParseConnectState_rejectsTamperedToken(t *testing.T) {
	now := time.Now()
	token, err := state.MintConnectState(testSecret, "tenant-1", "https://mystore.com", now, 10*time.Minute)
	require.NoError(t, err)

	tampered := token[:len(token)-2] + "xx"
	_, err = state.ParseConnectState(testSecret, tampered, now.Add(1*time.Minute))
	assert.Error(t, err)
}

func TestParseConnectState_rejectsWrongSecret(t *testing.T) {
	now := time.Now()
	token, err := state.MintConnectState(testSecret, "tenant-1", "https://mystore.com", now, 10*time.Minute)
	require.NoError(t, err)

	_, err = state.ParseConnectState([]byte("different-secret"), token, now.Add(1*time.Minute))
	assert.Error(t, err)
}

func TestMintConnectState_producesDifferentTokensEachTime(t *testing.T) {
	now := time.Now()
	a, err := state.MintConnectState(testSecret, "tenant-1", "https://mystore.com", now, 10*time.Minute)
	require.NoError(t, err)
	b, err := state.MintConnectState(testSecret, "tenant-1", "https://mystore.com", now, 10*time.Minute)
	require.NoError(t, err)

	assert.NotEqual(t, a, b, "random nonce must make each mint unique")
}
