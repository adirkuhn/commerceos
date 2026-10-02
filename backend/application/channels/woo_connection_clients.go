package channels

import (
	"context"
	"fmt"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// WooConnectionClients builds the WooCommerce API client for a stored
// connection: loads it, insists it is a WooCommerce connection, decrypts
// its consumer secret. Implements wooClientResolver.
type WooConnectionClients struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	wooFactory       integrationsDomain.WooClientFactory
	encryptionSecret string
}

func NewWooConnectionClients(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, wooFactory integrationsDomain.WooClientFactory, encryptionSecret string) *WooConnectionClients {
	return &WooConnectionClients{db: db, connRepo: connRepo, wooFactory: wooFactory, encryptionSecret: encryptionSecret}
}

func (w *WooConnectionClients) ClientFor(ctx context.Context, connectionID string) (integrationsDomain.WooClient, error) {
	var conn integrationsDomain.Connection
	var secret string
	err := tenant.WithTenant(ctx, w.db, func(ctx context.Context) error {
		existing, err := w.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		if existing.Provider != integrationsDomain.ProviderWooCommerce {
			return integrationsDomain.ErrProviderNotSupported
		}
		conn = *existing
		secret, err = crypto.Decrypt(w.encryptionSecret, conn.ConsumerSecretEncrypted)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("wooConnectionClients: %w", err)
	}
	client, err := w.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("wooConnectionClients: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}
	return client, nil
}
