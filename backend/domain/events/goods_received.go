package events

// GoodsReceived is raised when a purchase-order line item's stock is
// actually received into inventory -- a real inventory.Batch created with
// a landed cost basis (application/inventory.ReceiveStockUseCase), not
// merely a PO being created or edited (see procurement.PurchaseOrder's own
// design: "don't change inventory cost merely because a PO was created").
// Only raised for a costed receipt tied to a real PurchaseOrderLineItemID
// -- a manual or POS stock-in with no PO behind it is not a procurement
// event. No consumer yet; exists so future work (supplier performance
// reporting, a receiving-completed notification) has a stable event to
// subscribe to instead of polling purchase_order_line_items.
type GoodsReceived struct {
	Base
	TenantID                 string `json:"tenant_id"`
	SKUID                    string `json:"sku_id"`
	WarehouseID              string `json:"warehouse_id"`
	BatchID                  string `json:"batch_id"`
	PurchaseOrderLineItemID  string `json:"purchase_order_line_item_id"`
	Quantity                 int64  `json:"quantity"`
	LandedUnitCostMinorUnits int64  `json:"landed_unit_cost_minor_units"`
}

func (GoodsReceived) EventName() string { return "procurement.goods.received" }
