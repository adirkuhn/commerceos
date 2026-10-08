package integrations

import "errors"

var (
	ErrConnectionNotFound   = errors.New("integration connection not found")
	ErrStoreURLInvalid      = errors.New("store URL must be a valid http(s) URL")
	ErrProviderNotSupported = errors.New("integration provider not supported")
	ErrStateInvalid         = errors.New("connect state is invalid or expired")
	// ErrConnectionTestFailed distinguishes an upstream WooCommerce rejection
	// (bad/revoked credentials) from an internal error — handlers map it to
	// 502, not 500.
	ErrConnectionTestFailed = errors.New("connection test failed")
	// ErrWooRequestFailed distinguishes a live WooCommerce API call failure
	// (network error, non-2xx response) from an internal error — handlers
	// map it to 502, not 500. Used by any use case that calls WooClient
	// beyond the connection test itself (e.g. product import).
	ErrWooRequestFailed = errors.New("woocommerce request failed")
	// ErrWooDuplicateSKU distinguishes WooCommerce rejecting a product
	// create specifically because a product with this SKU already exists
	// on the store (Woo's own SKU-uniqueness constraint, WC REST error
	// code "product_invalid_sku") from any other request failure — lets
	// PushProductToWooUseCase self-heal (search Woo by SKU, link the
	// existing product) instead of failing the same way on every retry.
	// This happens in practice when this codebase's own link to an
	// already-existing Woo product was lost (e.g. its
	// product_channel_listings row got orphaned by a connection
	// disconnect/reconnect) while the real product stays on the store.
	ErrWooDuplicateSKU = errors.New("woocommerce: a product with this SKU already exists")
	// ErrWooProductNotFoundBySKU guards LinkExistingWooProductUseCase:
	// staff supplied a SKU that doesn't match any product currently on
	// the store.
	ErrWooProductNotFoundBySKU = errors.New("no woocommerce product found with this SKU")
	// ErrNoTenantContext fires when the request's JWT carries no tenant at
	// all — should not happen in practice (lm2 always resolves a company on
	// login), kept as a defensive guard.
	ErrNoTenantContext      = errors.New("no tenant in request context")
	ErrAIConnectionNotFound = errors.New("ai connection not found")
	// ErrAPIKeyRequired guards providers other than Ollama, which needs no
	// credential (self-hosted, reachable via BaseURL alone).
	ErrAPIKeyRequired = errors.New("an API key is required for this provider")
	// ErrBaseURLRequired guards AIProviderOllama, which has no fixed
	// default endpoint the way hosted providers do.
	ErrBaseURLRequired = errors.New("a base URL is required for this provider")
	// ErrInvalidBolDeliveryCode guards UpdateConnectionSettingsUseCase --
	// checked against ValidBolDeliveryCodes so a typo is caught at save
	// time instead of surfacing as an opaque Bol 400 on the next push.
	ErrInvalidBolDeliveryCode = errors.New("invalid bol delivery code")
	// ErrBolEconomicOperatorsRequestFailed wraps a live call to Bol's
	// Economic Operator API -- handlers map it to 502, not 500. Commonly a
	// 403 because the connection's app registration lacks the
	// economic-operators.read/.manage OAuth scope, not a bug in this codebase.
	ErrBolEconomicOperatorsRequestFailed = errors.New("failed to fetch economic operators from bol")
	// ErrBolRequestRejected means Bol refused a request as invalid (a 4xx
	// other than auth/rate limiting) -- sending it again won't help, so
	// callers record it rather than retry.
	ErrBolRequestRejected = errors.New("bol rejected the request")
	// ErrWebhookSecretNotSet guards the provider-webhooks admin feature's
	// create action (application/integrations.CreateProviderWebhookUseCase)
	// -- a subscription can't be created before the tenant has a webhook
	// secret provisioned (SetWebhookSecretUseCase), since the new
	// subscription needs to embed/sign with it (WooCommerce: request body
	// `secret` field; MyParcel: `?token=` query param -- see that use
	// case's doc comment for why both reuse the same
	// WebhookSecretEncrypted field despite verifying differently).
	ErrWebhookSecretNotSet = errors.New("integrations: connection has no webhook secret set yet")
	// ErrProviderWebhookRequestFailed distinguishes a live provider API
	// call failure (list/create/delete a webhook subscription) from an
	// internal error -- handlers map it to 502, not 500.
	ErrProviderWebhookRequestFailed = errors.New("provider webhook request failed")
	ErrEmailHostRequired            = errors.New("smtp host is required")
	ErrEmailPortRequired            = errors.New("smtp port is required")
	ErrEmailFromRequired            = errors.New("from address is required")
	// ErrWhatsAppTestPhoneRequired guards SetWhatsAppTestModeUseCase --
	// enabling test mode with no redirect number would silently drop every
	// outbound WhatsApp message instead of the intended "sent to me
	// instead of the customer" behavior.
	ErrWhatsAppTestPhoneRequired = errors.New("a test phone number is required to enable whatsapp test mode")
	// ErrNotWhatsAppConnection guards SetWhatsAppTestModeUseCase against
	// being pointed at a connection of any other provider.
	ErrNotWhatsAppConnection = errors.New("connection is not a whatsapp connection")
)
