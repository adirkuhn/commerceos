package events

// Order management actions (backend/application/oms). Typed events
// oms.sales_order.status_changed and payments.refund.* stay as they are.
var (
	SalesOrderCreated = defineAction("oms.sales_order.created",
		"A sales order was created. data: source (manual, pos, ...), status, number, currency_code, total_minor_units, customer_id, sales_channel_id, warehouse_id, line_count.")
	SalesOrderUpdated = defineAction("oms.sales_order.updated",
		"A sales order's flags or totals changed (changes: fulfilled, packed, needs_sku_review, needs_stock_review, total_minor_units). data.reason says why: picking_completed, packing_completed, marked_picked, line_item_added, line_items_linked, stock_shortfall, stock_restored, stock_review_repair.")
	SalesOrderDeleted = defineAction("oms.sales_order.deleted",
		"A sales order was deleted with its lines (sold stock is put back; inventory raises its own events). data: source, number, status, stock_restored.")
	SalesOrderLineItemAdded = defineAction("oms.sales_order_line_item.added",
		"A line was added to a sales order. data: sales_order_id, sku_id, quantity, price_minor_units, total_minor_units.")
	SalesOrderLineItemUpdated = defineAction("oms.sales_order_line_item.updated",
		"A sales order line changed (changes: stock_status -- pending when stock couldn't be taken yet, decremented once it was). data: sales_order_id, reason (stock_shortfall, stock_taken, stock_review_repair).")
	SalesOrderLineItemLinked = defineAction("oms.sales_order_line_item.linked",
		"A sales order line was linked to a catalog SKU (changes: sku_id). data: sales_order_id, source -- manual, or pick_scan (matched by product name when the SKU was scanned).")
	SalesOrderActivityRecorded = defineAction("oms.sales_order_activity.recorded",
		"An entry was added to a sales order's activity log. data: sales_order_id, activity_type, summary.")
	LineDiscrepancyDetected = defineAction("oms.line_discrepancy.detected",
		"An imported order differs from the channel's own order (reported on Stock Issues, the order itself is not changed): an item missing, a different quantity, or the channel's lines not adding up to its own total. Also raised when a difference reopens or its numbers change. data: sales_order_id, kind (missing, quantity, total), sku_code, name, channel_quantity, local_quantity, channel_total_minor_units, local_total_minor_units.")
	LineDiscrepancyResolved = defineAction("oms.line_discrepancy.resolved",
		"A reported order difference no longer shows when comparing with the channel (the order was fixed). data: sales_order_id, kind, sku_code, name.")
	LineDiscrepancyDismissed = defineAction("oms.line_discrepancy.dismissed",
		"Staff dismissed a reported order difference. data: sales_order_id, kind, sku_code, name.")

	PickingSessionStarted = defineAction("oms.picking_session.started",
		"Picking started for a sales order. data: sales_order_id.")
	PickingSessionResumed = defineAction("oms.picking_session.resumed",
		"A paused picking session was picked up again. data: sales_order_id.")
	PickingSessionPaused = defineAction("oms.picking_session.paused",
		"A picking session was paused. data: sales_order_id.")
	PickingSessionReset = defineAction("oms.picking_session.reset",
		"A picking session was started over: its scans were removed. data: sales_order_id, removed_scan_count.")
	PickingSessionAbandoned = defineAction("oms.picking_session.abandoned",
		"An open picking session was closed because packing took over or the order was marked picked. data: sales_order_id, abandoned_by.")
	PickingSessionCompleted = defineAction("oms.picking_session.completed",
		"Picking was finished; staged order changes were applied. data: sales_order_id, completed_by, missing_item_count, change_count.")
	PickScanCreated = defineAction("oms.pick_scan.created",
		"An item was scanned while picking. data: session_id, line_item_id, sku_id, action_type, quantity, reason.")
	PickScanDeleted = defineAction("oms.pick_scan.deleted",
		"A pick scan was undone. data: session_id, line_item_id, sku_id, action_type, quantity.")

	PackingSessionStarted = defineAction("oms.packing_session.started",
		"Packing started for a sales order. data: sales_order_id.")
	PackingSessionReset = defineAction("oms.packing_session.reset",
		"A packing session was started over: its scans were removed. data: sales_order_id, removed_scan_count.")
	PackingSessionCompleted = defineAction("oms.packing_session.completed",
		"Packing was finished; staged order changes (substitutions, extra items, refunds) were applied. data: sales_order_id, completed_by, missing_item_count, change_count.")
	PackScanCreated = defineAction("oms.pack_scan.created",
		"An item was handled while packing. data: session_id, line_item_id, sku_id, action_type (pack, freebie, new_item, substitute, refund), quantity, price_minor_units, original_qty_covered, return_to_stock, reason.")
	PackScanDeleted = defineAction("oms.pack_scan.deleted",
		"A pack scan was undone. data: session_id, line_item_id, sku_id, action_type, quantity.")
)

var SalesOrderLineItemSubstituted = defineAction("oms.sales_order_line_item.substituted",
	"A different product was given for an order line while picking or packing (its stock movements raise inventory events). data: sales_order_id, substitute_sku_id, quantity, original_qty_covered, return_to_stock, source (picking or packing).")
