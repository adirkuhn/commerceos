// Package tenantmail sends email as the current tenant: through its own
// email connection when it has one, otherwise the platform SMTP client
// (see application/tenantemail.Resolve).
package tenantmail

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/tenantemail"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// connectionLister is the tenant's integration connections.
type connectionLister interface {
	ListByTenant(ctx context.Context) ([]integrationsDomain.Connection, error)
}

// Mailer is a messaging.Client resolving the tenant's email client on
// every send. The lookup runs in its own tenant transaction; the send
// itself never does.
type Mailer struct {
	db       *gorm.DB
	conns    connectionLister
	secret   string
	factory  messaging.EmailClientFactory
	fallback messaging.Client
}

func New(db *gorm.DB, conns connectionLister, encryptionSecret string, factory messaging.EmailClientFactory, fallback messaging.Client) *Mailer {
	return &Mailer{db: db, conns: conns, secret: encryptionSecret, factory: factory, fallback: fallback}
}

var _ messaging.Client = (*Mailer)(nil)

func (m *Mailer) Send(ctx context.Context, msg messaging.Message) error {
	var client messaging.Client
	err := tenant.WithTenant(ctx, m.db, func(ctx context.Context) error {
		conns, err := m.conns.ListByTenant(ctx)
		if err != nil {
			return err
		}
		client, err = tenantemail.Resolve(conns, m.secret, m.factory, m.fallback)
		return err
	})
	if err != nil {
		return fmt.Errorf("tenant mailer: %w", err)
	}
	return client.Send(ctx, msg)
}
