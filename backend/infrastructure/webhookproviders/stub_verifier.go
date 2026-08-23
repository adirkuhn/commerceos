package webhookproviders

import "github.com/adirkuhn/commerceos/domain/webhooks"

// StubVerifier registers a provider name in the routing table (so
// :provider resolves to something and gets a 501, not a 404) without
// claiming to verify anything. Real signature schemes vary enough
// per-vendor (header names, algorithms, secret formats) that guessing one
// without vendor docs/sample payloads to test against risks either
// rejecting every legitimate webhook or, worse, accepting a forged one --
// see the design discussion this shipped from. Every non-WooCommerce
// provider below uses this until implemented for real.
type StubVerifier struct{}

func NewStubVerifier() *StubVerifier { return &StubVerifier{} }

func (StubVerifier) Verify(rawBody []byte, headers map[string]string, secret string) error {
	return webhooks.ErrProviderNotImplemented
}

func (StubVerifier) EventID(headers map[string]string, rawBody []byte) string { return "" }
