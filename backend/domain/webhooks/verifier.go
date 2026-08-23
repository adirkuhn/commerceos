package webhooks

// ProviderVerifier is one webhook provider's signature scheme, adapted to a
// common shape so ReceiveWebhookUseCase never branches on provider name
// itself (Open/Closed: a new provider is a new ProviderVerifier
// implementation registered in infrastructure/webhookproviders, never a new
// `if provider == "x"` in the use case).
type ProviderVerifier interface {
	// Verify checks rawBody+headers against secret (the tenant's decrypted
	// per-provider webhook secret). Returns ErrInvalidSignature on any
	// mismatch, ErrProviderNotImplemented if this provider has no real
	// verification yet (stub providers).
	Verify(rawBody []byte, headers map[string]string, secret string) error
	// EventID extracts this provider's own delivery/idempotency id from the
	// headers if it sends one, else "" -- ReceiveWebhookUseCase falls back
	// to hashing rawBody when this returns "".
	EventID(headers map[string]string, rawBody []byte) string
}
