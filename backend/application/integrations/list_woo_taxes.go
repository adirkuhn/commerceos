package integrations

import (
	"context"
	"fmt"
	"sort"

	"gorm.io/gorm"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// WooTaxClassOption is one distinct tax class configured in a WooCommerce
// store -- Slug is the raw value WooCommerce uses (what ends up as a
// product's tax_class / an ExternalTaxClassLink.ExternalSlug), Label is a
// human-readable name for display.
type WooTaxClassOption struct {
	Slug  string
	Label string
}

// DedupeWooTaxClasses collapses a store's tax rate table (one row per
// country/state) down to the distinct tax classes it defines, sorted with
// Standard first and the rest alphabetically by label.
func DedupeWooTaxClasses(rates []integrationsDomain.WooTaxRate) []WooTaxClassOption {
	seen := make(map[string]bool)
	options := make([]WooTaxClassOption, 0, len(rates))
	for _, r := range rates {
		if seen[r.Class] {
			continue
		}
		seen[r.Class] = true
		options = append(options, WooTaxClassOption{Slug: r.Class, Label: appchannels.HumanizeWooTaxClassSlug(r.Class)})
	}
	sort.Slice(options, func(i, j int) bool {
		iStd := options[i].Label == "Standard"
		jStd := options[j].Label == "Standard"
		if iStd != jStd {
			return iStd
		}
		return options[i].Label < options[j].Label
	})
	return options
}

type ListWooTaxesUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	wooFactory       integrationsDomain.WooClientFactory
	encryptionSecret string
}

func NewListWooTaxesUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
) *ListWooTaxesUseCase {
	return &ListWooTaxesUseCase{db: db, connRepo: connRepo, wooFactory: wooFactory, encryptionSecret: encryptionSecret}
}

// Execute fetches the live tax rate table from the connection's WooCommerce
// store and reduces it to the distinct tax classes configured there. Read
// only -- nothing is persisted (see ImportWooProductsUseCase for the
// reconciliation that creates TaxClass/ExternalTaxClassLink rows).
func (uc *ListWooTaxesUseCase) Execute(ctx context.Context, connectionID string) ([]WooTaxClassOption, error) {
	var options []WooTaxClassOption
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
		rates, err := client.ListTaxRates(ctx)
		if err != nil {
			return err
		}
		options = DedupeWooTaxClasses(rates)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listWooTaxes: %w", err)
	}
	return options, nil
}
