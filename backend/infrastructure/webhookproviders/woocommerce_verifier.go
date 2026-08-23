// Package webhookproviders adapts each third-party provider's webhook
// signature scheme to domain/webhooks.ProviderVerifier. One file per
// provider (Open/Closed) -- see registry.go for how they're wired up.
package webhookproviders

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"

	"github.com/adirkuhn/commerceos/domain/webhooks"
)

// ProviderWooCommerce is the registry key + URL path segment for WooCommerce.
const ProviderWooCommerce = "woocommerce"

// wcSignatureHeader is the header WooCommerce signs every webhook delivery
// with: base64(HMAC-SHA256(raw request body, the webhook's configured
// secret)). Header name/casing per WooCommerce's REST API webhook docs;
// Go's net/http canonicalizes header names, and this codebase passes
// headers through as a plain map (see infrastructure/handlers/webhooks),
// so the lookup here must match however that map's keys were built.
const wcSignatureHeader = "X-Wc-Webhook-Signature"

// WooCommerceVerifier implements domain/webhooks.ProviderVerifier for
// WooCommerce.
type WooCommerceVerifier struct{}

func NewWooCommerceVerifier() *WooCommerceVerifier { return &WooCommerceVerifier{} }

func (WooCommerceVerifier) Verify(rawBody []byte, headers map[string]string, secret string) error {
	got := headers[wcSignatureHeader]
	if got == "" {
		return webhooks.ErrInvalidSignature
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	// hmac.Equal is constant-time -- a naive == would leak how many
	// leading bytes matched via response timing, letting an attacker
	// iterate toward a valid forgery.
	if !hmac.Equal([]byte(got), []byte(want)) {
		return webhooks.ErrInvalidSignature
	}
	return nil
}

// EventID always defers to ReceiveWebhookUseCase's raw-body-hash fallback.
// WooCommerce's delivery-tracking headers (X-WC-Webhook-ID identifies the
// webhook subscription, not a single delivery) aren't a documented,
// stable per-delivery id we can rely on for dedup -- guessing a header
// name here risks a silent no-op (falls through to the hash anyway) or,
// worse, a wrong assumption baked in as if verified. The raw-body hash is
// deterministic and naturally dedups WooCommerce's own retries, which
// resend the identical payload.
func (WooCommerceVerifier) EventID(headers map[string]string, rawBody []byte) string {
	return ""
}
