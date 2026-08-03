package channels

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

const maxStockSyncPages = 100 // same safety cap as import_woo_products.go

// StockSyncResult reports the outcome of a stock reconciliation run (either
// direction) against every existing ProductChannelListing for a connection.
type StockSyncResult struct {
	Updated   int
	Unchanged int
	Errors    []string
}

type ImportWooStockLevelsInput struct {
	ConnectionID       string
	DefaultWarehouseID string
}

// ImportWooStockLevelsUseCase pulls WooCommerce's current stock_quantity for
// every product already linked to this connection and reconciles
// CommerceOS's ledger to match it via an adjustment movement. Deliberately
// does not import new products (that's ImportWooProductsUseCase's job) —
// this only touches SKUs that already have a listing.
type ImportWooStockLevelsUseCase struct {
	db          *gorm.DB
	connRepo    integrationsDomain.ConnectionRepository
	wooFactory  integrationsDomain.WooClientFactory
	encSecret   string
	variantRepo pmsDomain.VariantRepository
	skuRepo     pmsDomain.SKURepository
	listingRepo channelsDomain.ProductChannelListingRepository
	stockLevel  stockLevelGetter
	stock       stockRecorder
}

func NewImportWooStockLevelsUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
	variantRepo pmsDomain.VariantRepository,
	skuRepo pmsDomain.SKURepository,
	listingRepo channelsDomain.ProductChannelListingRepository,
	stockLevel stockLevelGetter,
	stock stockRecorder,
) *ImportWooStockLevelsUseCase {
	return &ImportWooStockLevelsUseCase{
		db: db, connRepo: connRepo, wooFactory: wooFactory, encSecret: encryptionSecret,
		variantRepo: variantRepo, skuRepo: skuRepo, listingRepo: listingRepo,
		stockLevel: stockLevel, stock: stock,
	}
}

func (uc *ImportWooStockLevelsUseCase) Execute(ctx context.Context, in ImportWooStockLevelsInput) (*StockSyncResult, error) {
	var conn integrationsDomain.Connection
	var secret string
	var listings []channelsDomain.ProductChannelListing
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, in.ConnectionID)
		if err != nil {
			return err
		}
		conn = *c
		secret, err = crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
		if err != nil {
			return err
		}
		listings, err = uc.listingRepo.ListByConnection(ctx, in.ConnectionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("importWooStockLevels: %w", err)
	}
	if len(listings) == 0 {
		return &StockSyncResult{}, nil
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("importWooStockLevels: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}

	wooStockByExternalID := map[string]int{}
	for page := 1; page <= maxStockSyncPages; page++ {
		products, err := client.ListProducts(ctx, page, 50)
		if err != nil {
			return nil, fmt.Errorf("importWooStockLevels: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
		}
		if len(products) == 0 {
			break
		}
		for _, wp := range products {
			if wp.ManageStock && wp.StockQuantity != nil {
				wooStockByExternalID[wp.ID] = *wp.StockQuantity
			}
		}
	}

	result := &StockSyncResult{}
	for _, listing := range listings {
		wooQty, ok := wooStockByExternalID[listing.ExternalID]
		if !ok {
			continue // Woo isn't managing stock for this product, nothing to reconcile
		}
		uc.reconcileOne(ctx, in, listing, wooQty, result)
	}
	return result, nil
}

// reconcileOne resolves the listing's single SKU and, if CommerceOS's
// current syncable stock differs from Woo's reported quantity, records an
// adjustment movement to match it. Failures on one listing are recorded on
// the result and never abort the rest of the run.
func (uc *ImportWooStockLevelsUseCase) reconcileOne(ctx context.Context, in ImportWooStockLevelsInput, listing channelsDomain.ProductChannelListing, wooQty int, result *StockSyncResult) {
	var skuID string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		variants, err := uc.variantRepo.ListByProduct(ctx, listing.ProductID)
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
		skuID = skus[0].ID
		return nil
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("product %s: %v", listing.ProductID, err))
		return
	}

	level, err := uc.stockLevel.Execute(ctx, skuID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("sku %s: stock level: %v", skuID, err))
		return
	}
	delta := int64(wooQty) - level.Syncable
	if delta == 0 {
		result.Unchanged++
		return
	}
	_, err = uc.stock.Execute(ctx, invapp.RecordStockMovementInput{
		SKUID: skuID, WarehouseID: in.DefaultWarehouseID,
		Type: invDomain.MovementTypeAdjustment, QuantityDelta: delta,
		Reason: "woocommerce stock import", OccurredAt: time.Now(),
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("sku %s: record adjustment: %v", skuID, err))
		return
	}
	result.Updated++
}
