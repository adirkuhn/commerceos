package events

// Procurement actions -- purchase orders, their documents, receiving and
// stock booking (backend/application/procurement).
var (
	PurchaseOrderCreated = defineAction("procurement.purchase_order.created",
		"A purchase order was created. data.source: manual, document or purchase_planner.")
)

var (
	PurchaseOrderUpdated = defineAction("procurement.purchase_order.updated",
		"A purchase order changed (changes: supplier_id, warehouse_id, status, currency_code, expected_date, notes). data.reason is set when the system changed it: line_items_resolved (needs_review -> draft once every product line is linked), deliveries_reconciled (partially_received -> received once all deliveries together brought everything).")
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
		"An invoice or delivery differs from its purchase order. data: receiving_record_id, type (missing_item, quantity_mismatch, unexpected_item, expiry_mismatch, damaged_item), line_item_id, expected_qty, received_qty, source -- invoice (found by comparing the invoice) or manual (flagged by a person, e.g. damaged).")
)

var (
	ReceivingDiscrepancyAcknowledged = defineAction("procurement.receiving_discrepancy.acknowledged",
		"Someone reviewed an invoice discrepancy and accepted it. data: receiving_record_id, type.")
	ReceivingDiscrepancyLinked = defineAction("procurement.receiving_discrepancy.linked",
		"An unexpected invoice line and a missing purchase order line were matched as the same item (both become acknowledged; one event each). data: receiving_record_id, type, linked_discrepancy_id.")
	ReceivingRecordCompleted = defineAction("procurement.receiving_record.completed",
		"Review of an uploaded invoice was finished. data: purchase_order_id.")
	BookingStarted = defineAction("procurement.booking.started",
		"Stock booking (scanning a delivery in) started for a purchase order. data: purchase_order_id.")
	BookingResumed = defineAction("procurement.booking.resumed",
		"A paused stock booking was picked up again. data: purchase_order_id.")
	BookingPaused = defineAction("procurement.booking.paused",
		"A stock booking was paused. data: purchase_order_id.")
	BookingCompleted = defineAction("procurement.booking.completed",
		"A stock booking was finished: scans became stock (inventory raises its own events) and the PO became received or partially_received. data: purchase_order_id, completed_by, missing_item_count.")
	BookingScanCreated = defineAction("procurement.booking_scan.created",
		"An item was scanned during stock booking. data: booking_id, sku_id, action_type (receive, correction, damaged, lost), quantity, write_off_quantity, lot_number, expiry_date.")
	BookingScanDeleted = defineAction("procurement.booking_scan.deleted",
		"A booking scan was undone. data: booking_id, sku_id, action_type.")
	SupplierIssueUpdated = defineAction("procurement.supplier_issue.updated",
		"The claim status of a purchase order's missing, damaged or lost item changed (changes: status -- open, claimed, resolved; note). entity_id is the issue's source_id. data: purchase_order_id, source (booking_shortfall, booking_scan, receiving_discrepancy), kind (missing, damaged, lost).")
	GoodsReceiptCorrected = defineAction("procurement.goods.receipt_corrected",
		"A scanner booking's correction took received stock back off a purchase order line (counted wrong); entity_id is the SKU. data: purchase_order_line_item_id, booking_id, scan_id, quantity (negative).")
	SupplierCostRecorded = defineAction("procurement.supplier_cost.recorded",
		"A SKU's cost history by supplier gained rows (append-only, COM-11). entity_id is the SKU. data: kind (receipt -- stock booked in against a PO line; reversal -- received stock taken back off a line; correction -- a line repriced after the fact), purchase_order_id, purchase_order_line_item_id, supplier_id, quantity, landed_unit_cost_minor_units, currency_code.")
	SupplierIssueReportEmailed = defineAction("procurement.supplier_issue_report.emailed",
		"A purchase order's list of missing/damaged/lost items was emailed to the supplier; the open ones became claimed (one supplier_issue.updated each). entity_id is the purchase order. data: to, issue_count, claimed_count.")
)
