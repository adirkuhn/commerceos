package events

// Tenant settings and review moderation actions
// (backend/application/settings, backend/application/reviews).
var (
	TenantSettingsUpdated = defineAction("settings.tenant_settings.updated",
		"The tenant's settings changed; entity_id is the tenant (changes: default_currency, timezone, date_format, language, prices_include_tax, default_tax_country, inventory_sync_rules_enabled, batch_expiry_threshold_days, default_target_markup_basis_points, bundle_confidence_threshold, recommendations_min_support, recommendations_lookback_days, picking_mode, expiry_alert_enabled, expiry_alert_windows, low_stock_threshold).")
	AppearanceUpdated = defineAction("settings.appearance.updated",
		"The shop's admin appearance changed; entity_id is the tenant (changes: font, accent_color, sidebar_color, corners, text_size, density -- the first save diffs against the defaults).")
	ReviewDeleted = defineAction("reviews.review.deleted",
		"A product review was removed by moderation.")
)
