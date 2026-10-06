package channels

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"strconv"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type ExportWooStockLevelsInput struct {
	ConnectionID string
}

// ExportWooStockLevelsUseCase pushes CommerceOS's current syncable stock
// level for every existing ProductChannelListing under a connection to
// WooCommerce. Unlike the import direction, this doesn't need a
// before/after comparison — it just pushes the current truth on every run.
type ExportWooStockLevelsUseCase struct {
	db          *gorm.DB
	connRepo    integrationsDomain.ConnectionRepository
	wooFactory  integrationsDomain.WooClientFactory
	encSecret   string
	variantRepo pmsDomain.VariantRepository
	skuRepo     pmsDomain.SKURepository
	listingRepo channelsDomain.ProductChannelListingRepository
	syncLogRepo channelsDomain.SyncLogRepository
	stockLevel  stockLevelGetter
	outbox      eventrecord.Enqueuer
}

func NewExportWooStockLevelsUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
	variantRepo pmsDomain.VariantRepository,
	skuRepo pmsDomain.SKURepository,
	listingRepo channelsDomain.ProductChannelListingRepository,
	syncLogRepo channelsDomain.SyncLogRepository,
	stockLevel stockLevelGetter,
	outbox eventrecord.Enqueuer,
) *ExportWooStockLevelsUseCase {
	return &ExportWooStockLevelsUseCase{
		db: db, connRepo: connRepo, wooFactory: wooFactory, encSecret: encryptionSecret,
		variantRepo: variantRepo, skuRepo: skuRepo, listingRepo: listingRepo, syncLogRepo: syncLogRepo, stockLevel: stockLevel,
		outbox: outbox,
	}
}

func (uc *ExportWooStockLevelsUseCase) Execute(ctx context.Context, in ExportWooStockLevelsInput) (*StockSyncResult, error) {
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
		return nil, fmt.Errorf("exportWooStockLevels: %w", err)
	}
	if len(listings) == 0 {
		return &StockSyncResult{}, nil
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("exportWooStockLevels: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}

	result := &StockSyncResult{}
	for _, listing := range listings {
		uc.pushOne(ctx, client, listings, listing, result)
	}
	return result, nil
}

// hasVariantSibling reports whether listing is the product-level "parent"
// row of a variations-mode listing set (VariantID nil, but a sibling row
// for the same product+connection has VariantID set) -- see
// channelsDomain.ProductChannelListing's own doc comment for the three
// shapes a product's listing set can take. The parent row represents the
// variable product container itself, never a SKU, so it has no stock of
// its own to push.
func hasVariantSibling(listings []channelsDomain.ProductChannelListing, listing channelsDomain.ProductChannelListing) bool {
	for _, l := range listings {
		if l.ProductID == listing.ProductID && l.ConnectionID == listing.ConnectionID && l.VariantID != nil {
			return true
		}
	}
	return false
}

// pushOne resolves the listing's single SKU and pushes its current
// syncable stock level to Woo. Virtual SKUs are skipped (never have a
// ledger to report), and so is a variations-mode parent row (see
// hasVariantSibling). Failures on one listing are recorded on the result
// and never abort the rest of the run.
//
// listing.VariantID, when set, names the SKU's variant directly -- no need
// to (incorrectly) re-derive it from the whole product, which breaks for
// any real multi-variant product (see the 2026-09-11/12 incident this
// fixed: every push for a product with more than one variant failed with
// ErrCannotPushVariableProduct even though each variant, individually,
// already had exactly one SKU). Only a legacy nil-VariantID listing for a
// genuinely single-SKU product still needs that whole-product fallback.
// Mirrors sync_stock_to_channels.go's resolveListingSKUID.
func (uc *ExportWooStockLevelsUseCase) pushOne(ctx context.Context, client integrationsDomain.WooClient, listings []channelsDomain.ProductChannelListing, listing channelsDomain.ProductChannelListing, result *StockSyncResult) {
	if listing.VariantID == nil && hasVariantSibling(listings, listing) {
		return
	}

	var skuID string
	var isVirtual bool
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if listing.VariantID != nil {
			skus, err := uc.skuRepo.ListByVariant(ctx, *listing.VariantID)
			if err != nil {
				return err
			}
			if len(skus) != 1 {
				return channelsDomain.ErrCannotPushVariableProduct
			}
			skuID = skus[0].ID
			isVirtual = skus[0].IsVirtual
			return nil
		}

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
		isVirtual = skus[0].IsVirtual
		return nil
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("product %s: %v", listing.ProductID, err))
		recordSyncLog(ctx, uc.db, uc.syncLogRepo, uc.outbox, listing, channelsDomain.SyncFlowPush, channelsDomain.SyncLogStatusError, err.Error(), nil)
		return
	}
	if isVirtual {
		return
	}

	level, err := uc.stockLevel.Execute(ctx, skuID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("sku %s: stock level: %v", skuID, err))
		recordSyncLog(ctx, uc.db, uc.syncLogRepo, uc.outbox, listing, channelsDomain.SyncFlowPush, channelsDomain.SyncLogStatusError, err.Error(), nil)
		return
	}

	fields := map[string]string{"stock_quantity": strconv.FormatInt(level.Syncable, 10)}
	metadata := toWooMetaData(level.SourceBatch)
	for _, md := range metadata {
		fields[md.Key] = md.Value
	}
	if err := client.UpdateProductStock(ctx, parentExternalIDFor(listings, listing), listing.ExternalID, int(level.Syncable), metadata); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("sku %s: %v: %v", skuID, integrationsDomain.ErrWooRequestFailed, err))
		recordSyncLog(ctx, uc.db, uc.syncLogRepo, uc.outbox, listing, channelsDomain.SyncFlowPush, channelsDomain.SyncLogStatusError, err.Error(), fields)
		return
	}
	result.Updated++
	recordSyncLog(ctx, uc.db, uc.syncLogRepo, uc.outbox, listing, channelsDomain.SyncFlowPush, channelsDomain.SyncLogStatusSuccess, "", fields)
}
