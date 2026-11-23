package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/assistant"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

type warehousesLister interface {
	Execute(ctx context.Context) ([]invDomain.Warehouse, error)
}

// scopedStockReader is GetScopedStockUseCase: one SKU's quantity in one
// warehouse (batchID nil = every batch).
type scopedStockReader interface {
	Execute(ctx context.Context, skuID, warehouseID string, batchID *string) (int64, error)
}

// GetStockByWarehouseTool shows where a product's stock is.
type GetStockByWarehouseTool struct {
	product    productGetter
	warehouses warehousesLister
	scoped     scopedStockReader
	levels     stockLevelsReader
}

func NewGetStockByWarehouseTool(product productGetter, warehouses warehousesLister, scoped scopedStockReader, levels stockLevelsReader) *GetStockByWarehouseTool {
	return &GetStockByWarehouseTool{product: product, warehouses: warehouses, scoped: scoped, levels: levels}
}

func (t *GetStockByWarehouseTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "get_stock_by_warehouse",
		Description: "Stock per warehouse for a product's SKUs (or one SKU): units on hand in each warehouse, the total, and " +
			"what the sales channels are told is sellable. Give product_id (from search), and sku_id to narrow to one SKU.",
		Parameters: object(map[string]any{
			"product_id": str("The product id; its SKUs are listed with names and codes."),
			"sku_id":     str("Only this SKU (from get_product)."),
		}),
	}
}

type warehouseStockView struct {
	WarehouseID string `json:"warehouse_id"`
	Warehouse   string `json:"warehouse"`
	OnHand      int64  `json:"on_hand"`
}

type skuStockView struct {
	SKUID              string               `json:"sku_id"`
	SKUCode            string               `json:"sku_code,omitempty"`
	Variant            string               `json:"variant,omitempty"`
	TotalOnHand        int64                `json:"total_on_hand"`
	SellableOnChannels int64                `json:"sellable_on_channels"`
	Warehouses         []warehouseStockView `json:"warehouses"`
}

func (t *GetStockByWarehouseTool) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		ProductID string `json:"product_id"`
		SKUID     string `json:"sku_id"`
	}
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	if args.ProductID == "" && args.SKUID == "" {
		return "", fmt.Errorf("%w: product_id or sku_id is required", ErrInvalidArguments)
	}
	out := struct {
		ProductID   string         `json:"product_id,omitempty"`
		AdminPath   string         `json:"admin_path,omitempty"`
		ProductName string         `json:"product_name,omitempty"`
		SKUs        []skuStockView `json:"skus"`
	}{SKUs: []skuStockView{}}

	skus := []skuStockView{{SKUID: args.SKUID}}
	if args.ProductID != "" {
		d, err := t.product.Execute(ctx, args.ProductID)
		if err != nil {
			return "", err
		}
		out.ProductID, out.ProductName = d.Product.ID, d.Product.Name
		out.AdminPath = adminPath(productPage, d.Product.ID)
		skus = nil
		for _, v := range d.Variants {
			for _, s := range v.SKUs {
				if args.SKUID == "" || s.ID == args.SKUID {
					skus = append(skus, skuStockView{SKUID: s.ID, SKUCode: s.SKUCode, Variant: v.Variant.Name})
				}
			}
		}
		if len(skus) == 0 {
			return "", fmt.Errorf("%w: sku %s is not a SKU of product %s", ErrInvalidArguments, args.SKUID, args.ProductID)
		}
	}

	warehouses, err := t.warehouses.Execute(ctx)
	if err != nil {
		return "", err
	}
	ids := make([]string, len(skus))
	for i, s := range skus {
		ids[i] = s.SKUID
	}
	levels, err := t.levels.Execute(ctx, ids)
	if err != nil {
		return "", err
	}
	for _, s := range skus {
		s.TotalOnHand, s.SellableOnChannels = levels[s.SKUID].Total, levels[s.SKUID].Syncable
		s.Warehouses = make([]warehouseStockView, 0, len(warehouses))
		for _, w := range warehouses {
			qty, err := t.scoped.Execute(ctx, s.SKUID, w.ID, nil)
			if err != nil {
				return "", err
			}
			s.Warehouses = append(s.Warehouses, warehouseStockView{WarehouseID: w.ID, Warehouse: w.Name, OnHand: qty})
		}
		out.SKUs = append(out.SKUs, s)
	}
	return toJSON(out)
}
