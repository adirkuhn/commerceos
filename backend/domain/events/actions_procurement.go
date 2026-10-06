package events

// Procurement actions -- purchase orders, their documents, receiving and
// stock booking (backend/application/procurement).
var (
	PurchaseOrderCreated = defineAction("procurement.purchase_order.created",
		"A purchase order was created. data.source: manual, document or purchase_planner.")
)

var (
	PurchaseOrderUpdated = defineAction("procurement.purchase_order.updated",
		"A purchase order changed (changes: supplier_id, warehouse_id, status, currency_code, expected_date, notes). data.reason is set when the system changed it: line_items_resolved (needs_review -> draft once every product line is linked).")
	PurchaseOrderDeleted = defineAction("procurement.purchase_order.deleted",
		"A purchase order was deleted with its line items. data: supplier_id, status.")
	PurchaseOrderLineItemAdded = defineAction("procurement.purchase_order_line_item.added",
		"A line was added to a purchase order. data: purchase_order_id, sku_id, quantity, units_per_purchased_item, unit_cost_minor_units, line_type, source -- manual, or duplicate_document (copied from the PO an identical document was already attached to).")
	PurchaseOrderLineItemUpdated = defineAction("procurement.purchase_order_line_item.updated",
		"A purchase order line changed (changes: sku_id, raw_sku, raw_description, quantity, units_per_purchased_item, unit_cost_minor_units, line_type). data: purchase_order_id.")
	PurchaseOrderLineItemRemoved = defineAction("procurement.purchase_order_line_item.removed",
		"A line was removed from a purchase order. data: purchase_order_id, sku_id.")
	PurchaseOrderLineItemLinked = defineAction("procurement.purchase_order_line_item.linked",
		"A purchase order line was linked to a catalog SKU (changes: sku_id, units_per_purchased_item). data: purchase_order_id, source -- manual (a person picked the SKU) or supplier_alias (matched from the supplier's learned aliases).")
)

var (
	PurchaseOrderDocumentAttached = defineAction("procurement.purchase_order_document.attached",
		"A supplier document (invoice / order confirmation file) was attached to a purchase order. data: purchase_order_id, file_name, content_type, parsing_job_id, duplicate_of_purchase_order_id (set when the same file is already on another PO; its lines are then copied instead of parsed).")
)

var (
	PurchaseOrderDocumentDeleted = defineAction("procurement.purchase_order_document.deleted",
		"A supplier document was removed from a purchase order (its file deleted from storage). data: purchase_order_id, file_name.")
	PurchaseOrderDocumentParsingRetried = defineAction("procurement.purchase_order_document.parsing_retried",
		"Parsing of a supplier document was requested again. data: purchase_order_id, parsing_job_id.")
	PurchaseOrderDocumentParsed = defineAction("procurement.purchase_order_document.parsed",
		"A supplier document was read into purchase order lines (each line also raises purchase_order_line_item.added, source document). data: purchase_order_id, parsing_job_id, line_count, unmatched_count, suggested_supplier_name, suggested_currency_code.")
	ReceivingRecordCreated = defineAction("procurement.receiving_record.created",
		"A supplier invoice was uploaded against a purchase order to check what was delivered. data: purchase_order_id, file_name, content_type, parsing_job_id, potential_duplicate.")
	ReceivingRecordParsed = defineAction("procurement.receiving_record.parsed",
		"An uploaded invoice was compared with its purchase order (each difference also raises receiving_discrepancy.detected). data: purchase_order_id, parsing_job_id, discrepancy_count.")
	ReceivingDiscrepancyDetected = defineAction("procurement.receiving_discrepancy.detected",
		"An invoice differs from its purchase order. data: receiving_record_id, type (missing_item, quantity_mismatch, unexpected_item, expiry_mismatch), line_item_id, expected_qty, received_qty.")
)
