// Package tenantemail resolves the email client to use for a tenant: its
// own connected email integration (domain/integrations.ProviderEmail) when
// it has one, otherwise the platform-level SMTP client. Shared by every
// feature that emails on a tenant's behalf (Message Center automations,
// notification emails).
package tenantemail

import (
	"fmt"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

// Resolve picks the client from the tenant's connections. For
// ProviderEmail, StoreURL holds the SMTP host, ConsumerKey the username and
// ConsumerSecretEncrypted the password (see that constant's doc comment).
func Resolve(conns []integrationsDomain.Connection, encryptionSecret string, factory messaging.EmailClientFactory, fallback messaging.Client) (messaging.Client, error) {
	for i := range conns {
		if conns[i].Provider != integrationsDomain.ProviderEmail {
			continue
		}
		conn := conns[i]
		password, err := crypto.Decrypt(encryptionSecret, conn.ConsumerSecretEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypt email connection secret: %w", err)
		}
		port := ""
		if conn.SMTPPort != nil {
			port = *conn.SMTPPort
		}
		from := ""
		if conn.EmailFrom != nil {
			from = *conn.EmailFrom
		}
		return factory.NewClient(conn.StoreURL.String(), port, conn.ConsumerKey, password, from), nil
	}
	return fallback, nil
}
