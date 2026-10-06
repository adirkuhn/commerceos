package procurement

// Where a purchase order came from -- data.source of
// procurement.purchase_order.created.
const (
	PurchaseOrderSourceManual   = "manual"
	PurchaseOrderSourceDocument = "document"
	PurchaseOrderSourcePlanner  = "purchase_planner"
)
