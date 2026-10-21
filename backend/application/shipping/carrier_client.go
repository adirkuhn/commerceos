package shipping

import (
	"fmt"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

// buildCarrierClient decrypts conn's API key and asks factory for the
// CarrierClient of conn's own provider -- the one step every shipping use
// case repeats before talking to a carrier.
func buildCarrierClient(factory integrationsDomain.CarrierClientFactory, encSecret string, conn integrationsDomain.Connection) (integrationsDomain.CarrierClient, error) {
	apiKey, err := crypto.Decrypt(encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt api key: %w", err)
	}
	client, err := factory.NewClient(integrationsDomain.CarrierCredentialsFor(conn, apiKey))
	if err != nil {
		return nil, fmt.Errorf("build client: %w", err)
	}
	return client, nil
}
