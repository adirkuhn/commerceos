package channels

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// bolClientSource builds a BolClient for one Bol connection -- narrowed to
// point of use so use cases can be tested with a fake.
type bolClientSource interface {
	ClientFor(ctx context.Context, connectionID string) (integrationsDomain.BolClient, error)
}

// BolConnectionClients builds a BolClient from a stored Bol connection's
// decrypted credentials -- the Bol counterpart of WooConnectionClients.
type BolConnectionClients struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	bolFactory       integrationsDomain.BolClientFactory
	encryptionSecret string
}

func NewBolConnectionClients(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, bolFactory integrationsDomain.BolClientFactory, encryptionSecret string) *BolConnectionClients {
	return &BolConnectionClients{db: db, connRepo: connRepo, bolFactory: bolFactory, encryptionSecret: encryptionSecret}
}

func (b *BolConnectionClients) ClientFor(ctx context.Context, connectionID string) (integrationsDomain.BolClient, error) {
	var conn integrationsDomain.Connection
	var secret string
	err := tenant.WithTenant(ctx, b.db, func(ctx context.Context) error {
		existing, err := b.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		if existing.Provider != integrationsDomain.ProviderBol {
			return integrationsDomain.ErrProviderNotSupported
		}
		conn = *existing
		secret, err = crypto.Decrypt(b.encryptionSecret, conn.ConsumerSecretEncrypted)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("bolConnectionClients: %w", err)
	}
	client, err := b.bolFactory.NewClient(conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("bolConnectionClients: build client: %w", err)
	}
	return client, nil
}
