package events

// API key actions (backend/application/apikeys). The key itself never
// appears in an event.
var (
	APIKeyCreated = defineAction("apikeys.api_key.created",
		"An API key was created (e.g. for the WooCommerce plugin). data: name, sales_channel_id.")
	APIKeyRevoked = defineAction("apikeys.api_key.revoked",
		"An API key was revoked and stops working.")
)
