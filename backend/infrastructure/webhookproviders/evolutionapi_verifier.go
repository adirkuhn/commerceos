package webhookproviders

import (
	"crypto/hmac"

	"github.com/adirkuhn/commerceos/domain/webhooks"
)

// ProviderWhatsApp is the registry key + URL path segment for the WhatsApp
// integration's inbound webhook (see integrations.ProviderWhatsApp for the
// Connection side of this).
const ProviderWhatsApp = "whatsapp"

// EvolutionAPIVerifier implements domain/webhooks.ProviderVerifier for
// Evolution API (the self-hosted, Baileys-based WhatsApp gateway this
// codebase talks to -- see integrations.ProviderWhatsApp's doc comment).
// Like MyParcel, Evolution API signs nothing on its outbound webhook
// deliveries -- verification is a shared-secret token embedded in the
// callback URL we ourselves register with it, travelling as a path segment
// (api/webhooks/openapi.yaml's receiveWebhookWithToken route) into the same
// queryTokenKey every shared-secret-only provider verifies against.
type EvolutionAPIVerifier struct{}

func NewEvolutionAPIVerifier() *EvolutionAPIVerifier { return &EvolutionAPIVerifier{} }

func (EvolutionAPIVerifier) Verify(rawBody []byte, headers map[string]string, secret string) error {
	got := headers[queryTokenKey]
	if got == "" || secret == "" {
		return webhooks.ErrInvalidSignature
	}
	if !hmac.Equal([]byte(got), []byte(secret)) {
		return webhooks.ErrInvalidSignature
	}
	return nil
}

// EventID always defers to ReceiveWebhookUseCase's raw-body-hash fallback.
// Evolution API's messages.upsert payload carries the WhatsApp message's
// own id (key.id), which would make a better idempotency key than a raw
// body hash, but that field name is unconfirmed against a real webhook
// delivery -- same caution as this codebase's other vendor-payload doc
// comments (e.g. MyParcelClient's). Left as the safe fallback until
// confirmed.
func (EvolutionAPIVerifier) EventID(headers map[string]string, rawBody []byte) string {
	return ""
}
