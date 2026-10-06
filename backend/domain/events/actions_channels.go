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
