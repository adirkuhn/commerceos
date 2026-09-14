package messagecenter

import (
	"fmt"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

// findWhatsAppConnection returns the tenant's whatsapp Connection, or nil if
// none exists -- same simplification (at most one per tenant, first match
// wins) as application/whatsapp's own copy of this helper. Kept as a
// separate copy in this package rather than imported, since
// application/whatsapp doesn't export it and the two bounded contexts
// shouldn't depend on each other's application-layer internals.
func findWhatsAppConnection(conns []integrationsDomain.Connection) *integrationsDomain.Connection {
	for i := range conns {
		if conns[i].Provider == integrationsDomain.ProviderWhatsApp {
			return &conns[i]
		}
	}
	return nil
}

// buildEvolutionClient decrypts conn's stored API key and builds a client
// scoped to it -- same as application/whatsapp's own copy (see that
// package's evolution_client.go doc comment).
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
