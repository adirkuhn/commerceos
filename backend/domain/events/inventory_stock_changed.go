package events

// InventoryStockChanged is raised whenever a SKU's stock ledger changes —
// a movement recorded via adjust/recount/transfer/mark-damaged-lost.
// Consumers re-push the SKU's *current* Syncable stock level to every
// channel its product is linked to — carries only the SKU id, not a
// quantity snapshot, since that could be stale by the time it's processed.
type InventoryStockChanged struct {
	Base
	TenantID string `json:"tenant_id"`
	SKUID    string `json:"sku_id"`
}

func (InventoryStockChanged) EventName() string { return "inventory.stock.changed" }
