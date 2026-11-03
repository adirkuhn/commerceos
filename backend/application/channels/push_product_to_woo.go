package channels

import (
	"context"
	"errors"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
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
	outbox       eventrecord.Enqueuer
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
	outbox eventrecord.Enqueuer,
) *PushProductToWooUseCase {
	return &PushProductToWooUseCase{
		db: db, connRepo: connRepo, wooFactory: wooFactory, encSecret: encryptionSecret,
		productRepo: productRepo, variantRepo: variantRepo, skuRepo: skuRepo,
		listingRepo: listingRepo, syncLogRepo: syncLogRepo, stockLevel: stockLevel,
		mediaRepo: mediaRepo, categoryLink: categoryLink,
		outbox: outbox,
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
		if !product.IsSyncableToChannels() {
			return channelsDomain.ErrProductNotSyncable
		}

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
					before := *legacy
					legacy.VariantID = &variantID
					if err := uc.listingRepo.Update(ctx, legacy); err != nil {
						return err
					}
					if err := recordListingUpdated(ctx, uc.outbox, before, *legacy); err != nil {
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
	recordSyncLog(ctx, uc.db, uc.syncLogRepo, uc.outbox, listing, channelsDomain.SyncFlowPush,
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
		recordSyncLog(ctx, uc.db, uc.syncLogRepo, uc.outbox, listing, channelsDomain.SyncFlowPush,
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
		recordSyncLog(ctx, uc.db, uc.syncLogRepo, uc.outbox, parentListing, channelsDomain.SyncFlowPush,
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
		recordSyncLog(ctx, uc.db, uc.syncLogRepo, uc.outbox, listing, channelsDomain.SyncFlowPush,
			channelsDomain.SyncLogStatusSuccess, "", wooVariationToFields(wooVariation))
		listings = append(listings, listing)
	}
	return listings, nil
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
		if err := uc.listingRepo.Create(ctx, &listing); err != nil {
			return err
		}
		return recordListingCreated(ctx, uc.outbox, listing, ListingSourcePush)
	})
	if err != nil {
		return channelsDomain.ProductChannelListing{}, fmt.Errorf("pushProductToWoo: %w", err)
	}
	return listing, nil
}

// productPusher is PushProductToWooUseCase as its callers need it (ISP) --
// defined once here so tests don't need the full real use case (which
// itself needs a WooClient, stock getter, connection repo, etc).
type productPusher interface {
	Execute(ctx context.Context, in PushProductToWooInput) ([]channelsDomain.ProductChannelListing, error)
}
