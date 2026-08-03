// Package woocommerce adapts github.com/adirkuhn/go-woocommerce-api to
// domain/integrations' WooClient/WooClientFactory ports, so the domain and
// application layers never import the third-party client directly.
package woocommerce

import (
	"context"
	"strconv"

	wc "github.com/adirkuhn/go-woocommerce-api"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

type Factory struct{}

func NewFactory() *Factory { return &Factory{} }

func (f *Factory) NewClient(storeURL, consumerKey, consumerSecret string) (integrationsDomain.WooClient, error) {
	client, err := wc.New(wc.Config{
		ShopURL:        storeURL,
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
	})
	if err != nil {
		return nil, err
	}
	return &wooClient{client: client}, nil
}

type wooClient struct {
	client *wc.WooCommerce
}

// TestConnection lists a single product as the cheapest real call that
// exercises WooCommerce's REST auth — a non-2xx response (bad key/secret,
// revoked access) surfaces as an error here.
func (c *wooClient) TestConnection(ctx context.Context) error {
	_, _, err := c.client.Products.List(ctx, &wc.ListProductParams{PerPage: 1})
	return err
}

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

func (c *wooClient) CreateProduct(ctx context.Context, p integrationsDomain.WooProduct) (*integrationsDomain.WooProduct, error) {
	created, _, err := c.client.Products.Create(ctx, &wc.Product{
		Name:          p.Name,
		Type:          "simple",
		Status:        p.Status,
		Description:   p.Description,
		Sku:           p.SKU,
		RegularPrice:  wc.FlexibleString(p.RegularPrice),
		Virtual:       p.Virtual,
		ManageStock:   p.ManageStock,
		StockQuantity: p.StockQuantity,
	})
	if err != nil {
		return nil, err
	}
	result := toWooProduct(*created)
	return &result, nil
}

func (c *wooClient) UpdateProductStock(ctx context.Context, externalProductID string, quantity int) error {
	_, _, err := c.client.Products.Update(ctx, externalProductID, &wc.Product{StockQuantity: &quantity})
	return err
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
	return integrationsDomain.WooProduct{
		ID:            strconv.Itoa(p.ID),
		Name:          p.Name,
		Slug:          p.Slug,
		SKU:           p.Sku,
		Description:   p.Description,
		Status:        p.Status,
		Type:          p.Type,
		RegularPrice:  string(p.RegularPrice),
		Virtual:       p.Virtual,
		ManageStock:   p.ManageStock,
		StockQuantity: p.StockQuantity,
		Images:        images,
		Categories:    categories,
	}
}
