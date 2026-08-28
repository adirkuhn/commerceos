package webhookproviders

import (
	"crypto/hmac"

	"github.com/adirkuhn/commerceos/domain/webhooks"
)

// ProviderMyParcel is the registry key + URL path segment for MyParcel.
const ProviderMyParcel = "myparcel"

// myParcelQueryTokenKey is the headers-map key
// infrastructure/handlers/webhooks.Handler injects the `?token=` query
// parameter under, since MyParcel's webhook has nothing in the request
// body or headers to verify against -- see this file's own doc comment.
const myParcelQueryTokenKey = "__query_token"

// MyParcelVerifier implements domain/webhooks.ProviderVerifier for
// MyParcel. Unlike WooCommerce (HMAC-SHA256 over the body), MyParcel signs
// nothing at all -- its docs state webhook authenticity relies on HTTPS
// transport alone, no signature header or scheme. Verification here
// instead checks a shared-secret token embedded in the callback URL itself
// against the tenant's Connection.WebhookSecretEncrypted, so only whoever
// holds that URL (us, via the URL we ourselves chose) can produce a
// matching request. The token travels as a path segment
// (api/webhooks/openapi.yaml's receiveWebhookWithToken route,
// RegisterMyParcelWebhookUseCase / CreateProviderWebhookUseCase both build
// this shape) -- infrastructure/handlers/webhooks.Handler forwards either
// that or the older `?token=` query-string form (still accepted for
// callback URLs registered before the path form existed) into the same
// headers key this verifier reads. Weaker than an HMAC by construction (a
// URL can leak into logs/proxies more easily than a header), but it's what
// the vendor's API actually supports.
type MyParcelVerifier struct{}

func NewMyParcelVerifier() *MyParcelVerifier { return &MyParcelVerifier{} }

func (MyParcelVerifier) Verify(rawBody []byte, headers map[string]string, secret string) error {
	got := headers[myParcelQueryTokenKey]
	if got == "" || secret == "" {
		return webhooks.ErrInvalidSignature
	}
	// hmac.Equal is constant-time -- same reasoning as WooCommerceVerifier's
	// use of it, even though this isn't itself an HMAC.
	if !hmac.Equal([]byte(got), []byte(secret)) {
		return webhooks.ErrInvalidSignature
	}
	return nil
}

// EventID always defers to ReceiveWebhookUseCase's raw-body-hash fallback.
// MyParcel's shipment_status_change payload has no documented per-delivery
// id (shipment_id identifies the shipment, not a specific delivery
// attempt), and the raw-body hash naturally dedups an identical retried
// payload.
func (MyParcelVerifier) EventID(headers map[string]string, rawBody []byte) string {
	return ""
}
