package events

// PurchaseOrderCreated is raised when a draft purchase order is created
// from the Purchase Planner (application/procurement.
// CreatePurchaseOrderFromPlanUseCase). Source says where it came from so a
// later raiser (manual / document upload) can share the event. No
// consumer yet; exists so supplier notifications or planner analytics
// ("how much of the plan was ordered") can subscribe instead of polling.
type PurchaseOrderCreated struct {
	Base
	TenantID        string `json:"tenant_id"`
	PurchaseOrderID string `json:"purchase_order_id"`
	SupplierID      string `json:"supplier_id"`
	CurrencyCode    string `json:"currency_code"`
	Source          string `json:"source"`
	LineCount       int    `json:"line_count"`
}

// PurchaseOrderSourcePlanner: created from a purchase plan.
const PurchaseOrderSourcePlanner = "purchase_planner"

func (PurchaseOrderCreated) EventName() string { return "procurement.purchase_order.created" }
