package webhookproviders_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	stripewebhook "github.com/stripe/stripe-go/v82/webhook"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/infrastructure/webhookproviders"
)

func signedStripePayload(t *testing.T, secret string, body []byte, at time.Time) (string, []byte) {
	t.Helper()
	signed := stripewebhook.GenerateTestSignedPayload(&stripewebhook.UnsignedPayload{
		Payload: body, Secret: secret, Timestamp: at,
	})
	return signed.Header, signed.Payload
}

func TestStripeVerifier_acceptsValidSignature(t *testing.T) {
	v := webhookproviders.NewStripeVerifier()
	body := []byte(`{"id":"evt_1"}`)
	header, payload := signedStripePayload(t, "whsec_test", body, time.Now())

	err := v.Verify(payload, map[string]string{"Stripe-Signature": header}, "whsec_test")

	require.NoError(t, err)
}

func TestStripeVerifier_rejectsWrongSecret(t *testing.T) {
	v := webhookproviders.NewStripeVerifier()
	body := []byte(`{"id":"evt_1"}`)
	header, payload := signedStripePayload(t, "whsec_wrong", body, time.Now())

	err := v.Verify(payload, map[string]string{"Stripe-Signature": header}, "whsec_test")

	assert.ErrorIs(t, err, webhooks.ErrInvalidSignature)
}

func TestStripeVerifier_rejectsTamperedBody(t *testing.T) {
	v := webhookproviders.NewStripeVerifier()
	header, payload := signedStripePayload(t, "whsec_test", []byte(`{"id":"evt_1"}`), time.Now())

	err := v.Verify(append(payload, 'x'), map[string]string{"Stripe-Signature": header}, "whsec_test")

	assert.ErrorIs(t, err, webhooks.ErrInvalidSignature)
}

func TestStripeVerifier_rejectsStaleTimestamp(t *testing.T) {
	v := webhookproviders.NewStripeVerifier()
	body := []byte(`{"id":"evt_1"}`)
	header, payload := signedStripePayload(t, "whsec_test", body, time.Now().Add(-time.Hour))

	err := v.Verify(payload, map[string]string{"Stripe-Signature": header}, "whsec_test")

	assert.ErrorIs(t, err, webhooks.ErrInvalidSignature)
}

func TestStripeVerifier_rejectsMissingHeader(t *testing.T) {
	v := webhookproviders.NewStripeVerifier()

	err := v.Verify([]byte(`{}`), map[string]string{}, "whsec_test")

	assert.ErrorIs(t, err, webhooks.ErrInvalidSignature)
}

func TestStripeVerifier_rejectsEmptySecret(t *testing.T) {
	v := webhookproviders.NewStripeVerifier()
	body := []byte(`{"id":"evt_1"}`)
	header, payload := signedStripePayload(t, "whsec_test", body, time.Now())

	err := v.Verify(payload, map[string]string{"Stripe-Signature": header}, "")

	assert.ErrorIs(t, err, webhooks.ErrInvalidSignature)
}

func TestStripeVerifier_EventID_returnsThePayloadsID(t *testing.T) {
	v := webhookproviders.NewStripeVerifier()

	assert.Equal(t, "evt_123", v.EventID(map[string]string{}, []byte(`{"id":"evt_123"}`)))
}

func TestStripeVerifier_EventID_emptyOnMalformedPayload(t *testing.T) {
	v := webhookproviders.NewStripeVerifier()

	assert.Equal(t, "", v.EventID(map[string]string{}, []byte(`not json`)))
}
