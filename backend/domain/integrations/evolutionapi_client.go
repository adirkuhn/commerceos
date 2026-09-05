package integrations

import "context"

// EvolutionAPIClient sends outbound WhatsApp text messages through one
// tenant's Evolution API instance -- the unofficial (Baileys-based) gateway
// this codebase talks to instead of WhatsApp's official Cloud API (see
// ProviderWhatsApp's doc comment). Deliberately just one method: Phase 1
// only ever replies with plain text (stock/price/expiry lookups), no
// media/template messages yet.
type EvolutionAPIClient interface {
	// SendText sends body to toPhone (E.164, no leading '+') through this
	// client's instance. Returns an error on any non-2xx response or
	// network failure -- callers treat that as "the reply was not
	// delivered", not a fatal condition (a WhatsApp send failing must not
	// fail the whole inbound-message dispatch retry loop forever, since
	// redelivery would just resend a question the customer already saw).
	SendText(ctx context.Context, toPhone, body string) error
}

// EvolutionAPIClientFactory builds an EvolutionAPIClient scoped to one
// connection's base URL/instance/API key -- same shape as
// MyParcelClientFactory, for the same reason: the use case that needs to
// send a message never constructs a concrete client itself (DI, see
// CLAUDE.md's SOLID D rule).
type EvolutionAPIClientFactory interface {
	NewClient(baseURL, instanceName, apiKey string) (EvolutionAPIClient, error)
}
