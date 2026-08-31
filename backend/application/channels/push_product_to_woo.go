package channels

import (
	"context"
	"errors"
	"fmt"
	"log"
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

// variantPush is one Variant paired with its single sellable SKU — the
// unit of work for every push mode below.
type variantPush struct {
	Variant pmsDomain.Variant
	SKU     pmsDomain.SKU
}

type PushProductToWooUseCase struct {
	db           *gorm.DB
	connRepo     integrationsDomain.ConnectionRepository
	wooFactory   integrationsDomain.WooClientFactory
	encSecret    string
	productRepo  pmsDomain.ProductRepository
	variantRepo  pmsDomain.VariantRepository
	skuRepo      pmsDomain.SKURepository
	listingRepo  channelsDomain.ProductChannelListingRepository
	syncLogRepo  channelsDomain.SyncLogRepository
	stockLevel   stockLevelGetter
	mediaRepo    productMediaLister
	categoryLink categoryLinkResolver
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
	mediaRepo productMediaLister,
	categoryLink categoryLinkResolver,
) *PushProductToWooUseCase {
	return &PushProductToWooUseCase{
		db: db, connRepo: connRepo, wooFactory: wooFactory, encSecret: encryptionSecret,
		productRepo: productRepo, variantRepo: variantRepo, skuRepo: skuRepo,
		listingRepo: listingRepo, syncLogRepo: syncLogRepo, stockLevel: stockLevel,
		mediaRepo: mediaRepo, categoryLink: categoryLink,
	}
}

// Execute pushes a product to WooCommerce, in one of three shapes:
//
//   - !HasVariants: a single "simple" product (the one Variant/SKU pair).
//   - HasVariants && !SyncVariantsAsWooVariations: one standalone simple
//     product per Variant ("separate-products" mode).
//   - HasVariants && SyncVariantsAsWooVariations: one "variable" parent
//     product plus one Woo variation per Variant ("variations" mode).
//
// Idempotent per row: a Variant (or the whole product, in the single-SKU
// case) already linked to this connection is left alone — its existing
// listing is returned instead of being pushed again. Mode migration for an
// already-linked product (the flag flipping after the initial push) is
// SyncProductToChannelsUseCase's job, not this one's.
func (uc *PushProductToWooUseCase) Execute(ctx context.Context, in PushProductToWooInput) ([]channelsDomain.ProductChannelListing, error) {
	var conn integrationsDomain.Connection
	var secret string
	var product pmsDomain.Product
	var pushes []variantPush
	var existingParent *channelsDomain.ProductChannelListing
	var media []pmsDomain.Media
	existingByVariant := map[string]channelsDomain.ProductChannelListing{}

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		p, err := uc.productRepo.FindByID(ctx, in.ProductID)
		if err != nil {
			return err
		}
		product = *p

		media, err = uc.mediaRepo.ListByProduct(ctx, in.ProductID)
		if err != nil {
			return err
		}

		variants, err := uc.variantRepo.ListByProduct(ctx, in.ProductID)
		if err != nil {
			return err
		}
		if len(variants) == 0 || (!product.HasVariants && len(variants) != 1) {
			return channelsDomain.ErrCannotPushVariableProduct
		}

		for _, v := range variants {
			skus, err := uc.skuRepo.ListByVariant(ctx, v.ID)
			if err != nil {
				return err
			}
			if len(skus) != 1 {
				return channelsDomain.ErrCannotPushVariableProduct
			}
			pushes = append(pushes, variantPush{Variant: v, SKU: skus[0]})

			l, err := uc.listingRepo.FindByProductVariantAndConnection(ctx, in.ProductID, rowVariantID(product, v.ID), in.ConnectionID)
			if err == nil {
				existingByVariant[v.ID] = *l
			} else if !errors.Is(err, channelsDomain.ErrListingNotFound) {
				return err
			}
		}

		// A bulk pool (UnitType=kilo) SKU is an internal weight ledger,
		// never itself sellable -- excluded from what gets pushed. See
		// pms.SKU.PortionedFromSKUID for the sellable items peeled off it.
		sellable := pushes[:0]
		for _, p := range pushes {
			if p.SKU.UnitType != pmsDomain.SKUUnitTypeKilo {
				sellable = append(sellable, p)
			}
		}
		pushes = sellable
		if len(pushes) == 0 {
			return channelsDomain.ErrCannotPushBulkPoolSKU
		}

		// A product that used to be flat (!HasVariants) keeps its one
		// variant's ID unchanged when flipped into variants mode --
		// UpdateProductUseCase relinks that variant's pre-existing
		// product-level listing (VariantID nil) to the variant's own ID
		// the moment the flag flips, but a product that already flipped
		// before that fix existed is stuck with the old nil-VariantID row
		// forever otherwise. Catching it here too, on every push attempt,
		// self-heals those regardless of when the flip happened -- and
		// unlike createProductSelfHealingDuplicateSKU below, needs no Woo
		// API call at all, so it isn't at the mercy of Woo's own search
		// (e.g. a stale/cached GET result) failing to find what the
		// create rejection just proved exists. Only unambiguous for
		// separate-products mode (or the trivial !HasVariants case, which
		// already matches directly above) with exactly one sellable
		// variant -- SyncVariantsAsWooVariations' own nil-VariantID row
		// means something different (the variable parent, see
		// existingParent below), and more than one variant here means
		// there's no single "the" variant the orphaned row could
		// unambiguously belong to.
		if product.HasVariants && !product.SyncVariantsAsWooVariations && len(pushes) == 1 {
			vp := pushes[0]
			if _, ok := existingByVariant[vp.Variant.ID]; !ok {
				legacy, err := uc.listingRepo.FindByProductVariantAndConnection(ctx, in.ProductID, nil, in.ConnectionID)
				if err == nil {
					variantID := vp.Variant.ID
					legacy.VariantID = &variantID
					if err := uc.listingRepo.Update(ctx, legacy); err != nil {
						return err
					}
					existingByVariant[vp.Variant.ID] = *legacy
				} else if !errors.Is(err, channelsDomain.ErrListingNotFound) {
					return err
				}
			}
		}

		if product.HasVariants && product.SyncVariantsAsWooVariations {
			l, err := uc.listingRepo.FindByProductVariantAndConnection(ctx, in.ProductID, nil, in.ConnectionID)
			if err == nil {
				existingParent = l
			} else if !errors.Is(err, channelsDomain.ErrListingNotFound) {
				return err
			}
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
		return nil, fmt.Errorf("pushProductToWoo: %w", err)
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("pushProductToWoo: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}
	images := wooImagesFromMedia(media, product.Name)

	if !product.HasVariants {
		if l, ok := existingByVariant[pushes[0].Variant.ID]; ok {
			return []channelsDomain.ProductChannelListing{l}, nil
		}
		return uc.pushSingleSKU(ctx, client, product, pushes[0], in, images)
	}
	if product.SyncVariantsAsWooVariations {
		return uc.pushVariations(ctx, client, product, pushes, existingParent, existingByVariant, in, images)
	}
	return uc.pushSeparateProducts(ctx, client, product, pushes, existingByVariant, in, images)
}

// rowVariantID is the VariantID a listing row for this variant should
// carry: nil for a single-SKU (!HasVariants) product's one product-level
// row, the Variant's own ID for every per-variant row (both
// separate-products and variations modes).
func rowVariantID(product pmsDomain.Product, variantID string) *string {
	if !product.HasVariants {
		return nil
	}
	id := variantID
	return &id
}

// createProductSelfHealingDuplicateSKU wraps client.CreateProduct --
// WooCommerce rejects a create when a product with this SKU already
// exists on the store (its own SKU-uniqueness constraint); in production
// that happens when this codebase's own link to an already-existing Woo
// product was lost (e.g. a connection disconnect/reconnect orphaned the
// product_channel_listings row, see integrationsDomain.ErrWooDuplicateSKU's
// doc), not because the product is actually new. Rather than failing this
// push forever on every retry, look the existing product up by SKU and
// treat it as if that had just been created -- re-links CommerceOS to the
// product Woo already has instead of endlessly retrying a create Woo will
// always reject. Only wired into the two CreateProduct call sites
// (single-SKU and separate-products modes) that actually hit this failure
// in practice; CreateVariation (variations mode) isn't covered.
func (uc *PushProductToWooUseCase) createProductSelfHealingDuplicateSKU(ctx context.Context, client integrationsDomain.WooClient, wooProduct integrationsDomain.WooProduct) (*integrationsDomain.WooProduct, error) {
	created, err := client.CreateProduct(ctx, wooProduct)
	if err == nil {
		return created, nil
	}
	if !errors.Is(err, integrationsDomain.ErrWooDuplicateSKU) {
		return nil, err
	}
	found, findErr := client.FindProductBySKU(ctx, wooProduct.SKU)
	if findErr != nil {
		log.Printf("createProductSelfHealingDuplicateSKU: FindProductBySKU(%q) errored, self-heal miss: %v", wooProduct.SKU, findErr)
		return nil, err // self-heal didn't pan out -- surface the original error
	}
	if found == nil {
		log.Printf("createProductSelfHealingDuplicateSKU: FindProductBySKU(%q) found nothing, self-heal miss", wooProduct.SKU)
		return nil, err // self-heal didn't pan out -- surface the original error
	}
	log.Printf("createProductSelfHealingDuplicateSKU: FindProductBySKU(%q) found existing Woo product %s, self-healing", wooProduct.SKU, found.ID)
	return found, nil
}

func (uc *PushProductToWooUseCase) pushSingleSKU(ctx context.Context, client integrationsDomain.WooClient, product pmsDomain.Product, vp variantPush, in PushProductToWooInput, images []integrationsDomain.WooProductImage) ([]channelsDomain.ProductChannelListing, error) {
	wooProduct := uc.buildWooProduct(ctx, client, in.ConnectionID, product, vp.SKU, product.Name, images)
	created, err := uc.createProductSelfHealingDuplicateSKU(ctx, client, wooProduct)
	if err != nil {
		return nil, fmt.Errorf("pushProductToWoo: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}

	listing, err := uc.createListing(ctx, in, nil, false, created.ID)
	if err != nil {
		return nil, err
	}
	recordSyncLog(ctx, uc.db, uc.syncLogRepo, listing, channelsDomain.SyncFlowPush,
		channelsDomain.SyncLogStatusSuccess, "", wooProductToFields(wooProduct))

	return []channelsDomain.ProductChannelListing{listing}, nil
}

func (uc *PushProductToWooUseCase) pushSeparateProducts(ctx context.Context, client integrationsDomain.WooClient, product pmsDomain.Product, pushes []variantPush, existingByVariant map[string]channelsDomain.ProductChannelListing, in PushProductToWooInput, images []integrationsDomain.WooProductImage) ([]channelsDomain.ProductChannelListing, error) {
	var listings []channelsDomain.ProductChannelListing
	for _, vp := range pushes {
		if l, ok := existingByVariant[vp.Variant.ID]; ok {
			listings = append(listings, l)
			continue
		}

		name := fmt.Sprintf("%s - %s", product.Name, vp.Variant.Name)
		wooProduct := uc.buildWooProduct(ctx, client, in.ConnectionID, product, vp.SKU, name, images)
		created, err := uc.createProductSelfHealingDuplicateSKU(ctx, client, wooProduct)
		if err != nil {
			return nil, fmt.Errorf("pushProductToWoo: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
		}

		variantID := vp.Variant.ID
		listing, err := uc.createListing(ctx, in, &variantID, false, created.ID)
		if err != nil {
			return nil, err
		}
		recordSyncLog(ctx, uc.db, uc.syncLogRepo, listing, channelsDomain.SyncFlowPush,
			channelsDomain.SyncLogStatusSuccess, "", wooProductToFields(wooProduct))
		listings = append(listings, listing)
	}
	return listings, nil
}

func (uc *PushProductToWooUseCase) pushVariations(ctx context.Context, client integrationsDomain.WooClient, product pmsDomain.Product, pushes []variantPush, existingParent *channelsDomain.ProductChannelListing, existingByVariant map[string]channelsDomain.ProductChannelListing, in PushProductToWooInput, images []integrationsDomain.WooProductImage) ([]channelsDomain.ProductChannelListing, error) {
	var listings []channelsDomain.ProductChannelListing
	var parentExternalID string

	if existingParent != nil {
		parentExternalID = existingParent.ExternalID
		listings = append(listings, *existingParent)
	} else {
		attributeValues := make([]string, len(pushes))
		for i, vp := range pushes {
			attributeValues[i] = vp.Variant.Name
		}
		parentProduct := integrationsDomain.WooProduct{
			Name: product.Name, Description: product.Description,
			Status:     MapCommerceStatusToWoo(product.Status),
			Images:     images,
			Categories: uc.resolveCategoryForWoo(ctx, client, in.ConnectionID, product.CategoryID),
		}
		createdParent, err := client.CreateVariableProduct(ctx, parentProduct, wooVariantAttributeName, attributeValues)
		if err != nil {
			return nil, fmt.Errorf("pushProductToWoo: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
		}
		parentExternalID = createdParent.ID

		parentListing, err := uc.createListing(ctx, in, nil, false, parentExternalID)
		if err != nil {
			return nil, err
		}
		recordSyncLog(ctx, uc.db, uc.syncLogRepo, parentListing, channelsDomain.SyncFlowPush,
			channelsDomain.SyncLogStatusSuccess, "", wooProductToFields(parentProduct))
		listings = append(listings, parentListing)
	}

	for _, vp := range pushes {
		if l, ok := existingByVariant[vp.Variant.ID]; ok {
			listings = append(listings, l)
			continue
		}

		wooVariation := uc.buildWooVariation(ctx, vp)
		created, err := client.CreateVariation(ctx, parentExternalID, wooVariation)
		if err != nil {
			return nil, fmt.Errorf("pushProductToWoo: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
		}

		variantID := vp.Variant.ID
		listing, err := uc.createListing(ctx, in, &variantID, true, created.ID)
		if err != nil {
			return nil, err
		}
		recordSyncLog(ctx, uc.db, uc.syncLogRepo, listing, channelsDomain.SyncFlowPush,
			channelsDomain.SyncLogStatusSuccess, "", wooVariationToFields(wooVariation))
		listings = append(listings, listing)
	}
	return listings, nil
}

// buildWooProduct assembles the WooProduct payload for a simple product —
// used for both the single-SKU shape and each row of separate-products
// mode, which differ only in the name passed in (product name alone vs.
// "product - variant").
func (uc *PushProductToWooUseCase) buildWooProduct(ctx context.Context, client integrationsDomain.WooClient, connectionID string, product pmsDomain.Product, sku pmsDomain.SKU, name string, images []integrationsDomain.WooProductImage) integrationsDomain.WooProduct {
	salePrice, saleEndDate := WooSaleFields(sku)
	wooProduct := integrationsDomain.WooProduct{
		Name: name, Description: product.Description,
		Status: MapCommerceStatusToWoo(product.Status),
		SKU:    sku.SKUCode, GTIN: sku.Barcode, RegularPrice: FormatMinorUnitsAsWooPrice(sku.PriceMinorUnits),
		SalePrice: salePrice, SaleEndDate: saleEndDate,
		Virtual:    sku.IsVirtual,
		Images:     images,
		Categories: uc.resolveCategoryForWoo(ctx, client, connectionID, product.CategoryID),
	}
	// Only round-tripped when this SKU actually came from Woo originally
	// (WooTaxClassSlug set by ImportWooProductsUseCase) -- a native
	// CommerceOS SKU has no known Woo slug for its domain/tax.TaxClass
	// (there's no reverse mapping from an arbitrary TaxClass name back to
	// one), so it's left unset and Woo applies its own default (Standard,
	// taxable) exactly as it did before this field existed.
	if sku.WooTaxClassSlug != nil {
		wooProduct.TaxClass = *sku.WooTaxClassSlug
	}
	if sku.WooTaxStatus != nil {
		wooProduct.TaxStatus = *sku.WooTaxStatus
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
			wooProduct.MetaData = toWooMetaData(level.SourceBatch)
		}
	}
	return wooProduct
}

// resolveCategoryForWoo returns the single-entry Categories slice Woo's
// write API expects, or nil when the product has no category or none of
// this connection's Woo categories match it (see
// ResolveCategoryLinkUseCase) -- nil leaves the product Uncategorized on
// Woo's side, same as today, rather than failing the push.
func (uc *PushProductToWooUseCase) resolveCategoryForWoo(ctx context.Context, client integrationsDomain.WooClient, connectionID string, categoryID *string) []integrationsDomain.WooProductCategory {
	if categoryID == nil {
		return nil
	}
	externalID, err := uc.categoryLink.Execute(ctx, client, connectionID, *categoryID)
	if err != nil || externalID == "" {
		return nil
	}
	return []integrationsDomain.WooProductCategory{{ExternalID: externalID}}
}

// buildWooVariation is buildWooProduct's counterpart for variations mode —
// no name/description/status (inherited from the parent), plus the
// attribute option this variation represents.
func (uc *PushProductToWooUseCase) buildWooVariation(ctx context.Context, vp variantPush) integrationsDomain.WooVariation {
	salePrice, saleEndDate := WooSaleFields(vp.SKU)
	variation := integrationsDomain.WooVariation{
		SKU: vp.SKU.SKUCode, GTIN: vp.SKU.Barcode, RegularPrice: FormatMinorUnitsAsWooPrice(vp.SKU.PriceMinorUnits),
		SalePrice: salePrice, SaleEndDate: saleEndDate,
		AttributeName: wooVariantAttributeName, AttributeValue: vp.Variant.Name,
	}
	// See buildWooProduct's doc comment -- same "only round-trip a slug we
	// actually got from Woo" reasoning.
	if vp.SKU.WooTaxClassSlug != nil {
		variation.TaxClass = *vp.SKU.WooTaxClassSlug
	}
	if vp.SKU.WooTaxStatus != nil {
		variation.TaxStatus = *vp.SKU.WooTaxStatus
	}
	if !vp.SKU.IsVirtual {
		if level, err := uc.stockLevel.Execute(ctx, vp.SKU.ID); err == nil && level != nil {
			qty := int(level.Syncable)
			variation.ManageStock = true
			variation.StockQuantity = &qty
			variation.MetaData = toWooMetaData(level.SourceBatch)
		}
	}
	return variation
}

func (uc *PushProductToWooUseCase) createListing(ctx context.Context, in PushProductToWooInput, variantID *string, isVariation bool, externalID string) (channelsDomain.ProductChannelListing, error) {
	var listing channelsDomain.ProductChannelListing
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		listing = channelsDomain.ProductChannelListing{
			TenantID: tenant.IDFromContext(ctx), ProductID: in.ProductID, ConnectionID: in.ConnectionID,
			Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: externalID,
			VariantID: variantID, IsVariation: isVariation,
			SyncStatus: channelsDomain.SyncStatusSynced, LastSyncedAt: &now,
			CreatedAt: now, UpdatedAt: now,
		}
		return uc.listingRepo.Create(ctx, &listing)
	})
	if err != nil {
		return channelsDomain.ProductChannelListing{}, fmt.Errorf("pushProductToWoo: %w", err)
	}
	return listing, nil
}
