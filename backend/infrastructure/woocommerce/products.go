package woocommerce

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	wc "github.com/adirkuhn/go-woocommerce-api"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func (c *wooClient) ListProducts(ctx context.Context, page, perPage int) ([]integrationsDomain.WooProduct, error) {
	products, _, err := c.client.Products.List(ctx, &wc.ListProductParams{Page: page, PerPage: perPage})
	if err != nil {
		return nil, err
	}
	out := make([]integrationsDomain.WooProduct, len(products))
	for i, p := range products {
		out[i] = toWooProduct(p)
	}
	return out, nil
}

// ListCategories pages through /wc/v3/products/categories directly — see
// the comment on wooClient's storeURL/consumerKey/consumerSecret fields for
// why this isn't going through the wrapped library.
func (c *wooClient) DeleteProduct(ctx context.Context, externalProductID string) error {
	// force=true: a soft "move to trash" would leave the product findable
	// (and re-orderable) on the store -- a CommerceOS-side delete should
	// remove it outright, not park it in the trash.
	_, _, err := c.client.Products.Delete(ctx, externalProductID, &wc.DeleteProductParams{Force: "true"})
	return err
}

func (c *wooClient) CreateProduct(ctx context.Context, p integrationsDomain.WooProduct) (*integrationsDomain.WooProduct, error) {
	created, _, err := c.client.Products.Create(ctx, &wc.Product{
		Name:           p.Name,
		Type:           "simple",
		Status:         p.Status,
		Description:    p.Description,
		Sku:            p.SKU,
		GlobalUniqueID: p.GTIN,
		RegularPrice:   wc.FlexibleString(p.RegularPrice),
		SalePrice:      salePricePtr(p.SalePrice),
		DateOnSaleTo:   saleEndDatePtr(p.SaleEndDate),
		Virtual:        p.Virtual,
		ManageStock:    p.ManageStock,
		StockQuantity:  p.StockQuantity,
		TaxStatus:      p.TaxStatus,
		TaxClass:       p.TaxClass,
		MetaData:       toWcMetaData(p.MetaData),
		Images:         toWcImages(p.Images),
		Categories:     toWcCategories(p.Categories),
	})
	if err != nil {
		return nil, translateDuplicateSKUError(err)
	}
	result := toWooProduct(*created)
	return &result, nil
}

// translateDuplicateSKUError wraps as integrationsDomain.ErrWooDuplicateSKU
// any CreateProduct rejection whose HTTP status is 400, so
// PushProductToWooUseCase's self-heal (search by SKU, link to what's found)
// gets a chance to run.
//
// This used to gate on WooCommerce's own REST error code instead (an exact
// string match, later a substring match after the exact one turned out
// wrong) -- both attempts kept missing in production, because the actual
// code WooCommerce returns for this rejection was never directly observed:
// the logged error only ever rendered .Message (translated to the store's
// locale), never .Code, so every fix so far was a guess at a string this
// codebase can't actually see. Gating on status alone needs no such guess,
// and is safe to apply broadly: self-heal's own FindProductBySKU lookup is
// a harmless miss for a 400 that wasn't actually a duplicate SKU (nothing
// found -> the original error still surfaces unchanged, see
// createProductSelfHealingDuplicateSKU), so mislabeling some other 400
// costs one extra read, not a wrong result.
func translateDuplicateSKUError(err error) error {
	var wcErr *wc.ErrorResponse
	if errors.As(err, &wcErr) && wcErr.Response != nil && wcErr.Response.StatusCode == http.StatusBadRequest {
		return fmt.Errorf("%w: %w", integrationsDomain.ErrWooDuplicateSKU, err)
	}
	return err
}

func (c *wooClient) FindProductBySKU(ctx context.Context, sku string) (*integrationsDomain.WooProduct, error) {
	return c.findProductBySKUWithStatus(ctx, sku, "")
}

// FindTrashedProductBySKU is FindProductBySKU scoped to WordPress's trash
// status specifically -- WordPress's own "any" status (what an unset
// Status param resolves to) excludes trash, so a product moved to trash
// still reserves its SKU (blocking a future create with the same SKU)
// while being entirely invisible to FindProductBySKU's normal search. This
// is what lets self-heal actually find the real cause of a duplicate-SKU
// rejection that FindProductBySKU alone comes up empty on.
func (c *wooClient) FindTrashedProductBySKU(ctx context.Context, sku string) (*integrationsDomain.WooProduct, error) {
	return c.findProductBySKUWithStatus(ctx, sku, "trash")
}

func (c *wooClient) findProductBySKUWithStatus(ctx context.Context, sku, status string) (*integrationsDomain.WooProduct, error) {
	products, _, err := c.client.Products.List(ctx, &wc.ListProductParams{Sku: sku, Status: status})
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return nil, nil
	}
	result := toWooProduct(products[0])
	return &result, nil
}

func (c *wooClient) UpdateProductStock(ctx context.Context, parentExternalID, externalID string, quantity int, metadata []integrationsDomain.WooMetaData) error {
	if parentExternalID != "" {
		_, _, err := c.client.ProductVariations.Update(ctx, parentExternalID, externalID, &wc.ProductVariation{
			ManageStock:   true,
			StockQuantity: &quantity,
			MetaData:      toWcMetaData(metadata),
		})
		return err
	}
	_, _, err := c.client.Products.Update(ctx, externalID, &wc.Product{
		ManageStock:   true,
		StockQuantity: &quantity,
		MetaData:      toWcMetaData(metadata),
	})
	return err
}

func toWcMetaData(m []integrationsDomain.WooMetaData) []wc.MetaData {
	if len(m) == 0 {
		return nil
	}
	out := make([]wc.MetaData, len(m))
	for i, md := range m {
		out[i] = wc.MetaData{Key: md.Key, Value: md.Value}
	}
	return out
}

// toWcImages is the push-direction inverse of toWooProduct's image mapping
// -- Source is the field Woo's API reads as a URL to fetch and re-host, so
// only it (and Alt) are set; ID/DateCreated* are response-only fields Woo
// assigns itself.
func toWcImages(images []integrationsDomain.WooProductImage) []wc.Image {
	if len(images) == 0 {
		return nil
	}
	out := make([]wc.Image, len(images))
	for i, img := range images {
		out[i] = wc.Image{Source: img.URL, Alt: img.Alt}
	}
	return out
}

// toWcCategories is the push-direction inverse of toWooProduct's category
// mapping -- Woo's write API only ever looks at a category's numeric id
// (name/slug are read-direction/display only, ignored on write), so only
// ID is set. A CommerceOS category with no resolved Woo id (see
// application/channels.ResolveCategoryLinkUseCase) never reaches here --
// callers only append entries once they have a real ExternalID.
func toWcCategories(categories []integrationsDomain.WooProductCategory) []wc.ProductCategory {
	if len(categories) == 0 {
		return nil
	}
	out := make([]wc.ProductCategory, 0, len(categories))
	for _, cat := range categories {
		id, err := strconv.Atoi(cat.ExternalID)
		if err != nil {
			continue
		}
		out = append(out, wc.ProductCategory{ID: id})
	}
	return out
}

func (c *wooClient) UpdateProduct(ctx context.Context, externalProductID string, p integrationsDomain.WooProduct) (*integrationsDomain.WooProduct, error) {
	var dims *wc.ProductDimensions
	if p.Dimensions.Length != "" || p.Dimensions.Width != "" || p.Dimensions.Height != "" {
		dims = &wc.ProductDimensions{Length: p.Dimensions.Length, Width: p.Dimensions.Width, Height: p.Dimensions.Height}
	}
	updated, _, err := c.client.Products.Update(ctx, externalProductID, &wc.Product{
		Name:           p.Name,
		Status:         p.Status,
		Description:    p.Description,
		Sku:            p.SKU,
		GlobalUniqueID: p.GTIN,
		RegularPrice:   wc.FlexibleString(p.RegularPrice),
		SalePrice:      salePricePtr(p.SalePrice),
		DateOnSaleTo:   saleEndDatePtr(p.SaleEndDate),
		Virtual:        p.Virtual,
		ManageStock:    p.ManageStock,
		StockQuantity:  p.StockQuantity,
		Weight:         p.Weight,
		Dimensions:     dims,
		TaxStatus:      p.TaxStatus,
		TaxClass:       p.TaxClass,
		MetaData:       toWcMetaData(p.MetaData),
		Images:         toWcImages(p.Images),
		Categories:     toWcCategories(p.Categories),
	})
	if err != nil {
		return nil, err
	}
	result := toWooProduct(*updated)
	return &result, nil
}

func toWooProduct(p wc.Product) integrationsDomain.WooProduct {
	images := make([]integrationsDomain.WooProductImage, len(p.Images))
	for i, img := range p.Images {
		images[i] = integrationsDomain.WooProductImage{URL: img.Source, Alt: img.Alt}
	}
	categories := make([]integrationsDomain.WooProductCategory, len(p.Categories))
	for i, cat := range p.Categories {
		categories[i] = integrationsDomain.WooProductCategory{
			ExternalID: strconv.Itoa(cat.ID),
			Name:       cat.Name,
			Slug:       cat.Slug,
		}
	}
	var dimensions integrationsDomain.WooProductDimensions
	if p.Dimensions != nil {
		dimensions = integrationsDomain.WooProductDimensions{
			Length: p.Dimensions.Length, Width: p.Dimensions.Width, Height: p.Dimensions.Height,
		}
	}
	return integrationsDomain.WooProduct{
		ID:            strconv.Itoa(p.ID),
		Name:          p.Name,
		Slug:          p.Slug,
		SKU:           p.Sku,
		GTIN:          p.GlobalUniqueID,
		Description:   p.Description,
		Status:        p.Status,
		Type:          p.Type,
		RegularPrice:  string(p.RegularPrice),
		SalePrice:     salePriceFromWire(p.SalePrice),
		SaleEndDate:   fromWire(p.DateOnSaleTo),
		OnSale:        p.OnSale,
		Virtual:       p.Virtual,
		ManageStock:   p.ManageStock,
		StockQuantity: p.StockQuantity,
		Weight:        p.Weight,
		Dimensions:    dimensions,
		Images:        images,
		Categories:    categories,
		TaxStatus:     p.TaxStatus,
		TaxClass:      p.TaxClass,
	}
}
