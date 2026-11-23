package tools

import (
	"context"
	"encoding/json"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	"github.com/adirkuhn/commerceos/domain/assistant"
)

type productGetter interface {
	Execute(ctx context.Context, id string) (*apppms.ProductDetail, error)
}

type stockLevelsReader interface {
	Execute(ctx context.Context, skuIDs []string) (map[string]appinventory.StockLevel, error)
}

// GetProductTool reads a product with its SKUs, prices and stock.
type GetProductTool struct {
	get   productGetter
	stock stockLevelsReader
}

func NewGetProductTool(get productGetter, stock stockLevelsReader) *GetProductTool {
	return &GetProductTool{get: get, stock: stock}
}

func (t *GetProductTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "get_product",
		Description: "Read one product (id from search): status, tags and every SKU with its code, barcode, price, sale " +
			"price, cost price and current stock.",
		Parameters: object(map[string]any{"product_id": str("The product id.")}, "product_id"),
	}
}

type skuView struct {
	ID                  string `json:"id"`
	Variant             string `json:"variant,omitempty"`
	SKUCode             string `json:"sku_code"`
	Barcode             string `json:"barcode,omitempty"`
	UnitType            string `json:"unit_type,omitempty"`
	CurrencyCode        string `json:"currency_code"`
	PriceMinorUnits     int64  `json:"price_minor_units"`
	SalePriceMinorUnits *int64 `json:"sale_price_minor_units,omitempty"`
	SaleEndDate         string `json:"sale_end_date,omitempty"`
	CostPriceMinorUnits *int64 `json:"cost_price_minor_units,omitempty"`
	// StockOnHand is every unit in stock; StockSellableOnChannels is what
	// the channels are told (expiring batches may be held back).
	StockOnHand             int64 `json:"stock_on_hand"`
	StockSellableOnChannels int64 `json:"stock_sellable_on_channels"`
}

func (t *GetProductTool) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		ProductID string `json:"product_id"`
	}
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	if err := required("product_id", args.ProductID); err != nil {
		return "", err
	}
	d, err := t.get.Execute(ctx, args.ProductID)
	if err != nil {
		return "", err
	}
	var skuIDs []string
	for _, v := range d.Variants {
		for _, s := range v.SKUs {
			skuIDs = append(skuIDs, s.ID)
		}
	}
	levels, err := t.stock.Execute(ctx, skuIDs)
	if err != nil {
		return "", err
	}
	view := struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		Status      string    `json:"status"`
		Tags        []string  `json:"tags,omitempty"`
		HasVariants bool      `json:"has_variants"`
		SKUs        []skuView `json:"skus"`
	}{ID: d.Product.ID, Name: d.Product.Name, Status: string(d.Product.Status), Tags: d.Product.Tags, HasVariants: d.Product.HasVariants, SKUs: []skuView{}}
	for _, v := range d.Variants {
		for _, s := range v.SKUs {
			level := levels[s.ID]
			view.SKUs = append(view.SKUs, skuView{
				ID: s.ID, Variant: v.Variant.Name, SKUCode: s.SKUCode, Barcode: s.Barcode, UnitType: string(s.UnitType),
				CurrencyCode: s.CurrencyCode, PriceMinorUnits: s.PriceMinorUnits, SalePriceMinorUnits: s.SalePriceMinorUnits,
				SaleEndDate: formatTimePtr(s.SaleEndDate), CostPriceMinorUnits: s.CostPriceMinorUnits,
				StockOnHand: level.Total, StockSellableOnChannels: level.Syncable,
			})
		}
	}
	return toJSON(view)
}
