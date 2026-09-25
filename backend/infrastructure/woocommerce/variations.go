package woocommerce

import (
	"context"
	"strconv"

	wc "github.com/adirkuhn/go-woocommerce-api"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func (c *wooClient) CreateVariableProduct(ctx context.Context, p integrationsDomain.WooProduct, attributeName string, attributeValues []string) (*integrationsDomain.WooProduct, error) {
	created, _, err := c.client.Products.Create(ctx, &wc.Product{
		Name:        p.Name,
		Type:        "variable",
		Status:      p.Status,
		Description: p.Description,
		Virtual:     p.Virtual,
		Images:      toWcImages(p.Images),
		Categories:  toWcCategories(p.Categories),
		Attributes: []wc.ProductAttributes{
			{Name: attributeName, Variation: true, Visible: true, Options: attributeValues},
		},
	})
	if err != nil {
		return nil, err
	}
	result := toWooProduct(*created)
	return &result, nil
}

// ensureAttributeOptionDeclared appends value to attributeName's Options on
// the parent product if it isn't already there -- a no-op GET+maybe-PUT.
// WooCommerce rejects creating/updating a variation whose attribute value
// isn't already declared on the parent, so this always runs before
// ProductVariations.Create.
func (c *wooClient) ensureAttributeOptionDeclared(ctx context.Context, parentExternalID, attributeName, value string) error {
	parent, _, err := c.client.Products.Get(ctx, parentExternalID)
	if err != nil {
		return err
	}
	for i, attr := range parent.Attributes {
		if attr.Name != attributeName {
			continue
		}
		for _, opt := range attr.Options {
			if opt == value {
				return nil
			}
		}
		parent.Attributes[i].Options = append(parent.Attributes[i].Options, value)
		_, _, err := c.client.Products.Update(ctx, parentExternalID, &wc.Product{Attributes: parent.Attributes})
		return err
	}
	// The named attribute doesn't exist on the parent at all -- shouldn't
	// happen for a product created via CreateVariableProduct, but declare
	// it fresh rather than failing outright.
	attrs := append(parent.Attributes, wc.ProductAttributes{Name: attributeName, Variation: true, Visible: true, Options: []string{value}})
	_, _, err = c.client.Products.Update(ctx, parentExternalID, &wc.Product{Attributes: attrs})
	return err
}

func (c *wooClient) CreateVariation(ctx context.Context, parentExternalID string, v integrationsDomain.WooVariation) (*integrationsDomain.WooVariation, error) {
	if err := c.ensureAttributeOptionDeclared(ctx, parentExternalID, v.AttributeName, v.AttributeValue); err != nil {
		return nil, err
	}
	created, _, err := c.client.ProductVariations.Create(ctx, parentExternalID, &wc.ProductVariation{
		Sku:            v.SKU,
		GlobalUniqueID: v.GTIN,
		RegularPrice:   wc.FlexibleString(v.RegularPrice),
		SalePrice:      salePricePtr(v.SalePrice),
		DateOnSaleTo:   v.SaleEndDate,
		ManageStock:    v.ManageStock,
		StockQuantity:  v.StockQuantity,
		Weight:         v.Weight,
		Dimensions:     wooVariationDimensions(v.Dimensions),
		Attributes:     []wc.VariationAttribute{{Name: v.AttributeName, Option: v.AttributeValue}},
		TaxStatus:      v.TaxStatus,
		TaxClass:       v.TaxClass,
		MetaData:       toWcMetaData(v.MetaData),
	})
	if err != nil {
		return nil, err
	}
	result := toWooVariation(*created)
	return &result, nil
}

func (c *wooClient) UpdateVariation(ctx context.Context, parentExternalID, variationExternalID string, v integrationsDomain.WooVariation) (*integrationsDomain.WooVariation, error) {
	updated, _, err := c.client.ProductVariations.Update(ctx, parentExternalID, variationExternalID, &wc.ProductVariation{
		Sku:            v.SKU,
		GlobalUniqueID: v.GTIN,
		RegularPrice:   wc.FlexibleString(v.RegularPrice),
		SalePrice:      salePricePtr(v.SalePrice),
		DateOnSaleTo:   v.SaleEndDate,
		ManageStock:    v.ManageStock,
		StockQuantity:  v.StockQuantity,
		Weight:         v.Weight,
		Dimensions:     wooVariationDimensions(v.Dimensions),
		TaxStatus:      v.TaxStatus,
		TaxClass:       v.TaxClass,
		MetaData:       toWcMetaData(v.MetaData),
	})
	if err != nil {
		return nil, err
	}
	result := toWooVariation(*updated)
	return &result, nil
}

func (c *wooClient) DeleteVariation(ctx context.Context, parentExternalID, variationExternalID string) error {
	// force=true: same reasoning as DeleteProduct -- remove outright, don't
	// park it in the trash.
	_, _, err := c.client.ProductVariations.Delete(ctx, parentExternalID, variationExternalID, &wc.DeleteProductParams{Force: "true"})
	return err
}

// salePricePtr always returns a non-nil pointer, even for "" -- every
// caller of this helper is on the product-fields sync path, which always
// has an opinion on the current sale price (including "none", represented
// as ""). A nil *wc.FlexibleString instead means "don't touch this field",
// which is what a stock-only push (UpdateProductStock) leaves it as by
// simply never calling this helper.
func salePricePtr(s string) *wc.FlexibleString {
	fs := wc.FlexibleString(s)
	return &fs
}

// salePriceFromWire is salePricePtr's read-side inverse -- a Woo response
// always includes sale_price (unlike an outgoing partial update, there's no
// "omitted means untouched" concern reading a GET/POST/PUT response), but
// the pointer is still nil-checked defensively rather than assumed non-nil.
func salePriceFromWire(fs *wc.FlexibleString) string {
	if fs == nil {
		return ""
	}
	return string(*fs)
}

func wooVariationDimensions(d integrationsDomain.WooProductDimensions) *wc.ProductDimensions {
	if d.Length == "" && d.Width == "" && d.Height == "" {
		return nil
	}
	return &wc.ProductDimensions{Length: d.Length, Width: d.Width, Height: d.Height}
}

func toWooVariation(v wc.ProductVariation) integrationsDomain.WooVariation {
	var attrName, attrValue string
	if len(v.Attributes) > 0 {
		attrName, attrValue = v.Attributes[0].Name, v.Attributes[0].Option
	}
	var dims integrationsDomain.WooProductDimensions
	if v.Dimensions != nil {
		dims = integrationsDomain.WooProductDimensions{Length: v.Dimensions.Length, Width: v.Dimensions.Width, Height: v.Dimensions.Height}
	}
	return integrationsDomain.WooVariation{
		ID: strconv.Itoa(v.ID), SKU: v.Sku, GTIN: v.GlobalUniqueID,
		RegularPrice: string(v.RegularPrice), SalePrice: salePriceFromWire(v.SalePrice),
		ManageStock: v.ManageStock, StockQuantity: v.StockQuantity,
		Weight: v.Weight, Dimensions: dims,
		AttributeName: attrName, AttributeValue: attrValue,
		TaxStatus: v.TaxStatus, TaxClass: v.TaxClass,
	}
}
