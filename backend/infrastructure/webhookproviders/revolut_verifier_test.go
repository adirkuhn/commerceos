package webhookproviders_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/infrastructure/webhookproviders"
)

func signRevolut(secret, timestampMillis string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("v1." + timestampMillis + "."))
	mac.Write(body)
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}

func revolutHeaders(secret string, at time.Time, body []byte) map[string]string {
	ts := strconv.FormatInt(at.UnixMilli(), 10)
	return map[string]string{
		"Revolut-Signature":         signRevolut(secret, ts, body),
		"Revolut-Request-Timestamp": ts,
	}
}

func TestRevolutVerifier_acceptsValidSignature(t *testing.T) {
	v := webhookproviders.NewRevolutVerifier()
	body := []byte(`{"event":"ORDER_COMPLETED"}`)

	err := v.Verify(body, revolutHeaders("wsk_test", time.Now(), body), "wsk_test")

	require.NoError(t, err)
}

func TestRevolutVerifier_acceptsAnyMatchingSignatureDuringRotation(t *testing.T) {
	v := webhookproviders.NewRevolutVerifier()
	body := []byte(`{"event":"ORDER_COMPLETED"}`)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	headers := map[string]string{
		"Revolut-Signature":         signRevolut("wsk_old", ts, body) + "," + signRevolut("wsk_new", ts, body),
		"Revolut-Request-Timestamp": ts,
	}

	err := v.Verify(body, headers, "wsk_new")

	require.NoError(t, err)
}

func TestRevolutVerifier_rejectsWrongSecret(t *testing.T) {
	v := webhookproviders.NewRevolutVerifier()
	body := []byte(`{"event":"ORDER_COMPLETED"}`)

	err := v.Verify(body, revolutHeaders("wsk_wrong", time.Now(), body), "wsk_test")

	assert.ErrorIs(t, err, webhooks.ErrInvalidSignature)
}

func TestRevolutVerifier_rejectsTamperedBody(t *testing.T) {
	v := webhookproviders.NewRevolutVerifier()
	body := []byte(`{"event":"ORDER_COMPLETED"}`)
	headers := revolutHeaders("wsk_test", time.Now(), body)

	err := v.Verify(append(body, 'x'), headers, "wsk_test")

	assert.ErrorIs(t, err, webhooks.ErrInvalidSignature)
}

func TestRevolutVerifier_rejectsStaleTimestamp(t *testing.T) {
	v := webhookproviders.NewRevolutVerifier()
	body := []byte(`{"event":"ORDER_COMPLETED"}`)

	err := v.Verify(body, revolutHeaders("wsk_test", time.Now().Add(-time.Hour), body), "wsk_test")

	assert.ErrorIs(t, err, webhooks.ErrInvalidSignature)
}

func TestRevolutVerifier_rejectsMissingHeaders(t *testing.T) {
	v := webhookproviders.NewRevolutVerifier()

	err := v.Verify([]byte(`{}`), map[string]string{}, "wsk_test")

	assert.ErrorIs(t, err, webhooks.ErrInvalidSignature)
}

func TestRevolutVerifier_rejectsEmptySecret(t *testing.T) {
	v := webhookproviders.NewRevolutVerifier()
	body := []byte(`{"event":"ORDER_COMPLETED"}`)

	err := v.Verify(body, revolutHeaders("wsk_test", time.Now(), body), "")

	assert.ErrorIs(t, err, webhooks.ErrInvalidSignature)
}

func TestRevolutVerifier_rejectsMalformedTimestamp(t *testing.T) {
	v := webhookproviders.NewRevolutVerifier()
	body := []byte(`{}`)
	headers := map[string]string{
		"Revolut-Signature":         signRevolut("wsk_test", "not-a-number", body),
		"Revolut-Request-Timestamp": "not-a-number",
	}

	err := v.Verify(body, headers, "wsk_test")

	assert.ErrorIs(t, err, webhooks.ErrInvalidSignature)
}

func TestRevolutVerifier_EventID_alwaysEmpty(t *testing.T) {
	v := webhookproviders.NewRevolutVerifier()

	assert.Equal(t, "", v.EventID(map[string]string{}, []byte(`{}`)))
}
