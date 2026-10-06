package channels

import (
	"context"
	"errors"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type LinkExistingWooProductInput struct {
	ProductID    string
	ConnectionID string
	// SKU is the product's SKU as it exists on WooCommerce -- resolved to
	// the Woo product's own ID via WooClient.FindProductBySKU. Only ever
	// links the product-level row (a single-SKU/!HasVariants product);
	// linking one variant of a multi-variant product isn't supported here.
	SKU string
}

// LinkExistingWooProductUseCase is the manual escape hatch for a product
// whose product_channel_listings row was lost while the real product still
// exists on WooCommerce (see PushProductToWooUseCase's own automatic
// self-heal for the common case -- this is for whatever that doesn't
// catch, e.g. the SKU changed on one side, or staff already knows exactly
// which Woo product this is). Staff supply the SKU as it exists on Woo
// right now; this looks it up via WooClient and links to whatever it
// finds, instead of requiring a full re-import.
type LinkExistingWooProductUseCase struct {
	db          *gorm.DB
	productRepo pmsDomain.ProductRepository
	connRepo    integrationsDomain.ConnectionRepository
	listingRepo channelsDomain.ProductChannelListingRepository
	wooFactory  integrationsDomain.WooClientFactory
	encSecret   string
	outbox      eventrecord.Enqueuer
}

func NewLinkExistingWooProductUseCase(
	db *gorm.DB,
	productRepo pmsDomain.ProductRepository,
	connRepo integrationsDomain.ConnectionRepository,
	listingRepo channelsDomain.ProductChannelListingRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
	outbox eventrecord.Enqueuer,
) *LinkExistingWooProductUseCase {
	return &LinkExistingWooProductUseCase{
		db: db, productRepo: productRepo, connRepo: connRepo, listingRepo: listingRepo,
		wooFactory: wooFactory, encSecret: encryptionSecret,
		outbox: outbox,
	}
}

func (uc *LinkExistingWooProductUseCase) Execute(ctx context.Context, in LinkExistingWooProductInput) (*channelsDomain.ProductChannelListing, error) {
	var conn integrationsDomain.Connection
	var secret string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.productRepo.FindByID(ctx, in.ProductID); err != nil {
			return err
		}

		_, err := uc.listingRepo.FindByProductVariantAndConnection(ctx, in.ProductID, nil, in.ConnectionID)
		if err == nil {
			return channelsDomain.ErrProductAlreadyLinkedToConnection
		}
		if !errors.Is(err, channelsDomain.ErrListingNotFound) {
			return err
		}

		c, err := uc.connRepo.FindByID(ctx, in.ConnectionID)
		if err != nil {
			return err
		}
		conn = *c
		secret, err = crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("linkExistingWooProduct: %w", err)
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("linkExistingWooProduct: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}
	found, err := client.FindProductBySKU(ctx, in.SKU)
	if err != nil {
		return nil, fmt.Errorf("linkExistingWooProduct: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}
	if found == nil {
		return nil, fmt.Errorf("linkExistingWooProduct: %w", integrationsDomain.ErrWooProductNotFoundBySKU)
	}

	var listing channelsDomain.ProductChannelListing
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		listing = channelsDomain.ProductChannelListing{
			TenantID: tenant.IDFromContext(ctx), ProductID: in.ProductID, ConnectionID: in.ConnectionID,
			Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: found.ID,
			SyncStatus: channelsDomain.SyncStatusSynced, LastSyncedAt: &now,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.listingRepo.Create(ctx, &listing); err != nil {
			return err
		}
		return recordListingCreated(ctx, uc.outbox, listing, ListingSourceLinkExisting)
	})
	if err != nil {
		return nil, fmt.Errorf("linkExistingWooProduct: %w", err)
	}
	return &listing, nil
}
