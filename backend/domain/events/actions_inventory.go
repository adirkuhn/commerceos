package events

// Inventory actions -- warehouses and stock batches (backend/application/inventory).
// Stock movements themselves raise inventory.stock.changed.
var (
	WarehouseCreated = defineAction("inventory.warehouse.created",
		"A warehouse was created. data: name, country, active.")
	WarehouseUpdated = defineAction("inventory.warehouse.updated",
		"A warehouse changed (changes: name, address1, address2, city, province, zip, country, phone, active).")
	WarehouseDeleted = defineAction("inventory.warehouse.deleted",
		"A warehouse was deleted. data: name.")
	BatchCreated = defineAction("inventory.batch.created",
		"A stock batch (lot) was created. data: sku_id, warehouse_id, lot_number, expiry_date.")
	BatchDeleted = defineAction("inventory.batch.deleted",
		"An unused stock batch was deleted. data: sku_id, warehouse_id, lot_number.")
	BatchExpiryAcknowledged = defineAction("inventory.batch_expiry.acknowledged",
		"A batch close to expiry was marked as seen for one close-to-expiry window: that window's list and email report skip it until it reaches the next window. data: window_days, sku_id, warehouse_id, lot_number.")
	BatchExpiryUnacknowledged = defineAction("inventory.batch_expiry.unacknowledged",
		"A batch's close-to-expiry acknowledgement was undone; it shows in that window's list and report again. data: window_days, sku_id, warehouse_id, lot_number.")
	BatchReviewed = defineAction("inventory.batch.reviewed",
		"An expired batch was dealt with and leaves the expiry review list. data: sku_id, warehouse_id, outcome -- dismissed (kept as is), written_off (remaining stock written off as lost) or discarded (a quantity discarded by hand).")
)
