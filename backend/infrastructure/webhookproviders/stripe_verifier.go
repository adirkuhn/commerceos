package webhookproviders

import (
	"encoding/json"

	"github.com/stripe/stripe-go/v82/webhook"

	"github.com/adirkuhn/commerceos/domain/webhooks"
)

// ProviderStripe is the registry key + URL path segment for Stripe.
const ProviderStripe = "stripe"

// stripeSignatureHeader is the header Stripe signs every webhook delivery
// with: "t=<timestamp>,v1=<hmac-sha256 hex>[,v0=...]". See
// https://docs.stripe.com/webhooks/signatures for the manual-verification
// algorithm this delegates to the official SDK for.
const stripeSignatureHeader = "Stripe-Signature"

// StripeVerifier implements domain/webhooks.ProviderVerifier for Stripe.
// Verification is delegated to stripe-go's own webhook.ValidatePayload
// (HMAC-SHA256 over "{timestamp}.{rawBody}", constant-time compare, 5min
// default timestamp tolerance against replay) rather than hand-rolled here
// -- Stripe's SDK is an approved dependency already used for outbound
// Checkout Session calls (infrastructure/stripe.Client), so there's no
// reason to reimplement a well-tested library's own signature check.
type StripeVerifier struct{}

func NewStripeVerifier() *StripeVerifier { return &StripeVerifier{} }

func (StripeVerifier) Verify(rawBody []byte, headers map[string]string, secret string) error {
	got := headers[stripeSignatureHeader]
	if got == "" || secret == "" {
		return webhooks.ErrInvalidSignature
	}
	if err := webhook.ValidatePayload(rawBody, got, secret); err != nil {
		return webhooks.ErrInvalidSignature
	}
	return nil
}

// EventID returns Stripe's own event id (evt_...), which is stable and
// unique per delivery attempt for the same logical event -- unlike
// MyParcel/WooCommerce/WhatsApp, Stripe's payload has a documented,
// reliable per-event id, so there's no need to fall back to a raw-body
// hash. Returns "" (raw-body-hash fallback) on any parse failure -- this
// method must never itself reject a payload; Verify already did that.
func (StripeVerifier) EventID(headers map[string]string, rawBody []byte) string {
	var event struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rawBody, &event); err != nil {
		return ""
	}
	return event.ID
}
