package events

// Procurement actions -- purchase orders, their documents, receiving and
// stock booking (backend/application/procurement).
var (
	PurchaseOrderCreated = defineAction("procurement.purchase_order.created",
		"A purchase order was created. data.source: manual, document or purchase_planner.")
)
