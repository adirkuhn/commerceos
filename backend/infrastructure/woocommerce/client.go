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
	"time"

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
func (c *wooClient) DeleteProduct(ctx context.Context, externalProductID string) error {
	// force=true: a soft "move to trash" would leave the product findable
	// (and re-orderable) on the store -- a CommerceOS-side delete should
	// remove it outright, not park it in the trash.
	_, _, err := c.client.Products.Delete(ctx, externalProductID, &wc.DeleteProductParams{Force: "true"})
	return err
}

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

type wooSettingDTO struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// GetProductUnits reads the store's weight/dimension unit settings —
// there's no per-product unit on Woo's product resource, so this is the
// only way to know whether "0.5" means 0.5kg or 0.5lb.
func (c *wooClient) GetProductUnits(ctx context.Context) (string, string, error) {
	url := fmt.Sprintf("%s/wp-json/wc/v3/settings/products", c.storeURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", c.basicAuthHeader())
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	var settings []wooSettingDTO
	decodeErr := json.NewDecoder(resp.Body).Decode(&settings)
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("get product units: %s: status %d", url, resp.StatusCode)
	}
	if decodeErr != nil {
		return "", "", decodeErr
	}

	var weightUnit, dimensionUnit string
	for _, s := range settings {
		switch s.ID {
		case "woocommerce_weight_unit":
			weightUnit = s.Value
		case "woocommerce_dimension_unit":
			dimensionUnit = s.Value
		}
	}
	return weightUnit, dimensionUnit, nil
}

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

func (c *wooClient) ListOrders(ctx context.Context, page, perPage int, modifiedAfter *time.Time) ([]integrationsDomain.WooOrder, error) {
	// DatesAreGMT: true, paired with reading the *_gmt fields in
	// toWooOrder -- Woo's plain date_modified/date_created are the
	// store's local time (ambiguous without knowing its timezone
	// setting), the _gmt variants are always UTC. modified_after is
	// compared against whichever set DatesAreGMT selects.
	params := &wc.ListOrdersParams{Page: page, PerPage: perPage, OrderBy: "modified", Order: "asc", DatesAreGMT: true}
	if modifiedAfter != nil {
		params.ModifiedAfter = modifiedAfter.UTC().Format(time.RFC3339)
	}
	orders, _, err := c.client.Orders.List(ctx, params)
	if err != nil {
		return nil, err
	}
	out := make([]integrationsDomain.WooOrder, len(orders))
	for i, o := range orders {
		out[i] = toWooOrder(o)
	}
	return out, nil
}

func (c *wooClient) GetOrder(ctx context.Context, externalOrderID string) (*integrationsDomain.WooOrder, error) {
	order, _, err := c.client.Orders.Get(ctx, externalOrderID, nil)
	if err != nil {
		return nil, err
	}
	result := toWooOrder(*order)
	return &result, nil
}

func (c *wooClient) ListCustomers(ctx context.Context, page, perPage int) ([]integrationsDomain.WooCustomer, error) {
	customers, _, err := c.client.Customers.List(ctx, &wc.ListCustomerParams{Page: page, PerPage: perPage})
	if err != nil {
		return nil, err
	}
	out := make([]integrationsDomain.WooCustomer, len(customers))
	for i, cu := range customers {
		out[i] = toWooCustomer(cu)
	}
	return out, nil
}

func toWooCustomer(cu wc.Customer) integrationsDomain.WooCustomer {
	var billing, shipping integrationsDomain.WooAddress
	if cu.Billing != nil {
		billing = integrationsDomain.WooAddress{
			FirstName: cu.Billing.FirstName, LastName: cu.Billing.LastName, Company: cu.Billing.Company,
			Address1: cu.Billing.Address1, Address2: cu.Billing.Address2, City: cu.Billing.City, State: cu.Billing.State,
			Postcode: cu.Billing.Postcode, Country: cu.Billing.Country, Email: cu.Billing.Email, Phone: cu.Billing.Phone,
		}
	}
	if cu.Shipping != nil {
		shipping = integrationsDomain.WooAddress{
			FirstName: cu.Shipping.FirstName, LastName: cu.Shipping.LastName, Company: cu.Shipping.Company,
			Address1: cu.Shipping.Address1, Address2: cu.Shipping.Address2, City: cu.Shipping.City, State: cu.Shipping.State,
			Postcode: cu.Shipping.Postcode, Country: cu.Shipping.Country, Phone: cu.Shipping.Phone,
		}
	}
	return integrationsDomain.WooCustomer{
		ID: cu.ID, Email: cu.Email, FirstName: cu.FirstName, LastName: cu.LastName,
		Username: cu.Username, Role: cu.Role, Billing: billing, Shipping: shipping,
		IsPayingCustomer: cu.IsPayingCustomer, AvatarURL: cu.AvatarURL,
		DateCreated: cu.DateCreatedGmt, DateModified: cu.DateModifiedGmt,
	}
}

func toWooAddress(a *wc.Billing) integrationsDomain.WooAddress {
	if a == nil {
		return integrationsDomain.WooAddress{}
	}
	return integrationsDomain.WooAddress{
		FirstName: a.FirstName, LastName: a.LastName, Company: a.Company,
		Address1: a.Address1, Address2: a.Address2, City: a.City, State: a.State,
		Postcode: a.Postcode, Country: a.Country, Email: a.Email, Phone: a.Phone,
	}
}

func toWooShippingAddress(a *wc.Shipping) integrationsDomain.WooAddress {
	if a == nil {
		return integrationsDomain.WooAddress{}
	}
	return integrationsDomain.WooAddress{
		FirstName: a.FirstName, LastName: a.LastName, Company: a.Company,
		Address1: a.Address1, Address2: a.Address2, City: a.City, State: a.State,
		Postcode: a.Postcode, Country: a.Country, Phone: a.Phone,
	}
}

func toWooOrder(o wc.Order) integrationsDomain.WooOrder {
	lineItems := make([]integrationsDomain.WooOrderLineItem, len(o.LineItems))
	for i, li := range o.LineItems {
		lineItems[i] = integrationsDomain.WooOrderLineItem{
			ProductID: li.ProductID, VariationID: li.VariationID, SKU: li.Sku, Name: li.Name, Quantity: int64(li.Quantity),
			Price:    strconv.FormatFloat(li.Price, 'f', -1, 64),
			Subtotal: li.Subtotal, SubtotalTax: li.SubtotalTax, Total: li.Total, TotalTax: li.TotalTax,
		}
	}
	var shippingMethod, shippingMethodTitle string
	if len(o.ShippingLines) > 0 {
		shippingMethod = o.ShippingLines[0].MethodID
		shippingMethodTitle = o.ShippingLines[0].MethodTitle
	}
	return integrationsDomain.WooOrder{
		ID: o.ID, Number: o.Number, OrderKey: o.OrderKey, Status: o.Status, CustomerID: o.CustomerID,
		Billing: toWooAddress(o.Billing), Shipping: toWooShippingAddress(o.Shipping), CustomerNote: o.CustomerNote,
		Currency: o.Currency, DiscountTotal: o.DiscountTotal, ShippingTotal: o.ShippingTotal,
		TotalTax: o.TotalTax, Total: o.Total, PaymentMethod: o.PaymentMethod, PaymentMethodTitle: o.PaymentMethodTitle,
		ShippingMethod: shippingMethod, ShippingMethodTitle: shippingMethodTitle,
		TransactionID: o.TransactionID, DateCreated: o.DateCreatedGmt, DateModified: o.DateModifiedGmt,
		DatePaid: o.DatePaidGmt, DateCompleted: o.DateCompletedGmt, LineItems: lineItems,
	}
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
		SalePrice:      wc.FlexibleString(p.SalePrice),
		DateOnSaleTo:   p.SaleEndDate,
		Virtual:        p.Virtual,
		ManageStock:    p.ManageStock,
		StockQuantity:  p.StockQuantity,
		TaxStatus:      p.TaxStatus,
		TaxClass:       p.TaxClass,
		MetaData:       toWcMetaData(p.MetaData),
		Images:         toWcImages(p.Images),
	})
	if err != nil {
		return nil, err
	}
	result := toWooProduct(*created)
	return &result, nil
}

func (c *wooClient) CreateVariableProduct(ctx context.Context, p integrationsDomain.WooProduct, attributeName string, attributeValues []string) (*integrationsDomain.WooProduct, error) {
	created, _, err := c.client.Products.Create(ctx, &wc.Product{
		Name:        p.Name,
		Type:        "variable",
		Status:      p.Status,
		Description: p.Description,
		Virtual:     p.Virtual,
		Images:      toWcImages(p.Images),
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
		SalePrice:      wc.FlexibleString(v.SalePrice),
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
		GlobalUniqueID: v.GTIN,
		RegularPrice:   wc.FlexibleString(v.RegularPrice),
		SalePrice:      wc.FlexibleString(v.SalePrice),
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
		RegularPrice: string(v.RegularPrice), SalePrice: string(v.SalePrice),
		ManageStock: v.ManageStock, StockQuantity: v.StockQuantity,
		Weight: v.Weight, Dimensions: dims,
		AttributeName: attrName, AttributeValue: attrValue,
		TaxStatus: v.TaxStatus, TaxClass: v.TaxClass,
	}
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

func (c *wooClient) UpdateProduct(ctx context.Context, externalProductID string, p integrationsDomain.WooProduct) (*integrationsDomain.WooProduct, error) {
	var dims *wc.ProductDimensions
	if p.Dimensions.Length != "" || p.Dimensions.Width != "" || p.Dimensions.Height != "" {
		dims = &wc.ProductDimensions{Length: p.Dimensions.Length, Width: p.Dimensions.Width, Height: p.Dimensions.Height}
	}
	updated, _, err := c.client.Products.Update(ctx, externalProductID, &wc.Product{
		Name:           p.Name,
		Status:         p.Status,
		Description:    p.Description,
		GlobalUniqueID: p.GTIN,
		RegularPrice:   wc.FlexibleString(p.RegularPrice),
		SalePrice:      wc.FlexibleString(p.SalePrice),
		DateOnSaleTo:   p.SaleEndDate,
		Virtual:        p.Virtual,
		ManageStock:    p.ManageStock,
		StockQuantity:  p.StockQuantity,
		Weight:         p.Weight,
		Dimensions:     dims,
		TaxStatus:      p.TaxStatus,
		TaxClass:       p.TaxClass,
		MetaData:       toWcMetaData(p.MetaData),
		Images:         toWcImages(p.Images),
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
		SalePrice:     string(p.SalePrice),
		SaleEndDate:   p.DateOnSaleTo,
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

// ListTaxRates fetches the store's whole tax rate table in one page --
// see WooClient.ListTaxRates's doc comment for why this isn't paged.
func (c *wooClient) ListTaxRates(ctx context.Context) ([]integrationsDomain.WooTaxRate, error) {
	rates, _, err := c.client.TaxRates.List(ctx, &wc.ListTaxRatesParams{PerPage: 100})
	if err != nil {
		return nil, err
	}
	out := make([]integrationsDomain.WooTaxRate, len(rates))
	for i, r := range rates {
		out[i] = integrationsDomain.WooTaxRate{
			ID: strconv.Itoa(r.ID), Country: r.Country, State: r.State, Rate: r.Rate, Class: r.Class,
		}
	}
	return out, nil
}
