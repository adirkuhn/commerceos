package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/adirkuhn/commerceos/domain/assistant"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
)

// recentPurchaseOrders caps the POs get_supplier lists.
const recentPurchaseOrders = 5

type supplierLister interface {
	Execute(ctx context.Context) ([]suppliersDomain.Supplier, error)
}

type purchaseOrderLister interface {
	Execute(ctx context.Context) ([]procurementDomain.PurchaseOrder, error)
}

type supplierView struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email,omitempty"`
	Phone        string `json:"phone,omitempty"`
	Country      string `json:"country,omitempty"`
	LeadTimeDays *int   `json:"lead_time_days,omitempty"`
	Status       string `json:"status"`
	Notes        string `json:"notes,omitempty"`
}

func toSupplierView(s suppliersDomain.Supplier) supplierView {
	return supplierView{ID: s.ID, Name: s.Name, Email: s.Email, Phone: s.Phone, Country: s.Country,
		LeadTimeDays: s.LeadTimeDays, Status: string(s.Status), Notes: s.Notes}
}

// SearchSuppliersTool finds suppliers by name, email or phone.
type SearchSuppliersTool struct{ suppliers supplierLister }

func NewSearchSuppliersTool(suppliers supplierLister) *SearchSuppliersTool {
	return &SearchSuppliersTool{suppliers: suppliers}
}

func (t *SearchSuppliersTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "search_suppliers",
		Description: "Find suppliers by part of their name, email or phone (or list them all without a query) -- " +
			"gives each supplier's id for get_supplier and list_purchase_orders.",
		Parameters: object(map[string]any{"query": str("Part of the supplier's name, email or phone."), "limit": limitProp()}),
	}
}

func (t *SearchSuppliersTool) Call(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := decode(args, &in); err != nil {
		return "", err
	}
	all, err := t.suppliers.Execute(ctx)
	if err != nil {
		return "", err
	}
	q := strings.ToLower(strings.TrimSpace(in.Query))
	var matches []supplierView
	for _, s := range all {
		if q == "" || strings.Contains(strings.ToLower(s.Name+" "+s.Email+" "+s.Phone), q) {
			matches = append(matches, toSupplierView(s))
		}
	}
	total := len(matches)
	if limit := clampLimit(in.Limit); len(matches) > limit {
		matches = matches[:limit]
	}
	if matches == nil {
		matches = []supplierView{}
	}
	return toJSON(map[string]any{"suppliers": matches, "total": total})
}

// GetSupplierTool reads one supplier with its latest purchase orders.
type GetSupplierTool struct {
	suppliers supplierLister
	orders    purchaseOrderLister
}

func NewGetSupplierTool(suppliers supplierLister, orders purchaseOrderLister) *GetSupplierTool {
	return &GetSupplierTool{suppliers: suppliers, orders: orders}
}

func (t *GetSupplierTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name:        "get_supplier",
		Description: "Read one supplier -- contact details, lead time, notes -- and its latest purchase orders. Needs the id from search_suppliers.",
		Parameters:  object(map[string]any{"supplier_id": str("The supplier's id.")}, "supplier_id"),
	}
}

type purchaseOrderSummary struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	CurrencyCode    string `json:"currency_code,omitempty"`
	TotalMinorUnits int64  `json:"total_minor_units"`
	ExpectedDate    string `json:"expected_date,omitempty"`
	CreatedAt       string `json:"created_at"`
}

func toPurchaseOrderSummary(po procurementDomain.PurchaseOrder) purchaseOrderSummary {
	return purchaseOrderSummary{ID: po.ID, Status: string(po.Status), CurrencyCode: po.CurrencyCode, TotalMinorUnits: po.TotalMinorUnits,
		ExpectedDate: dayOfPtr(po.ExpectedDate), CreatedAt: formatTime(po.CreatedAt)}
}

func (t *GetSupplierTool) Call(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		SupplierID string `json:"supplier_id"`
	}
	if err := decode(args, &in); err != nil {
		return "", err
	}
	if err := required("supplier_id", in.SupplierID); err != nil {
		return "", err
	}
	all, err := t.suppliers.Execute(ctx)
	if err != nil {
		return "", err
	}
	var found *suppliersDomain.Supplier
	for i := range all {
		if all[i].ID == in.SupplierID {
			found = &all[i]
			break
		}
	}
	if found == nil {
		return "", fmt.Errorf("supplier %s not found -- look it up with search_suppliers", in.SupplierID)
	}
	pos, err := t.orders.Execute(ctx)
	if err != nil {
		return "", err
	}
	var own []procurementDomain.PurchaseOrder
	for _, po := range pos {
		if po.SupplierID != nil && *po.SupplierID == found.ID {
			own = append(own, po)
		}
	}
	sortNewestFirst(own)
	recent := make([]purchaseOrderSummary, 0, recentPurchaseOrders)
	for i := 0; i < len(own) && i < recentPurchaseOrders; i++ {
		recent = append(recent, toPurchaseOrderSummary(own[i]))
	}
	return toJSON(struct {
		supplierView
		PurchaseOrderCount   int                    `json:"purchase_order_count"`
		RecentPurchaseOrders []purchaseOrderSummary `json:"recent_purchase_orders"`
	}{toSupplierView(*found), len(own), recent})
}

func sortNewestFirst(pos []procurementDomain.PurchaseOrder) {
	sort.SliceStable(pos, func(i, j int) bool { return pos[i].CreatedAt.After(pos[j].CreatedAt) })
}
