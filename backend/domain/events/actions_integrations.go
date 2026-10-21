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
)
