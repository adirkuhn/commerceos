package woocommerce

import (
	"context"
	"fmt"
	"strconv"

	wc "github.com/adirkuhn/go-woocommerce-api"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// ListProductStock reads one page of the store's products (any status but
// trash) as stock snapshots -- see integrationsDomain.WooClient.
func (c *wooClient) ListProductStock(ctx context.Context, page, perPage int) ([]integrationsDomain.WooStockSnapshot, error) {
	products, _, err := c.client.Products.List(ctx, &wc.ListProductParams{Page: page, PerPage: perPage})
	if err != nil {
		return nil, err
	}
	out := make([]integrationsDomain.WooStockSnapshot, len(products))
	for i, p := range products {
		out[i] = integrationsDomain.WooStockSnapshot{
			ID: strconv.Itoa(p.ID), Name: p.Name, SKU: p.Sku, Type: p.Type, ManageStock: p.ManageStock,
			StockQuantity: p.StockQuantity, Meta: metaStrings(p.MetaData),
		}
	}
	return out, nil
}

// ListVariationStock reads one page of a variable product's variations as
// stock snapshots.
func (c *wooClient) ListVariationStock(ctx context.Context, parentExternalID string, page, perPage int) ([]integrationsDomain.WooStockSnapshot, error) {
	variations, _, err := c.client.ProductVariations.List(ctx, parentExternalID, &wc.ListProductVariationParams{Page: page, PerPage: perPage})
	if err != nil {
		return nil, err
	}
	out := make([]integrationsDomain.WooStockSnapshot, len(variations))
	for i, v := range variations {
		out[i] = integrationsDomain.WooStockSnapshot{
			ID: strconv.Itoa(v.ID), SKU: v.Sku, ManageStock: v.ManageStock,
			StockQuantity: v.StockQuantity, Meta: metaStrings(v.MetaData),
		}
	}
	return out, nil
}

// metaStrings flattens Woo meta data to key -> string value (non-string
// values formatted, never dropped).
func metaStrings(md []wc.MetaData) map[string]string {
	out := make(map[string]string, len(md))
	for _, m := range md {
		if m.Value == nil {
			out[m.Key] = ""
			continue
		}
		if s, ok := m.Value.(string); ok {
			out[m.Key] = s
			continue
		}
		out[m.Key] = fmt.Sprint(m.Value)
	}
	return out
}
