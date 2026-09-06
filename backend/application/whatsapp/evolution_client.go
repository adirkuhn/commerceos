package whatsapp

import (
	"fmt"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

// findWhatsAppConnection returns the tenant's whatsapp Connection, or nil if
// none exists -- a tenant can have at most one in practice today (nothing
// enforces it at the DB level, this just takes the first match), same
// simplification HandleInboundMessageUseCase's Connection lookup makes.
func findWhatsAppConnection(conns []integrationsDomain.Connection) *integrationsDomain.Connection {
	for i := range conns {
		if conns[i].Provider == integrationsDomain.ProviderWhatsApp {
			return &conns[i]
		}
	}
	return nil
}

// buildEvolutionClient decrypts conn's stored API key and builds a client
// scoped to it -- shared by every use case that sends an outbound WhatsApp
// message (HandleInboundMessageUseCase's replies, NotifyOrderStatusUseCase,
// NotifyShipmentDeliveryUseCase).
func buildEvolutionClient(factory integrationsDomain.EvolutionAPIClientFactory, encryptionSecret string, conn integrationsDomain.Connection) (integrationsDomain.EvolutionAPIClient, error) {
	apiKey, err := crypto.Decrypt(encryptionSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt connection secret: %w", err)
	}
	client, err := factory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, apiKey)
	if err != nil {
		return nil, fmt.Errorf("build evolution api client: %w", err)
	}
	return client, nil
}

// buildPaymentClient decrypts conn's stored API key and builds a payment
// client scoped to it -- same pattern as buildEvolutionClient, for
// GeneratePaymentLinkUseCase's Stripe/Revolut factories (both take just an
// API key, unlike Evolution API's base-url+instance+key).
func buildPaymentClient(factory integrationsDomain.PaymentLinkClientFactory, encryptionSecret string, conn integrationsDomain.Connection) (integrationsDomain.PaymentLinkClient, error) {
	apiKey, err := crypto.Decrypt(encryptionSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt connection secret: %w", err)
	}
	client, err := factory.NewClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("build payment client: %w", err)
	}
	return client, nil
}
