package channels

import (
	"context"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

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
	// warehouse ledger entry to report. See wooInitialStock for why a
	// lookup failure creates it out of stock instead of unmanaged.
	if !sku.IsVirtual {
		resolveWooInitialStock(ctx, uc.stockLevel, sku).applyToProduct(&wooProduct)
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
		resolveWooInitialStock(ctx, uc.stockLevel, vp.SKU).applyToVariation(&variation)
	}
	return variation
}
