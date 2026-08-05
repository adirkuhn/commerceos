package channels

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// stockLevelGetter is the one method of application/inventory's
// GetSKUStockLevelUseCase this use case needs (ISP). Called as a plain
// sequential operation between the two tenant.WithTenant blocks below, for
// the same reason stockRecorder is in import_woo_products.go: it opens its
// own transaction via the same shared *gorm.DB, so it can't be nested
// inside another use case's transaction closure.
type stockLevelGetter interface {
	Execute(ctx context.Context, skuID string) (*invapp.StockLevel, error)
}

type PushProductToWooInput struct {
	ProductID    string
	ConnectionID string
}

type PushProductToWooUseCase struct {
	db          *gorm.DB
	connRepo    integrationsDomain.ConnectionRepository
	wooFactory  integrationsDomain.WooClientFactory
	encSecret   string
	productRepo pmsDomain.ProductRepository
	variantRepo pmsDomain.VariantRepository
	skuRepo     pmsDomain.SKURepository
	listingRepo channelsDomain.ProductChannelListingRepository
	syncLogRepo channelsDomain.SyncLogRepository
	stockLevel  stockLevelGetter
}

func NewPushProductToWooUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
	productRepo pmsDomain.ProductRepository,
	variantRepo pmsDomain.VariantRepository,
	skuRepo pmsDomain.SKURepository,
	listingRepo channelsDomain.ProductChannelListingRepository,
	syncLogRepo channelsDomain.SyncLogRepository,
	stockLevel stockLevelGetter,
) *PushProductToWooUseCase {
	return &PushProductToWooUseCase{
		db: db, connRepo: connRepo, wooFactory: wooFactory, encSecret: encryptionSecret,
		productRepo: productRepo, variantRepo: variantRepo, skuRepo: skuRepo,
		listingRepo: listingRepo, syncLogRepo: syncLogRepo, stockLevel: stockLevel,
	}
}

// Execute pushes a single-SKU (has_variants=false) product to WooCommerce as
// a new "simple" product. Idempotent: if this product is already linked to
// this connection, returns the existing listing without pushing again.
func (uc *PushProductToWooUseCase) Execute(ctx context.Context, in PushProductToWooInput) (*channelsDomain.ProductChannelListing, error) {
	var conn integrationsDomain.Connection
	var secret string
	var product pmsDomain.Product
	var sku pmsDomain.SKU
	var existing *channelsDomain.ProductChannelListing

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if l, err := uc.listingRepo.FindByProductAndConnection(ctx, in.ProductID, in.ConnectionID); err == nil {
			existing = l
			return nil
		} else if !errors.Is(err, channelsDomain.ErrListingNotFound) {
			return err
		}

		p, err := uc.productRepo.FindByID(ctx, in.ProductID)
		if err != nil {
			return err
		}
		product = *p
		if product.HasVariants {
			return channelsDomain.ErrCannotPushVariableProduct
		}

		variants, err := uc.variantRepo.ListByProduct(ctx, in.ProductID)
		if err != nil {
			return err
		}
		if len(variants) != 1 {
			return channelsDomain.ErrCannotPushVariableProduct
		}

		skus, err := uc.skuRepo.ListByVariant(ctx, variants[0].ID)
		if err != nil {
			return err
		}
		if len(skus) != 1 {
			return channelsDomain.ErrCannotPushVariableProduct
		}
		sku = skus[0]

		c, err := uc.connRepo.FindByID(ctx, in.ConnectionID)
		if err != nil {
			return err
		}
		conn = *c
		secret, err = crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("pushProductToWoo: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("pushProductToWoo: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}

	salePrice, saleEndDate := WooSaleFields(sku)
	wooProduct := integrationsDomain.WooProduct{
		Name: product.Name, Description: product.Description,
		Status: MapCommerceStatusToWoo(product.Status),
		SKU:    sku.SKUCode, RegularPrice: FormatMinorUnitsAsWooPrice(sku.PriceMinorUnits),
		SalePrice: salePrice, SaleEndDate: saleEndDate,
		Virtual: sku.IsVirtual,
	}
	// Stock is only pushed for physical SKUs — a virtual SKU never has a
	// warehouse ledger entry to report. A ledger lookup failure (e.g. no
	// movements recorded yet) is not fatal to the push; the product just
	// goes up without stock management.
	if !sku.IsVirtual {
		if level, err := uc.stockLevel.Execute(ctx, sku.ID); err == nil && level != nil {
			qty := int(level.Syncable)
			wooProduct.ManageStock = true
			wooProduct.StockQuantity = &qty
		}
	}

	created, err := client.CreateProduct(ctx, wooProduct)
	if err != nil {
		return nil, fmt.Errorf("pushProductToWoo: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}

	var listing channelsDomain.ProductChannelListing
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		listing = channelsDomain.ProductChannelListing{
			TenantID: tenant.IDFromContext(ctx), ProductID: in.ProductID, ConnectionID: in.ConnectionID,
			Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: created.ID,
			SyncStatus: channelsDomain.SyncStatusSynced, LastSyncedAt: &now,
			CreatedAt: now, UpdatedAt: now,
		}
		return uc.listingRepo.Create(ctx, &listing)
	})
	if err != nil {
		return nil, fmt.Errorf("pushProductToWoo: %w", err)
	}

	recordSyncLog(ctx, uc.db, uc.syncLogRepo, listing, channelsDomain.SyncFlowPush,
		channelsDomain.SyncLogStatusSuccess, "", wooProductToFields(wooProduct))

	return &listing, nil
}
