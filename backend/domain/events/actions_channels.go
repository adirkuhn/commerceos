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
