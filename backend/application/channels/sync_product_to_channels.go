package channels

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// listingShape is which of the three shapes ProductChannelListing rows for
// one product+connection are actually in, inferred from the rows
// themselves rather than trusted from the product's current flags — the
// flags describe what SHOULD be true, the rows describe what IS true on
// the channel right now. A mismatch between the two is exactly what
// triggers migration.
type listingShape int

const (
	shapeNone listingShape = iota
	shapeSingleSKU
	shapeSeparateProducts
	shapeVariations
)

func actualShape(listings []channelsDomain.ProductChannelListing) listingShape {
	if len(listings) == 0 {
		return shapeNone
	}
	for _, l := range listings {
		if l.VariantID != nil && l.IsVariation {
			return shapeVariations
		}
	}
	if len(listings) == 1 && listings[0].VariantID == nil {
		return shapeSingleSKU
	}
	return shapeSeparateProducts
}

func desiredShape(product pmsDomain.Product) listingShape {
	if !product.HasVariants {
		return shapeSingleSKU
	}
	if product.SyncVariantsAsWooVariations {
		return shapeVariations
	}
	return shapeSeparateProducts
}

// SyncProductToChannelsUseCase re-pushes a product's current state to every
// channel it's already linked to, in response to a pms.product.changed
// event. A product with no listings is a cheap no-op — most products
// aren't linked to any channel, and this runs on every core product/SKU
// edit, not just ones that happen to matter to a connected store.
//
// A product/variant whose SKU count isn't exactly 1 is silently skipped
// rather than erroring — same invariant PushProductToWooUseCase enforces
// on the initial push.
//
// When the actual shape of a connection's listings no longer matches what
// the product's HasVariants/SyncVariantsAsWooVariations flags call for
// (the flag toggled since the last push, or the product grew/lost
// variants), this deletes the old external representation and every old
// listing row for that connection, then re-pushes via PushProductToWooUseCase
// to recreate it in the new shape from scratch.
//
// Stock quantity is deliberately not pushed here — that's the sync
// schedule's job (ExportWooStockLevelsUseCase); duplicating it here would
// just mean two code paths racing to set the same field.
// bolProductPusher is the one method this use case needs from
// PushProductToBolUseCase (ISP), same reasoning as productPusher above.
// Optional: nil when the caller hasn't wired Bol support (e.g. an older
// call site not yet updated) -- syncBolConnection no-ops rather than
// panicking, since Bol support is additive and every existing call site
// predates it.
type bolProductPusher interface {
	Execute(ctx context.Context, in PushProductToBolInput) ([]channelsDomain.ProductChannelListing, error)
}

type SyncProductToChannelsUseCase struct {
	db           *gorm.DB
	productRepo  pmsDomain.ProductRepository
	variantRepo  pmsDomain.VariantRepository
	skuRepo      pmsDomain.SKURepository
	listingRepo  channelsDomain.ProductChannelListingRepository
	syncLogRepo  channelsDomain.SyncLogRepository
	connRepo     integrationsDomain.ConnectionRepository
	wooFactory   integrationsDomain.WooClientFactory
	encSecret    string
	pusher       productPusher
	bolPusher    bolProductPusher
	mediaRepo    productMediaLister
	categoryLink categoryLinkResolver
}

func NewSyncProductToChannelsUseCase(
	db *gorm.DB,
	productRepo pmsDomain.ProductRepository,
	variantRepo pmsDomain.VariantRepository,
	skuRepo pmsDomain.SKURepository,
	listingRepo channelsDomain.ProductChannelListingRepository,
	syncLogRepo channelsDomain.SyncLogRepository,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
	pusher productPusher,
	mediaRepo productMediaLister,
	categoryLink categoryLinkResolver,
	bolPusher bolProductPusher,
) *SyncProductToChannelsUseCase {
	return &SyncProductToChannelsUseCase{
		db: db, productRepo: productRepo, variantRepo: variantRepo, skuRepo: skuRepo,
		listingRepo: listingRepo, syncLogRepo: syncLogRepo,
		connRepo: connRepo, wooFactory: wooFactory, encSecret: encryptionSecret, pusher: pusher,
		mediaRepo: mediaRepo, categoryLink: categoryLink, bolPusher: bolPusher,
	}
}

// Execute loads the product's fresh state and fans it out to every linked
// channel. The returned error is reserved for infra-level failures (can't
// read from the DB) that warrant the caller nacking/retrying the whole
// event — a single channel being unreachable is recorded on that listing's
// SyncStatus instead, never returned, so one flaky store can't block
// (or endlessly retry) the whole event.
func (uc *SyncProductToChannelsUseCase) Execute(ctx context.Context, productID string) error {
	var product pmsDomain.Product
	var variants []pmsDomain.Variant
	skuByVariant := map[string]pmsDomain.SKU{}
	var listings []channelsDomain.ProductChannelListing
	var media []pmsDomain.Media

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		listings, err = uc.listingRepo.ListByProduct(ctx, productID)
		if err != nil {
			return err
		}
		if len(listings) == 0 {
			return nil
		}

		p, err := uc.productRepo.FindByID(ctx, productID)
		if err != nil {
			return err
		}
		product = *p

		media, err = uc.mediaRepo.ListByProduct(ctx, productID)
		if err != nil {
			return err
		}

		variants, err = uc.variantRepo.ListByProduct(ctx, productID)
		if err != nil {
			return err
		}
		for _, v := range variants {
			skus, err := uc.skuRepo.ListByVariant(ctx, v.ID)
			if err != nil {
				return err
			}
			// A bulk pool (UnitType=kilo) SKU is never itself sellable --
			// excluded here so it's treated the same as a "removed"
			// variant everywhere below: never (re)created on the channel,
			// and any stray listing for it gets cleaned up.
			if len(skus) == 1 && skus[0].UnitType != pmsDomain.SKUUnitTypeKilo {
				skuByVariant[v.ID] = skus[0]
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("syncProductToChannels: %w", err)
	}
	if len(listings) == 0 {
		return nil
	}

	byConnection := map[string][]channelsDomain.ProductChannelListing{}
	var order []string
	for _, l := range listings {
		if _, ok := byConnection[l.ConnectionID]; !ok {
			order = append(order, l.ConnectionID)
		}
		byConnection[l.ConnectionID] = append(byConnection[l.ConnectionID], l)
	}

	images := wooImagesFromMedia(media, product.Name)
	for _, connectionID := range order {
		uc.syncConnection(ctx, product, variants, skuByVariant, connectionID, byConnection[connectionID], images)
	}
	return nil
}

func (uc *SyncProductToChannelsUseCase) syncConnection(ctx context.Context, product pmsDomain.Product, variants []pmsDomain.Variant, skuByVariant map[string]pmsDomain.SKU, connectionID string, listings []channelsDomain.ProductChannelListing, images []integrationsDomain.WooProductImage) {
	switch integrationsDomain.Provider(listings[0].Provider) {
	case integrationsDomain.ProviderWooCommerce:
		// falls through to the existing shape-based logic below
	case integrationsDomain.ProviderBol:
		uc.syncBolConnection(ctx, product, connectionID)
		return
	default:
		return // no other provider implemented yet
	}

	desired := desiredShape(product)
	actual := actualShape(listings)
	if actual != desired {
		uc.migrateShape(ctx, product, connectionID, listings)
		return
	}

	switch desired {
	case shapeSingleSKU:
		sku, ok := skuByVariant[variants[0].ID]
		if !ok {
			return
		}
		uc.syncOne(ctx, product, sku, listings[0], images)
	case shapeSeparateProducts:
		uc.syncSeparateProducts(ctx, product, variants, skuByVariant, connectionID, listings, images)
	case shapeVariations:
		uc.syncVariations(ctx, product, variants, skuByVariant, connectionID, listings, images)
	}
}

// migrateShape tears down every existing external representation and
// listing row for this product+connection, then re-pushes from scratch via
// PushProductToWooUseCase — the mechanism that satisfies flipping
// SyncVariantsAsWooVariations either direction ("delete all other
// variants" and recreate in the new shape).
func (uc *SyncProductToChannelsUseCase) migrateShape(ctx context.Context, product pmsDomain.Product, connectionID string, listings []channelsDomain.ProductChannelListing) {
	client, err := uc.wooClientFor(ctx, connectionID)
	if err != nil {
		log.Printf("syncProductToChannels: migrating product %s on connection %s: %v", product.ID, connectionID, err)
		return
	}

	var parentExternalID string
	for _, l := range listings {
		if l.VariantID == nil {
			parentExternalID = l.ExternalID
		}
	}
	for _, l := range listings {
		switch {
		case l.VariantID != nil && l.IsVariation:
			if err := client.DeleteVariation(ctx, parentExternalID, l.ExternalID); err != nil {
				log.Printf("syncProductToChannels: deleting variation %s for migration: %v", l.ExternalID, err)
			}
		case l.VariantID != nil:
			if err := client.DeleteProduct(ctx, l.ExternalID); err != nil {
				log.Printf("syncProductToChannels: deleting product %s for migration: %v", l.ExternalID, err)
			}
		}
	}
	if parentExternalID != "" {
		if err := client.DeleteProduct(ctx, parentExternalID); err != nil {
			log.Printf("syncProductToChannels: deleting parent product %s for migration: %v", parentExternalID, err)
		}
	}

	if err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		for _, l := range listings {
			if err := uc.listingRepo.Delete(ctx, l.ID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		log.Printf("syncProductToChannels: deleting old listings for product %s: %v", product.ID, err)
		return
	}

	if _, err := uc.pusher.Execute(ctx, PushProductToWooInput{ProductID: product.ID, ConnectionID: connectionID}); err != nil {
		log.Printf("syncProductToChannels: re-pushing product %s after migration: %v", product.ID, err)
	}
}

func (uc *SyncProductToChannelsUseCase) syncSeparateProducts(ctx context.Context, product pmsDomain.Product, variants []pmsDomain.Variant, skuByVariant map[string]pmsDomain.SKU, connectionID string, listings []channelsDomain.ProductChannelListing, images []integrationsDomain.WooProductImage) {
	client, err := uc.wooClientFor(ctx, connectionID)
	if err != nil {
		log.Printf("syncProductToChannels: product %s -> connection %s: %v", product.ID, connectionID, err)
		return
	}

	listingByVariant := map[string]channelsDomain.ProductChannelListing{}
	for _, l := range listings {
		if l.VariantID != nil {
			listingByVariant[*l.VariantID] = l
		}
	}

	present := map[string]bool{}
	for _, v := range variants {
		sku, ok := skuByVariant[v.ID]
		if !ok {
			continue
		}
		present[v.ID] = true

		name := fmt.Sprintf("%s - %s", product.Name, v.Name)
		wooProduct := uc.buildWooProduct(ctx, client, connectionID, product, sku, name, images)

		l, existed := listingByVariant[v.ID]
		if !existed {
			created, err := client.CreateProduct(ctx, wooProduct)
			if err != nil {
				log.Printf("syncProductToChannels: creating separate product for variant %s: %v", v.ID, err)
				continue
			}
			variantID := v.ID
			newListing, err := uc.createListing(ctx, product.ID, connectionID, &variantID, false, created.ID)
			if err != nil {
				log.Printf("syncProductToChannels: recording new listing for variant %s: %v", v.ID, err)
				continue
			}
			recordSyncLog(ctx, uc.db, uc.syncLogRepo, newListing, channelsDomain.SyncFlowPush,
				channelsDomain.SyncLogStatusSuccess, "", wooProductToFields(wooProduct))
			continue
		}

		_, updateErr := client.UpdateProduct(ctx, l.ExternalID, wooProduct)
		uc.recordSyncResult(ctx, l, updateErr, wooProductToFields(wooProduct))
	}

	for variantID, l := range listingByVariant {
		if present[variantID] {
			continue
		}
		if err := client.DeleteProduct(ctx, l.ExternalID); err != nil {
			log.Printf("syncProductToChannels: deleting product for removed variant %s: %v", variantID, err)
		}
		uc.deleteListing(ctx, l.ID)
	}
}

func (uc *SyncProductToChannelsUseCase) syncVariations(ctx context.Context, product pmsDomain.Product, variants []pmsDomain.Variant, skuByVariant map[string]pmsDomain.SKU, connectionID string, listings []channelsDomain.ProductChannelListing, images []integrationsDomain.WooProductImage) {
	client, err := uc.wooClientFor(ctx, connectionID)
	if err != nil {
		log.Printf("syncProductToChannels: product %s -> connection %s: %v", product.ID, connectionID, err)
		return
	}

	var parentListing channelsDomain.ProductChannelListing
	listingByVariant := map[string]channelsDomain.ProductChannelListing{}
	for _, l := range listings {
		if l.VariantID == nil {
			parentListing = l
		} else {
			listingByVariant[*l.VariantID] = l
		}
	}

	parentProduct := integrationsDomain.WooProduct{
		Name: product.Name, Description: product.Description,
		Status:     MapCommerceStatusToWoo(product.Status),
		Images:     images,
		Categories: uc.resolveCategoryForWoo(ctx, client, connectionID, product.CategoryID),
	}
	if _, err := client.UpdateProduct(ctx, parentListing.ExternalID, parentProduct); err != nil {
		log.Printf("syncProductToChannels: updating parent product %s: %v", parentListing.ExternalID, err)
	}

	present := map[string]bool{}
	for _, v := range variants {
		sku, ok := skuByVariant[v.ID]
		if !ok {
			continue
		}
		present[v.ID] = true

		salePrice, saleEndDate := WooSaleFields(sku)
		wooVariation := integrationsDomain.WooVariation{
			SKU: sku.SKUCode, GTIN: sku.Barcode, RegularPrice: FormatMinorUnitsAsWooPrice(sku.PriceMinorUnits),
			SalePrice: salePrice, SaleEndDate: saleEndDate,
			AttributeName: wooVariantAttributeName, AttributeValue: v.Name,
		}
		if !sku.IsVirtual {
			applyWeightAndDimensions(ctx, client, sku, &wooVariation.Weight, &wooVariation.Dimensions)
		}

		l, existed := listingByVariant[v.ID]
		if !existed {
			created, err := client.CreateVariation(ctx, parentListing.ExternalID, wooVariation)
			if err != nil {
				log.Printf("syncProductToChannels: creating variation for variant %s: %v", v.ID, err)
				continue
			}
			variantID := v.ID
			newListing, err := uc.createListing(ctx, product.ID, connectionID, &variantID, true, created.ID)
			if err != nil {
				log.Printf("syncProductToChannels: recording new variation listing for variant %s: %v", v.ID, err)
				continue
			}
			recordSyncLog(ctx, uc.db, uc.syncLogRepo, newListing, channelsDomain.SyncFlowPush,
				channelsDomain.SyncLogStatusSuccess, "", wooVariationToFields(wooVariation))
			continue
		}

		_, updateErr := client.UpdateVariation(ctx, parentListing.ExternalID, l.ExternalID, wooVariation)
		uc.recordSyncResult(ctx, l, updateErr, wooVariationToFields(wooVariation))
	}

	for variantID, l := range listingByVariant {
		if present[variantID] {
			continue
		}
		if err := client.DeleteVariation(ctx, parentListing.ExternalID, l.ExternalID); err != nil {
			log.Printf("syncProductToChannels: deleting variation for removed variant %s: %v", variantID, err)
		}
		uc.deleteListing(ctx, l.ID)
	}
}

// syncBolConnection just re-runs PushProductToBolUseCase -- unlike Woo's
// three shapes, Bol offers have no "variation" mode to migrate between, and
// PushProductToBolUseCase is already idempotent (it looks up each
// variant's existing listing and PATCHes instead of re-creating), so
// there's no separate update-in-place logic to duplicate here.
func (uc *SyncProductToChannelsUseCase) syncBolConnection(ctx context.Context, product pmsDomain.Product, connectionID string) {
	if uc.bolPusher == nil {
		return
	}
	if _, err := uc.bolPusher.Execute(ctx, PushProductToBolInput{ProductID: product.ID, ConnectionID: connectionID}); err != nil {
		log.Printf("syncProductToChannels: bol product %s -> connection %s: %v", product.ID, connectionID, err)
	}
}

func (uc *SyncProductToChannelsUseCase) deleteListing(ctx context.Context, listingID string) {
	if err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.listingRepo.Delete(ctx, listingID)
	}); err != nil {
		log.Printf("syncProductToChannels: deleting listing %s: %v", listingID, err)
	}
}

func (uc *SyncProductToChannelsUseCase) createListing(ctx context.Context, productID, connectionID string, variantID *string, isVariation bool, externalID string) (channelsDomain.ProductChannelListing, error) {
	var listing channelsDomain.ProductChannelListing
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		listing = channelsDomain.ProductChannelListing{
			TenantID: tenant.IDFromContext(ctx), ProductID: productID, ConnectionID: connectionID,
			Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: externalID,
			VariantID: variantID, IsVariation: isVariation,
			SyncStatus: channelsDomain.SyncStatusSynced, LastSyncedAt: &now,
			CreatedAt: now, UpdatedAt: now,
		}
		return uc.listingRepo.Create(ctx, &listing)
	})
	return listing, err
}

func (uc *SyncProductToChannelsUseCase) recordSyncResult(ctx context.Context, listing channelsDomain.ProductChannelListing, syncErr error, fields map[string]string) {
	now := time.Now()
	listing.LastSyncedAt = &now
	listing.UpdatedAt = now
	status := channelsDomain.SyncLogStatusSuccess
	errMsg := ""
	if syncErr != nil {
		listing.SyncStatus = channelsDomain.SyncStatusError
		listing.LastSyncError = syncErr.Error()
		status = channelsDomain.SyncLogStatusError
		errMsg = syncErr.Error()
		log.Printf("syncProductToChannels: listing %s: %v", listing.ID, syncErr)
	} else {
		listing.SyncStatus = channelsDomain.SyncStatusSynced
		listing.LastSyncError = ""
	}

	recordSyncLog(ctx, uc.db, uc.syncLogRepo, listing, channelsDomain.SyncFlowPush, status, errMsg, fields)

	if uErr := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.listingRepo.Update(ctx, &listing)
	}); uErr != nil {
		log.Printf("syncProductToChannels: recording sync result for listing %s: %v", listing.ID, uErr)
	}
}

func (uc *SyncProductToChannelsUseCase) syncOne(ctx context.Context, product pmsDomain.Product, sku pmsDomain.SKU, listing channelsDomain.ProductChannelListing, images []integrationsDomain.WooProductImage) {
	client, err := uc.wooClientFor(ctx, listing.ConnectionID)
	if err != nil {
		log.Printf("syncProductToChannels: product %s -> connection %s: %v", product.ID, listing.ConnectionID, err)
		return
	}
	wooProduct := uc.buildWooProduct(ctx, client, listing.ConnectionID, product, sku, product.Name, images)
	fields := wooProductToFields(wooProduct)
	_, updateErr := client.UpdateProduct(ctx, listing.ExternalID, wooProduct)
	var wrapped error
	if updateErr != nil {
		wrapped = fmt.Errorf("%w: %w", integrationsDomain.ErrWooRequestFailed, updateErr)
	}
	uc.recordSyncResult(ctx, listing, wrapped, fields)
}

func (uc *SyncProductToChannelsUseCase) buildWooProduct(ctx context.Context, client integrationsDomain.WooClient, connectionID string, product pmsDomain.Product, sku pmsDomain.SKU, name string, images []integrationsDomain.WooProductImage) integrationsDomain.WooProduct {
	salePrice, saleEndDate := WooSaleFields(sku)
	wooProduct := integrationsDomain.WooProduct{
		Name: name, Description: product.Description,
		Status:       MapCommerceStatusToWoo(product.Status),
		SKU:          sku.SKUCode,
		GTIN:         sku.Barcode,
		RegularPrice: FormatMinorUnitsAsWooPrice(sku.PriceMinorUnits),
		SalePrice:    salePrice,
		SaleEndDate:  saleEndDate,
		Virtual:      sku.IsVirtual,
		Images:       images,
		Categories:   uc.resolveCategoryForWoo(ctx, client, connectionID, product.CategoryID),
	}
	if !sku.IsVirtual {
		applyWeightAndDimensions(ctx, client, sku, &wooProduct.Weight, &wooProduct.Dimensions)
	}
	return wooProduct
}

// resolveCategoryForWoo mirrors PushProductToWooUseCase's helper of the
// same name -- see its doc comment.
func (uc *SyncProductToChannelsUseCase) resolveCategoryForWoo(ctx context.Context, client integrationsDomain.WooClient, connectionID string, categoryID *string) []integrationsDomain.WooProductCategory {
	if categoryID == nil {
		return nil
	}
	externalID, err := uc.categoryLink.Execute(ctx, client, connectionID, *categoryID)
	if err != nil || externalID == "" {
		return nil
	}
	return []integrationsDomain.WooProductCategory{{ExternalID: externalID}}
}

// applyWeightAndDimensions needs the store's configured unit to convert
// into — a failure to fetch it just means weight/dimensions aren't pushed
// this round, not a fatal error for the whole sync.
func applyWeightAndDimensions(ctx context.Context, client integrationsDomain.WooClient, sku pmsDomain.SKU, weight *string, dims *integrationsDomain.WooProductDimensions) {
	weightUnit, dimensionUnit, err := client.GetProductUnits(ctx)
	if err != nil {
		return
	}
	if sku.WeightGrams != nil {
		if w, ok := gramsToWooWeight(*sku.WeightGrams, weightUnit); ok {
			*weight = w
		}
	}
	if sku.LengthMM != nil {
		if v, ok := mmToWooLength(*sku.LengthMM, dimensionUnit); ok {
			dims.Length = v
		}
	}
	if sku.WidthMM != nil {
		if v, ok := mmToWooLength(*sku.WidthMM, dimensionUnit); ok {
			dims.Width = v
		}
	}
	if sku.HeightMM != nil {
		if v, ok := mmToWooLength(*sku.HeightMM, dimensionUnit); ok {
			dims.Height = v
		}
	}
}

func (uc *SyncProductToChannelsUseCase) wooClientFor(ctx context.Context, connectionID string) (integrationsDomain.WooClient, error) {
	var conn integrationsDomain.Connection
	var secret string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		conn = *c
		secret, err = crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
		return err
	})
	if err != nil {
		return nil, err
	}
	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}
	return client, nil
}
