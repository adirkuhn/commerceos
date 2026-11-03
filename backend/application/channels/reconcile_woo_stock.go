package channels

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// wooStockPageSize is how many products (or variations) one read asks for --
// WooCommerce's maximum per_page.
const wooStockPageSize = 100

// maxWooStockPages bounds the product read (100 x 1000 products) so a
// store answering every page forever can't hang the job.
const maxWooStockPages = 1000

// StockReconcileResult reports one reconciliation run.
type StockReconcileResult struct {
	Checked int // listings compared with what Woo reports
	Pushed  int // of which drifted and were pushed
	Missing int // listings whose Woo product no longer exists
	Errors  []string
}

// ReconcileWooStockUseCase is the stock_reconciliation schedule (COM-18):
// it reads what a WooCommerce store actually reports for every linked
// product and variation -- stock quantity, managed stock, the lot/expiry
// meta -- compares it with what CommerceOS would push under the stock
// rules (the SKU's syncable level and its source batch, the same values
// the event-driven push sends), and pushes only the listings that drifted.
// In steady state that is zero writes to the store. It also catches stock
// changed directly in Woo.
//
// A listing whose Woo product is gone is marked missing and raises
// channels.product_listing.orphaned (HandleMissingListingUseCase then
// re-creates it when its sales channel still allows it); one already
// missing is raised again each run, so a product enrolled later is picked
// up. Gated listings (product not active) and virtual SKUs are skipped.
// One listing failing never stops the rest.
type ReconcileWooStockUseCase struct {
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

func NewReconcileWooStockUseCase(
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
) *ReconcileWooStockUseCase {
	return &ReconcileWooStockUseCase{
		db: db, connRepo: connRepo, wooFactory: wooFactory, encSecret: encryptionSecret,
		variantRepo: variantRepo, skuRepo: skuRepo, listingRepo: listingRepo, syncLogRepo: syncLogRepo,
		stockLevel: stockLevel, outbox: outbox,
	}
}

// wooStock is what the store reports: products by id, and each variable
// product's variations by id (absent when that product's variations could
// not be read).
type wooStock struct {
	products   map[string]integrationsDomain.WooStockSnapshot
	variations map[string]map[string]integrationsDomain.WooStockSnapshot
}

func (uc *ReconcileWooStockUseCase) Execute(ctx context.Context, connectionID string) (*StockReconcileResult, error) {
	var conn integrationsDomain.Connection
	var secret string
	var listings []channelsDomain.ProductChannelListing
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		conn = *c
		if secret, err = crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted); err != nil {
			return err
		}
		listings, err = uc.listingRepo.ListByConnection(ctx, connectionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("reconcileWooStock: %w", err)
	}
	result := &StockReconcileResult{}
	if len(listings) == 0 {
		return result, nil
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("reconcileWooStock: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}
	store, err := readWooStock(ctx, client, listings, result)
	if err != nil {
		return nil, fmt.Errorf("reconcileWooStock: read store: %w", err)
	}

	for _, l := range listings {
		uc.reconcileOne(ctx, client, listings, l, store, result)
	}
	return result, nil
}

// readWooStock pages through every product, then the variations of each
// variable product that has variation listings.
func readWooStock(ctx context.Context, client integrationsDomain.WooClient, listings []channelsDomain.ProductChannelListing, result *StockReconcileResult) (wooStock, error) {
	store := wooStock{products: map[string]integrationsDomain.WooStockSnapshot{}, variations: map[string]map[string]integrationsDomain.WooStockSnapshot{}}
	for page := 1; page <= maxWooStockPages; page++ {
		snaps, err := client.ListProductStock(ctx, page, wooStockPageSize)
		if err != nil {
			return store, err
		}
		if len(snaps) == 0 {
			break
		}
		for _, s := range snaps {
			store.products[s.ID] = s
		}
	}
	for _, parentID := range variationParents(listings) {
		if _, ok := store.products[parentID]; !ok {
			continue // the parent itself is gone: its listings are missing
		}
		vars := map[string]integrationsDomain.WooStockSnapshot{}
		for page := 1; page <= maxWooStockPages; page++ {
			snaps, err := client.ListVariationStock(ctx, parentID, page, wooStockPageSize)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("variations of woo product %s: %v", parentID, err))
				vars = nil
				break
			}
			if len(snaps) == 0 {
				break
			}
			for _, s := range snaps {
				vars[s.ID] = s
			}
		}
		if vars != nil {
			store.variations[parentID] = vars
		}
	}
	return store, nil
}

// variationParents lists the Woo ids of the parent products of every
// variation listing, once each.
func variationParents(listings []channelsDomain.ProductChannelListing) []string {
	seen := map[string]bool{}
	var out []string
	for _, l := range listings {
		if !l.IsVariation {
			continue
		}
		if p := parentExternalIDFor(listings, l); p != "" && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

func (uc *ReconcileWooStockUseCase) reconcileOne(ctx context.Context, client integrationsDomain.WooClient, listings []channelsDomain.ProductChannelListing,
	l channelsDomain.ProductChannelListing, store wooStock, result *StockReconcileResult) {
	if l.GatedAt != nil || isVariationParent(listings, l) {
		return
	}
	if l.SyncStatus == channelsDomain.SyncStatusMissing {
		result.Missing++
		uc.raiseOrphaned(ctx, l)
		return
	}
	parentID := parentExternalIDFor(listings, l)
	have, found, known := lookupWooStock(store, l, parentID)
	if !known {
		return // its parent's variations could not be read: already reported
	}
	if !found {
		result.Missing++
		uc.markMissing(ctx, l)
		return
	}

	sku, err := uc.listingSKU(ctx, l)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("listing %s: %v", l.ID, err))
		return
	}
	if sku.IsVirtual {
		return
	}
	level, err := uc.stockLevel.Execute(ctx, sku.ID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("sku %s: stock level: %v", sku.ID, err))
		return
	}
	result.Checked++
	metadata := toWooMetaData(level.SourceBatch)
	if !stockDrifted(int(level.Syncable), metadata, have) {
		return
	}

	fields := map[string]string{"stock_quantity": strconv.FormatInt(level.Syncable, 10)}
	for _, md := range metadata {
		fields[md.Key] = md.Value
	}
	pushErr := client.UpdateProductStock(ctx, parentID, l.ExternalID, int(level.Syncable), metadata)
	uc.recordPush(ctx, l, pushErr, fields)
	if pushErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("listing %s: %v", l.ID, pushErr))
		return
	}
	result.Pushed++
}

// lookupWooStock finds a listing's product or variation in what the store
// reported. known is false when its parent's variations could not be read.
func lookupWooStock(store wooStock, l channelsDomain.ProductChannelListing, parentID string) (snap integrationsDomain.WooStockSnapshot, found, known bool) {
	if !l.IsVariation {
		snap, found = store.products[l.ExternalID]
		return snap, found, true
	}
	if _, parentFound := store.products[parentID]; !parentFound {
		return snap, false, true
	}
	vars, ok := store.variations[parentID]
	if !ok {
		return snap, false, false
	}
	snap, found = vars[l.ExternalID]
	return snap, found, true
}

// isVariationParent: the product-level row of a variations-mode listing
// set -- the variable product container, no stock of its own.
func isVariationParent(listings []channelsDomain.ProductChannelListing, l channelsDomain.ProductChannelListing) bool {
	if l.VariantID != nil {
		return false
	}
	for _, other := range listings {
		if other.ProductID == l.ProductID && other.ConnectionID == l.ConnectionID && other.IsVariation {
			return true
		}
	}
	return false
}

// stockDrifted: Woo does not manage the stock, reports another quantity,
// or another value for any batch meta field CommerceOS sends (an absent
// field counts as empty).
func stockDrifted(quantity int, metadata []integrationsDomain.WooMetaData, have integrationsDomain.WooStockSnapshot) bool {
	if !have.ManageStock || have.StockQuantity == nil || *have.StockQuantity != quantity {
		return true
	}
	for _, md := range metadata {
		if have.Meta[md.Key] != md.Value {
			return true
		}
	}
	return false
}

// listingSKU is the sellable SKU a listing stands for: its variant's, or
// for a whole-product listing, the product's only variant's.
func (uc *ReconcileWooStockUseCase) listingSKU(ctx context.Context, l channelsDomain.ProductChannelListing) (pmsDomain.SKU, error) {
	var sku pmsDomain.SKU
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		variantID := ""
		if l.VariantID != nil {
			variantID = *l.VariantID
		} else {
			variants, err := uc.variantRepo.ListByProduct(ctx, l.ProductID)
			if err != nil {
				return err
			}
			if len(variants) != 1 {
				return channelsDomain.ErrCannotPushVariableProduct
			}
			variantID = variants[0].ID
		}
		skus, err := uc.skuRepo.ListByVariant(ctx, variantID)
		if err != nil {
			return err
		}
		for _, s := range skus {
			if s.UnitType != pmsDomain.SKUUnitTypeKilo { // a bulk pool is never sold itself
				sku = s
				return nil
			}
		}
		return fmt.Errorf("no sellable SKU for variant %s", variantID)
	})
	return sku, err
}

// recordPush writes the push's sync log row and the listing's new state;
// a product gone from Woo is marked missing and raised once.
func (uc *ReconcileWooStockUseCase) recordPush(ctx context.Context, l channelsDomain.ProductChannelListing, pushErr error, fields map[string]string) {
	newlyMissing := applyPushOutcome(&l, pushErr, time.Now())
	status, errMsg := channelsDomain.SyncLogStatusSuccess, ""
	if pushErr != nil {
		status, errMsg = channelsDomain.SyncLogStatusError, pushErr.Error()
	}
	recordSyncLog(ctx, uc.db, uc.syncLogRepo, uc.outbox, l, channelsDomain.SyncFlowPush, status, errMsg, fields)
	uc.saveListing(ctx, l, newlyMissing)
}

// markMissing: the store no longer lists the listing's product.
func (uc *ReconcileWooStockUseCase) markMissing(ctx context.Context, l channelsDomain.ProductChannelListing) {
	newlyMissing := applyPushOutcome(&l, fmt.Errorf("%w: not in the store's product list", integrationsDomain.ErrWooProductNotFound), time.Now())
	uc.saveListing(ctx, l, newlyMissing)
}

func (uc *ReconcileWooStockUseCase) saveListing(ctx context.Context, l channelsDomain.ProductChannelListing, raiseOrphaned bool) {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.listingRepo.Update(ctx, &l); err != nil {
			return err
		}
		if raiseOrphaned {
			return recordListingOrphaned(ctx, uc.outbox, l)
		}
		return nil
	})
	if err != nil {
		logReconcile(l, err)
	}
}

func (uc *ReconcileWooStockUseCase) raiseOrphaned(ctx context.Context, l channelsDomain.ProductChannelListing) {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return recordListingOrphaned(ctx, uc.outbox, l)
	})
	if err != nil {
		logReconcile(l, err)
	}
}

func logReconcile(l channelsDomain.ProductChannelListing, err error) {
	log.Printf("reconcileWooStock: listing %s: %v", l.ID, err)
}
