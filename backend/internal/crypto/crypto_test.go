package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/internal/crypto"
)

func TestEncryptDecrypt_roundTrips(t *testing.T) {
	ciphertext, err := crypto.Encrypt("super-secret-key", "consumer-secret-value")
	require.NoError(t, err)

	plaintext, err := crypto.Decrypt("super-secret-key", ciphertext)
	require.NoError(t, err)
	assert.Equal(t, "consumer-secret-value", plaintext)
}

func TestDecrypt_wrongKeyFails(t *testing.T) {
	ciphertext, err := crypto.Encrypt("correct-key", "consumer-secret-value")
	require.NoError(t, err)

	_, err = crypto.Decrypt("wrong-key", ciphertext)
	assert.Error(t, err)
}

func TestDecrypt_tamperedCiphertextFails(t *testing.T) {
	ciphertext, err := crypto.Encrypt("some-key", "consumer-secret-value")
	require.NoError(t, err)

	tampered := ciphertext[:len(ciphertext)-4] + "abcd"
	_, err = crypto.Decrypt("some-key", tampered)
	assert.Error(t, err)
}

func TestEncrypt_producesDifferentCiphertextEachTime(t *testing.T) {
	a, err := crypto.Encrypt("some-key", "same-plaintext")
	require.NoError(t, err)
	b, err := crypto.Encrypt("some-key", "same-plaintext")
	require.NoError(t, err)

	assert.NotEqual(t, a, b, "random nonce must make each encryption unique")
}
