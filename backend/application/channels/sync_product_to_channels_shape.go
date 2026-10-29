package channels

import (
	"context"
	"fmt"
	"log"
	"time"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
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
			if err := recordListingDeleted(ctx, uc.outbox, l, ListingDeletedReasonShapeMigration); err != nil {
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
			if !sku.IsVirtual {
				resolveWooInitialStock(ctx, uc.stockLevel, sku).applyToProduct(&wooProduct)
			}
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
			recordSyncLog(ctx, uc.db, uc.syncLogRepo, uc.outbox, newListing, channelsDomain.SyncFlowPush,
				channelsDomain.SyncLogStatusSuccess, "", wooProductToFields(wooProduct))
			uc.catchUpNewListingStock(ctx, sku)
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
		uc.deleteListing(ctx, l, ListingDeletedReasonVariantRemoved)
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
			if !sku.IsVirtual {
				resolveWooInitialStock(ctx, uc.stockLevel, sku).applyToVariation(&wooVariation)
			}
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
			recordSyncLog(ctx, uc.db, uc.syncLogRepo, uc.outbox, newListing, channelsDomain.SyncFlowPush,
				channelsDomain.SyncLogStatusSuccess, "", wooVariationToFields(wooVariation))
			uc.catchUpNewListingStock(ctx, sku)
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
		uc.deleteListing(ctx, l, ListingDeletedReasonVariantRemoved)
	}
}

func (uc *SyncProductToChannelsUseCase) deleteListing(ctx context.Context, l channelsDomain.ProductChannelListing, reason string) {
	if err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.listingRepo.Delete(ctx, l.ID); err != nil {
			return err
		}
		return recordListingDeleted(ctx, uc.outbox, l, reason)
	}); err != nil {
		log.Printf("syncProductToChannels: deleting listing %s: %v", l.ID, err)
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
		if err := uc.listingRepo.Create(ctx, &listing); err != nil {
			return err
		}
		return recordListingCreated(ctx, uc.outbox, listing, ListingSourceVariantAdded)
	})
	return listing, err
}
