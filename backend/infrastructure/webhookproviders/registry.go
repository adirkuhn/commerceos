package webhookproviders

import "github.com/adirkuhn/commerceos/domain/webhooks"

// stubProviders lists every provider name the ingestion endpoint routes
// (so POST /webhooks/v1/:tenant_id/:provider returns 501, not 404) but
// doesn't yet verify for real. Add a provider's real ProviderVerifier here
// and remove it from this list when it's implemented -- see
// StubVerifier's doc comment for why these aren't guessed at.
var stubProviders = []string{
	"bol", "moneybird", "shopify", "lightspeed", "myparcel", "sendcloud",
}

// NewRegistry wires every known provider name to its ProviderVerifier.
func NewRegistry() map[string]webhooks.ProviderVerifier {
	reg := map[string]webhooks.ProviderVerifier{
		ProviderWooCommerce: NewWooCommerceVerifier(),
	}
	for _, name := range stubProviders {
		reg[name] = NewStubVerifier()
	}
	return reg
}
