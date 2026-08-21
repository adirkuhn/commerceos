package integrations

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ListWooCategoriesUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	wooFactory       integrationsDomain.WooClientFactory
	encryptionSecret string
}

func NewListWooCategoriesUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
) *ListWooCategoriesUseCase {
	return &ListWooCategoriesUseCase{db: db, connRepo: connRepo, wooFactory: wooFactory, encryptionSecret: encryptionSecret}
}

// Execute fetches the live category list from the connection's WooCommerce
// store, for the manual-mapping Settings UI. Read only -- nothing is
// persisted (see ResolveCategoryLinkUseCase for the auto-match-on-push path
// that does persist a link).
func (uc *ListWooCategoriesUseCase) Execute(ctx context.Context, connectionID string) ([]integrationsDomain.WooCategory, error) {
	var categories []integrationsDomain.WooCategory
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		conn, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		secret, err := crypto.Decrypt(uc.encryptionSecret, conn.ConsumerSecretEncrypted)
		if err != nil {
			return err
		}
		client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
		if err != nil {
			return err
		}
		categories, err = client.ListCategories(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listWooCategories: %w", err)
	}
	return categories, nil
}
