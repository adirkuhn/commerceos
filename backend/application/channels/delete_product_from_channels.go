package channels

import (
	"context"
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type DeleteProductFromChannelsUseCase struct {
	db         *gorm.DB
	connRepo   integrationsDomain.ConnectionRepository
	wooFactory integrationsDomain.WooClientFactory
	encSecret  string
}

func NewDeleteProductFromChannelsUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
) *DeleteProductFromChannelsUseCase {
	return &DeleteProductFromChannelsUseCase{db: db, connRepo: connRepo, wooFactory: wooFactory, encSecret: encryptionSecret}
}

// Execute removes a deleted product from every external channel it was
// synced to, given the listing snapshot captured before the local delete
// (see application/pms.DeleteProductUseCase -- the local
// product_channel_listings row is already gone by DB cascade before this
// ever runs). One listing's failure is logged and does not block the
// others -- same "log and continue" contract as
// EnrollProductInChannelsUseCase, since there's no listing left to retry
// against once the product is gone locally.
func (uc *DeleteProductFromChannelsUseCase) Execute(ctx context.Context, listings []events.ProductDeletedListing) error {
	for _, l := range listings {
		if l.Provider != string(integrationsDomain.ProviderWooCommerce) {
			continue // no adapter for any other provider yet
		}
		if err := uc.deleteOne(ctx, l); err != nil {
			log.Printf("deleteProductFromChannels: connection %s external %s: %v", l.ConnectionID, l.ExternalID, err)
		}
	}
	return nil
}

func (uc *DeleteProductFromChannelsUseCase) deleteOne(ctx context.Context, l events.ProductDeletedListing) error {
	var conn integrationsDomain.Connection
	var secret string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, l.ConnectionID)
		if err != nil {
			return err
		}
		conn = *c
		secret, err = crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
		return err
	})
	if err != nil {
		return fmt.Errorf("lookup connection: %w", err)
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return fmt.Errorf("%w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}
	if err := client.DeleteProduct(ctx, l.ExternalID); err != nil {
		return fmt.Errorf("%w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}
	return nil
}
