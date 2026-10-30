package events

// Integration actions (backend/application/integrations). Credentials,
// API keys and webhook secrets never appear in an event -- only that they
// changed.
var (
	ConnectionDisconnected = defineAction("integrations.connection.disconnected",
		"A third-party connection (WooCommerce, Bol, MyParcel, WhatsApp, ...) was removed. data: provider.")
	ConnectionUpdated = defineAction("integrations.connection.updated",
		"A connection's settings changed (changes: default_currency, default_warehouse_id, bol_economic_operator_id, bol_delivery_code, bol_shipping_cost_minor_units, whatsapp_test_mode_enabled, whatsapp_test_phone, whatsapp_ai_parsing_enabled, base_url, instance_name, customer_code, customer_number, sandbox, api_key -- \"changed\" only, never the key). data: provider.")
	ConnectionTested = defineAction("integrations.connection.tested",
		"A connection's credentials were tested against the provider (changes: status). data: provider, succeeded.")
	ConnectionWebhookSecretSet = defineAction("integrations.connection.webhook_secret_set",
		"A connection's webhook signing secret was set or rotated (the secret itself is never in the event). data: provider.")
	ProviderWebhookCreated = defineAction("integrations.provider_webhook.created",
		"CommerceOS subscribed to a provider webhook (e.g. WooCommerce order.updated). data: connection_id, provider, topic.")
	ProviderWebhookDeleted = defineAction("integrations.provider_webhook.deleted",
		"A provider webhook subscription was removed. data: connection_id, provider.")
	AIConnectionConnected = defineAction("integrations.ai_connection.connected",
		"The tenant's AI provider was connected or switched. data: provider, model, base_url, fallback_models.")
	AIConnectionDisconnected = defineAction("integrations.ai_connection.disconnected",
		"The tenant's AI provider connection was removed.")
	MoneybirdSettingsUpdated = defineAction("integrations.moneybird_settings.updated",
		"A Moneybird connection's settings changed (changes: administration_id, default_ledger_account_id, auto_sync, tax_rates, payment_methods, guest_contact_id). entity_id is the connection.")
	MoneybirdDocumentSynced = defineAction("integrations.moneybird_document.synced",
		"A quote (as an estimate) or sales order (as an external sales invoice) was booked in Moneybird. entity_id is the quote/order. data: source_type, moneybird_id, reference, payment_registered.")
	MoneybirdDocumentDeleted = defineAction("integrations.moneybird_document.deleted",
		"A quote's estimate or an order's invoice was deleted from Moneybird and unlinked, so it can be sent again. entity_id is the quote/order. data: source_type, moneybird_id, reference.")
	MoneybirdDocumentSyncFailed = defineAction("integrations.moneybird_document.sync_failed",
		"Booking a quote or sales order in Moneybird failed. entity_id is the quote/order. data: source_type, moneybird_id (when the document itself was created), reference, error.")
)
