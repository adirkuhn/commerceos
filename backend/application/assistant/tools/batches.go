package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/adirkuhn/commerceos/domain/assistant"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

const (
	defaultExpiryWindowDays = 30
	maxExpiryWindowDays     = 365
)

type skuBatchLister interface {
	Execute(ctx context.Context, skuID string) ([]invDomain.Batch, error)
}

type expiringBatchLister interface {
	Execute(ctx context.Context, withinDays int) ([]invDomain.ExpiringBatch, error)
}

// ListBatchesTool lists one SKU's stock by lot and warehouse.
type ListBatchesTool struct {
	batches    skuBatchLister
	warehouses warehouseLister
}

func NewListBatchesTool(batches skuBatchLister, warehouses warehouseLister) *ListBatchesTool {
	return &ListBatchesTool{batches: batches, warehouses: warehouses}
}

func (t *ListBatchesTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "list_batches",
		Description: "List one SKU's stock by lot/batch and warehouse: lot number, expiry date, quantity, status, plus stock " +
			"held without a lot. Needs the sku_id (from get_product).",
		Parameters: object(map[string]any{"sku_id": str("The SKU's id.")}, "sku_id"),
	}
}

type batchView struct {
	BatchID    string   `json:"batch_id,omitempty"`
	Unlotted   bool     `json:"unlotted,omitempty"`
	LotNumber  string   `json:"lot_number,omitempty"`
	ExpiryDate string   `json:"expiry_date,omitempty"`
	Quantity   int64    `json:"quantity"`
	Status     string   `json:"status,omitempty"`
	ReceivedAt string   `json:"received_at,omitempty"`
	Warehouse  namedRef `json:"warehouse"`
}

func (t *ListBatchesTool) Call(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		SKUID string `json:"sku_id"`
	}
	if err := decode(args, &in); err != nil {
		return "", err
	}
	if err := required("sku_id", in.SKUID); err != nil {
		return "", err
	}
	batches, err := t.batches.Execute(ctx, in.SKUID)
	if err != nil {
		return "", err
	}
	ws, err := t.warehouses.Execute(ctx)
	if err != nil {
		return "", err
	}
	names := make(map[string]string, len(ws))
	for _, w := range ws {
		names[w.ID] = w.Name
	}
	out := []batchView{}
	var total int64
	for _, b := range batches {
		if b.IsGeneralLedger && b.Quantity == 0 {
			continue
		}
		total += b.Quantity
		v := batchView{Quantity: b.Quantity, Warehouse: namedRef{ID: b.WarehouseID, Name: names[b.WarehouseID]}}
		if b.IsGeneralLedger {
			v.Unlotted = true
		} else {
			v.BatchID, v.LotNumber, v.ExpiryDate, v.Status = b.ID, b.LotNumber, dayOfPtr(b.ExpiryDate), b.Status
			v.ReceivedAt = formatTime(b.ReceivedAt)
		}
		out = append(out, v)
	}
	return toJSON(map[string]any{"batches": out, "total_quantity": total})
}

// ListExpiringStockTool lists batches expiring soon, soonest first.
type ListExpiringStockTool struct{ expiring expiringBatchLister }

func NewListExpiringStockTool(expiring expiringBatchLister) *ListExpiringStockTool {
	return &ListExpiringStockTool{expiring: expiring}
}

func (t *ListExpiringStockTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "list_expiring_stock",
		Description: "List stock batches that expire within the next days (default 30), soonest first, with product, SKU, " +
			"lot, warehouse and quantity. Use for 'what expires soon' questions.",
		Parameters: object(map[string]any{
			"within_days": map[string]any{"type": "integer", "minimum": 1, "maximum": maxExpiryWindowDays,
				"description": fmt.Sprintf("Expiring within this many days (default %d).", defaultExpiryWindowDays)},
			"limit": limitProp(),
		}),
	}
}

type expiringView struct {
	BatchID       string `json:"batch_id"`
	ProductName   string `json:"product_name"`
	SKUID         string `json:"sku_id"`
	SKUCode       string `json:"sku_code"`
	LotNumber     string `json:"lot_number"`
	ExpiryDate    string `json:"expiry_date"`
	Quantity      int64  `json:"quantity"`
	WarehouseID   string `json:"warehouse_id"`
	WarehouseName string `json:"warehouse_name"`
}

func (t *ListExpiringStockTool) Call(ctx context.Context, args json.RawMessage) (string, error) {
	in := struct {
		WithinDays *int `json:"within_days"`
		Limit      int  `json:"limit"`
	}{}
	if err := decode(args, &in); err != nil {
		return "", err
	}
	days := defaultExpiryWindowDays
	if in.WithinDays != nil {
		days = *in.WithinDays
	}
	if days < 1 || days > maxExpiryWindowDays {
		return "", fmt.Errorf("%w: within_days must be 1..%d", ErrInvalidArguments, maxExpiryWindowDays)
	}
	batches, err := t.expiring.Execute(ctx, days)
	if err != nil {
		return "", err
	}
	sort.SliceStable(batches, func(i, j int) bool { return batches[i].ExpiryDate.Before(batches[j].ExpiryDate) })
	out := []expiringView{}
	for _, b := range batches {
		if len(out) == clampLimit(in.Limit) {
			break
		}
		out = append(out, expiringView{BatchID: b.BatchID, ProductName: b.ProductName, SKUID: b.SKUID, SKUCode: b.SKUCode,
			LotNumber: b.LotNumber, ExpiryDate: dayOf(b.ExpiryDate), Quantity: b.Quantity, WarehouseID: b.WarehouseID, WarehouseName: b.WarehouseName})
	}
	return toJSON(map[string]any{"batches": out, "total": len(batches), "within_days": days})
}
