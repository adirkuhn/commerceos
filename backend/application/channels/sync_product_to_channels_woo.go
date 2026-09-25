package channels

import (
	"context"
	"fmt"
	"log"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

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
