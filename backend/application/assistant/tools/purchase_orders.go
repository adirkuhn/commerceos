package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/domain/assistant"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

var purchaseOrderStatuses = []string{
	string(procurementDomain.PurchaseOrderStatusDraft), string(procurementDomain.PurchaseOrderStatusSent),
	string(procurementDomain.PurchaseOrderStatusPartiallyReceived), string(procurementDomain.PurchaseOrderStatusReceived),
	string(procurementDomain.PurchaseOrderStatusCancelled), string(procurementDomain.PurchaseOrderStatusNeedsReview),
}

type purchaseOrderGetter interface {
	Execute(ctx context.Context, id string) (*appprocurement.PurchaseOrderDetail, error)
}

type warehouseLister interface {
	Execute(ctx context.Context) ([]invDomain.Warehouse, error)
}

// dayOf renders a calendar date (expiry, expected delivery); zero is empty.
func dayOf(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02")
}

func dayOfPtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return dayOf(*t)
}

type namedRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func supplierRef(id *string, names map[string]string) *namedRef {
	if id == nil {
		return nil
	}
	return &namedRef{ID: *id, Name: names[*id]}
}

func supplierNames(ctx context.Context, suppliers supplierLister) (map[string]string, error) {
	all, err := suppliers.Execute(ctx)
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(all))
	for _, s := range all {
		names[s.ID] = s.Name
	}
	return names, nil
}

// ListPurchaseOrdersTool lists purchase orders, newest first.
type ListPurchaseOrdersTool struct {
	orders    purchaseOrderLister
	suppliers supplierLister
}

func NewListPurchaseOrdersTool(orders purchaseOrderLister, suppliers supplierLister) *ListPurchaseOrdersTool {
	return &ListPurchaseOrdersTool{orders: orders, suppliers: suppliers}
}

func (t *ListPurchaseOrdersTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "list_purchase_orders",
		Description: "List purchase orders to suppliers, newest first, optionally for one supplier (id from search_suppliers) " +
			"or one status. Use get_purchase_order for the lines.",
		Parameters: object(map[string]any{
			"supplier_id": str("Only this supplier's purchase orders."),
			"status":      enum("Only purchase orders in this status.", purchaseOrderStatuses...),
			"limit":       limitProp(),
		}),
	}
}

type purchaseOrderListItem struct {
	purchaseOrderSummary
	Supplier              *namedRef `json:"supplier"`
	SuggestedSupplierName string    `json:"suggested_supplier_name,omitempty"`
}

func (t *ListPurchaseOrdersTool) Call(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		SupplierID string `json:"supplier_id"`
		Status     string `json:"status"`
		Limit      int    `json:"limit"`
	}
	if err := decode(args, &in); err != nil {
		return "", err
	}
	if in.Status != "" && !contains(purchaseOrderStatuses, in.Status) {
		return "", fmt.Errorf("%w: unknown status %q", ErrInvalidArguments, in.Status)
	}
	pos, err := t.orders.Execute(ctx)
	if err != nil {
		return "", err
	}
	names, err := supplierNames(ctx, t.suppliers)
	if err != nil {
		return "", err
	}
	sortNewestFirst(pos)
	var out []purchaseOrderListItem
	total := 0
	for _, po := range pos {
		if in.SupplierID != "" && (po.SupplierID == nil || *po.SupplierID != in.SupplierID) {
			continue
		}
		if in.Status != "" && string(po.Status) != in.Status {
			continue
		}
		total++
		if len(out) < clampLimit(in.Limit) {
			out = append(out, purchaseOrderListItem{purchaseOrderSummary: toPurchaseOrderSummary(po),
				Supplier: supplierRef(po.SupplierID, names), SuggestedSupplierName: po.SuggestedSupplierName})
		}
	}
	if out == nil {
		out = []purchaseOrderListItem{}
	}
	return toJSON(map[string]any{"purchase_orders": out, "total": total})
}

// GetPurchaseOrderTool reads one purchase order with its lines.
type GetPurchaseOrderTool struct {
	get        purchaseOrderGetter
	suppliers  supplierLister
	warehouses warehouseLister
}

func NewGetPurchaseOrderTool(get purchaseOrderGetter, suppliers supplierLister, warehouses warehouseLister) *GetPurchaseOrderTool {
	return &GetPurchaseOrderTool{get: get, suppliers: suppliers, warehouses: warehouses}
}

func (t *GetPurchaseOrderTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "get_purchase_order",
		Description: "Read one purchase order -- status, supplier, expected date, receiving warehouse, notes and every line " +
			"(product, SKU, quantity, pack size, unit cost; shipping/discount/fee rows too). Needs the id from list_purchase_orders.",
		Parameters: object(map[string]any{"purchase_order_id": str("The purchase order's id.")}, "purchase_order_id"),
	}
}

type purchaseOrderLineView struct {
	ProductName           string `json:"product_name,omitempty"`
	SKUCode               string `json:"sku_code,omitempty"`
	SupplierSKU           string `json:"supplier_sku,omitempty"`
	SupplierDescription   string `json:"supplier_description,omitempty"`
	LineType              string `json:"line_type"`
	Quantity              int64  `json:"quantity"`
	UnitsPerPurchasedItem int64  `json:"units_per_purchased_item"`
	UnitCostMinorUnits    int64  `json:"unit_cost_minor_units"`
	LineTotalMinorUnits   int64  `json:"line_total_minor_units"`
}

func (t *GetPurchaseOrderTool) Call(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		PurchaseOrderID string `json:"purchase_order_id"`
	}
	if err := decode(args, &in); err != nil {
		return "", err
	}
	if err := required("purchase_order_id", in.PurchaseOrderID); err != nil {
		return "", err
	}
	d, err := t.get.Execute(ctx, in.PurchaseOrderID)
	if err != nil {
		return "", err
	}
	names, err := supplierNames(ctx, t.suppliers)
	if err != nil {
		return "", err
	}
	po := d.PurchaseOrder
	var warehouse *namedRef
	if po.WarehouseID != nil {
		ws, err := t.warehouses.Execute(ctx)
		if err != nil {
			return "", err
		}
		warehouse = &namedRef{ID: *po.WarehouseID}
		for _, w := range ws {
			if w.ID == *po.WarehouseID {
				warehouse.Name = w.Name
			}
		}
	}
	lines := make([]purchaseOrderLineView, len(d.LineItems))
	for i, l := range d.LineItems {
		lineType := string(l.LineType)
		if lineType == "" {
			lineType = string(procurementDomain.LineItemTypeProduct)
		}
		lines[i] = purchaseOrderLineView{
			ProductName: deref(l.ProductName), SKUCode: deref(l.SKUCode), SupplierSKU: l.RawSKU, SupplierDescription: l.RawDescription,
			LineType: lineType, Quantity: l.Quantity, UnitsPerPurchasedItem: l.EffectiveUnitsPerPurchasedItem(),
			UnitCostMinorUnits: l.UnitCostMinorUnits, LineTotalMinorUnits: l.Quantity * l.UnitCostMinorUnits,
		}
	}
	return toJSON(struct {
		purchaseOrderSummary
		Supplier              *namedRef               `json:"supplier"`
		SuggestedSupplierName string                  `json:"suggested_supplier_name,omitempty"`
		Warehouse             *namedRef               `json:"warehouse"`
		Notes                 string                  `json:"notes,omitempty"`
		PotentialDuplicate    bool                    `json:"potential_duplicate,omitempty"`
		DocumentCount         int                     `json:"document_count"`
		Lines                 []purchaseOrderLineView `json:"lines"`
	}{
		purchaseOrderSummary: toPurchaseOrderSummary(po), Supplier: supplierRef(po.SupplierID, names),
		SuggestedSupplierName: po.SuggestedSupplierName, Warehouse: warehouse, Notes: po.Notes,
		PotentialDuplicate: po.IsPotentialDuplicate, DocumentCount: len(d.Documents), Lines: lines,
	})
}
