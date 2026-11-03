package events

// Sales channel configuration (backend/application/channels).
var (
	SalesChannelCreated = defineAction("channels.sales_channel.created",
		"A sales channel (web shop, Bol, POS...) was set up. data: name, type, connection_id, enrollment_mode, currency_code.")
	SalesChannelUpdated = defineAction("channels.sales_channel.updated",
		"A sales channel's settings changed (changes: name, connection_id, enrollment_mode, enrollment_category_id, enrollment_tag, price_adjustment_basis_points, quantity_breaks, default_warehouse_id, storefront_mode, sign_in_url, storefront_locale, storefront_locales, active, payment_connection_id).")
	SalesChannelDeleted = defineAction("channels.sales_channel.deleted",
		"A sales channel was deleted. data: name, type.")
	SalesChannelProductOverrideSet = defineAction("channels.sales_channel_product_override.set",
		"A product was explicitly included in or excluded from a sales channel, overriding its enrollment rule. data: channel_id, product_id, included.")
	ExternalCategoryLinkSaved = defineAction("channels.external_category_link.saved",
		"A CommerceOS category was linked to a category in an external shop (a person mapped it, or it was matched by slug on push). data: connection_id, provider, external_id, category_id, source (manual, slug_match).")
	ExternalCategoryLinkDeleted = defineAction("channels.external_category_link.deleted",
		"A category mapping to an external shop was removed.")
	SyncScheduleSaved = defineAction("channels.sync_schedule.saved",
		"An automatic sync schedule for a connection was created or changed. data: connection_id, entity_type, direction, enabled, interval_minutes, default_warehouse_id, default_currency.")
	SyncScheduleFailureAlerted = defineAction("channels.sync_schedule.failure_alerted",
		"An automatic sync kept failing for an hour and staff were alerted. data: connection_id, entity_type, direction, failing_since, error.")
	SyncScheduleRecovered = defineAction("channels.sync_schedule.recovered",
		"An automatic sync that staff were alerted about succeeded again. data: connection_id, entity_type, direction, failing_since.")
	ImportJobCreated = defineAction("channels.import_job.created",
		"A product import from an external shop was queued. data: connection_id, default_warehouse_id, default_currency, is_first_import.")
)

// Product listings -- a product (or variant) published on an external
// channel, and every sync of it.
var (
	ProductListingCreated = defineAction("channels.product_listing.created",
		"A product or variant was linked to a listing on an external channel. data: product_id, variant_id, connection_id, provider, external_id, is_variation, source (push, variant_added, link_existing, import).")
	ProductListingUpdated = defineAction("channels.product_listing.updated",
		"A listing's link changed (changes: variant_id, external_id, not_for_sale_note). data: product_id, connection_id, provider.")
	ProductListingDeleted = defineAction("channels.product_listing.deleted",
		"A listing was removed. data: product_id, variant_id, connection_id, provider, external_id, reason (shape_migration, variant_removed).")
	ProductListingSynced = defineAction("channels.product_listing.synced",
		"A listing was synced with its channel (a sync log entry was written; an unchanged successful re-sync records nothing). changes: the synced fields that changed since the previous sync, before/after as strings. data: product_id, connection_id, provider, external_id, direction (push, import), status (success, error), error_message.")
	ProductListingGated = defineAction("channels.product_listing.gated",
		"A listing was taken off sale on its channel because the product is no longer active (Woo status pushed, Bol offer put on hold). data: product_id, connection_id, provider, external_id.")
	ProductListingUngated = defineAction("channels.product_listing.ungated",
		"A gated listing went back on sale after the product became active again. data: product_id, connection_id, provider, external_id.")
)

// Imports from external channels -- records created in other modules
// straight from a channel's data.
var (
	ChannelProductImported = defineAction("channels.product.imported",
		"A product was created from an external shop's product (its listing also raises product_listing.created, source import). data: connection_id, provider, external_id, variant_id, sku_id, sku_code, category_id, price_minor_units, currency_code.")
	ChannelCategoryImported = defineAction("channels.category.imported",
		"A category was created from an external shop's category while importing its products. data: provider, external_id, name, slug, parent_id.")
	ChannelTaxClassImported = defineAction("channels.tax_class.imported",
		"A tax class was created for an external shop's tax class. data: provider, external_slug, name.")
	ChannelTaxRateImported = defineAction("channels.tax_rate.imported",
		"A tax rate was created or updated from an external shop's tax rates. data: provider, tax_class_id, country, rate_basis_points.")
	ChannelCustomerImported = defineAction("channels.customer.imported",
		"A customer was created from an external shop's customer. data: provider, external_customer_id, email, source (customer_import, order_import).")
	ChannelOrderImported = defineAction("channels.order.imported",
		"A sales order was created from an external channel's order. data: provider, connection_id, external_order_id, number, status, currency_code, total_minor_units, line_count, unlinked_line_count, needs_sku_review.")
	ChannelOrderStockShortageDetected = defineAction("channels.order.stock_shortage_detected",
		"An imported order's stock could not be taken for some lines (marked pending, order flagged for stock review). data: pending_line_item_ids.")
	ChannelOrderStatusPushed = defineAction("channels.order.status_pushed",
		"A sales order's status was sent to the channel it came from. data: provider, connection_id, external_order_id, status, channel_status.")
	BolShipmentConfirmationSent = defineAction("channels.bol_shipment_confirmation.sent",
		"Once the carrier scanned a Bol order's parcel, its transporter + track & trace was sent to Bol so the order closes there (action create_shipment), or -- when the parcel Bol was told about had been cancelled and replaced -- the shipment on Bol was corrected (action update_transport, one per Bol transport). Status pending while Bol processes it; failure when it could not be sent (Bol refused it, or the carrier has no Bol transporter code) and must be fixed by hand in Bol. data: sales_order_id, shipment_id, connection_id, external_order_id, action, transport_id, transporter_code, track_and_trace, process_status_id, status (pending, success, failure, timeout), error_message.")
	BolShipmentConfirmationResolved = defineAction("channels.bol_shipment_confirmation.resolved",
		"Bol finished processing a shipment confirmation. changes: status (pending -> success, failure or timeout). data: sales_order_id, external_order_id, process_status_id, error_message.")
	ChannelExternalProductDeleted = defineAction("channels.external_product.deleted",
		"A deleted product was also deleted from an external shop (entity_id is the external id). data: provider, connection_id.")
	ImportedProductDeleted = defineAction("channels.imported_product.deleted",
		"A product imported from a connection was deleted when that connection's imported data was cleared. data: connection_id, external_id.")
	ImportedCategoryDeleted = defineAction("channels.imported_category.deleted",
		"A category left empty after clearing a connection's imported products was deleted. data: connection_id.")
	ChannelSalePriceBackfilled = defineAction("channels.sale_price.backfilled",
		"A SKU's missing sale price was filled in from its external shop listing (entity_id is the SKU). changes: sale_price_minor_units, sale_end_date. data: connection_id, external_id.")
)

var SyncLogEntryDismissed = defineAction("channels.sync_log_entry.dismissed",
	"A failed channel sync was dismissed from the sync log (no data).")
