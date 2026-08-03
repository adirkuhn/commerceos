// Package woocommerce adapts github.com/adirkuhn/go-woocommerce-api to
// domain/integrations' WooClient/WooClientFactory ports, so the domain and
// application layers never import the third-party client directly.
package woocommerce

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

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
	return &wooClient{
		client: client,
		// go-woocommerce-api has no Categories resource and doesn't expose
		// its internal httpClient/baseURL, so ListCategories builds its own
		// request against the same REST base the library itself targets —
		// these three are all it needs to replicate that.
		storeURL: strings.TrimRight(storeURL, "/"), consumerKey: consumerKey, consumerSecret: consumerSecret,
	}, nil
}

type wooClient struct {
	client                                *wc.WooCommerce
	storeURL, consumerKey, consumerSecret string
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

type wooCategoryDTO struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Parent int    `json:"parent"`
}

const categoriesPerPage = 100

// ListCategories pages through /wc/v3/products/categories directly — see
// the comment on wooClient's storeURL/consumerKey/consumerSecret fields for
// why this isn't going through the wrapped library.
func (c *wooClient) ListCategories(ctx context.Context) ([]integrationsDomain.WooCategory, error) {
	var out []integrationsDomain.WooCategory
	for page := 1; page <= maxCategoryPages; page++ {
		url := fmt.Sprintf("%s/wp-json/wc/v3/products/categories?page=%d&per_page=%d", c.storeURL, page, categoriesPerPage)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", c.basicAuthHeader())
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		var dtos []wooCategoryDTO
		decodeErr := json.NewDecoder(resp.Body).Decode(&dtos)
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("list categories: %s: status %d", url, resp.StatusCode)
		}
		if decodeErr != nil {
			return nil, decodeErr
		}
		if len(dtos) == 0 {
			break
		}
		for _, d := range dtos {
			parent := ""
			if d.Parent != 0 {
				parent = strconv.Itoa(d.Parent)
			}
			out = append(out, integrationsDomain.WooCategory{
				ExternalID: strconv.Itoa(d.ID), Name: d.Name, Slug: d.Slug, ParentExternalID: parent,
			})
		}
	}
	return out, nil
}

// maxCategoryPages mirrors import_woo_products.go's maxImportPages — a
// safety cap against a runaway pagination loop, not an expected real limit
// (100 pages * 100 per page = 10,000 categories).
const maxCategoryPages = 100

func (c *wooClient) basicAuthHeader() string {
	token := base64.StdEncoding.EncodeToString([]byte(c.consumerKey + ":" + c.consumerSecret))
	return "Basic " + token
}

// DownloadImage is a plain GET — Woo product images are regular media URLs,
// not behind the store's REST API auth.
func (c *wooClient) DownloadImage(ctx context.Context, url string) (io.ReadCloser, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, "", fmt.Errorf("download image: %s: status %d", url, resp.StatusCode)
	}
	return resp.Body, resp.Header.Get("Content-Type"), nil
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
