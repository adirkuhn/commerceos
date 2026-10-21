package integrations

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrConnectionNotFound   = errs.New("CONNECTION_NOT_FOUND", "integration connection not found")
	ErrStoreURLInvalid      = errs.New("STORE_URL_INVALID", "store URL must be a valid http(s) URL")
	ErrProviderNotSupported = errs.New("INTEGRATION_PROVIDER_NOT_SUPPORTED", "integration provider not supported")
	ErrStateInvalid         = errs.New("CONNECT_STATE_INVALID", "connect state is invalid or expired")
	// ErrConnectionTestFailed distinguishes an upstream WooCommerce rejection
	// (bad/revoked credentials) from an internal error — handlers map it to
	// 502, not 500.
	ErrConnectionTestFailed = errs.NewWithCause("CONNECTION_TEST_FAILED", "connection test failed")
	// ErrWooRequestFailed distinguishes a live WooCommerce API call failure
	// (network error, non-2xx response) from an internal error — handlers
	// map it to 502, not 500. Used by any use case that calls WooClient
	// beyond the connection test itself (e.g. product import).
	ErrWooRequestFailed = errs.NewWithCause("WOO_REQUEST_FAILED", "woocommerce request failed")
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
	ErrWooDuplicateSKU = errs.New("WOO_DUPLICATE_SKU", "woocommerce: a product with this SKU already exists")
	// ErrWooProductNotFoundBySKU guards LinkExistingWooProductUseCase:
	// staff supplied a SKU that doesn't match any product currently on
	// the store.
	ErrWooProductNotFoundBySKU = errs.New("WOO_PRODUCT_NOT_FOUND_BY_SKU", "no woocommerce product found with this SKU")
	// ErrNoTenantContext fires when the request's JWT carries no tenant at
	// all — should not happen in practice (lm2 always resolves a company on
	// login), kept as a defensive guard.
	ErrNoTenantContext      = errs.New("NO_TENANT_CONTEXT", "no tenant in request context")
	ErrAIConnectionNotFound = errs.New("AI_CONNECTION_NOT_FOUND", "ai connection not found")
	// ErrAPIKeyRequired guards providers other than Ollama, which needs no
	// credential (self-hosted, reachable via BaseURL alone).
	ErrAPIKeyRequired = errs.New("AI_API_KEY_REQUIRED", "an API key is required for this provider")
	// ErrBaseURLRequired guards AIProviderOllama, which has no fixed
	// default endpoint the way hosted providers do.
	ErrBaseURLRequired = errs.New("AI_BASE_URL_REQUIRED", "a base URL is required for this provider")
	// ErrInvalidBolDeliveryCode guards UpdateConnectionSettingsUseCase --
	// checked against ValidBolDeliveryCodes so a typo is caught at save
	// time instead of surfacing as an opaque Bol 400 on the next push.
	ErrInvalidBolDeliveryCode = errs.New("BOL_INVALID_DELIVERY_CODE", "invalid bol delivery code")
	// ErrInvalidBolShippingCost guards UpdateConnectionSettingsUseCase: the
	// per-parcel shipping cost can't be negative.
	ErrInvalidBolShippingCost = errs.New("BOL_INVALID_SHIPPING_COST", "bol shipping cost can't be negative")
	// ErrBolEconomicOperatorsRequestFailed wraps a live call to Bol's
	// Economic Operator API -- handlers map it to 502, not 500. Commonly a
	// 403 because the connection's app registration lacks the
	// economic-operators.read/.manage OAuth scope, not a bug in this codebase.
	ErrBolEconomicOperatorsRequestFailed = errs.NewWithCause("BOL_ECONOMIC_OPERATORS_REQUEST_FAILED", "failed to fetch economic operators from bol")
	// ErrBolRequestRejected means Bol refused a request as invalid (a 4xx
	// other than auth/rate limiting) -- sending it again won't help, so
	// callers record it rather than retry.
	ErrBolRequestRejected = errs.NewWithCause("BOL_REQUEST_REJECTED", "bol rejected the request")
	// ErrWebhookSecretNotSet guards the provider-webhooks admin feature's
	// create action (application/integrations.CreateProviderWebhookUseCase)
	// -- a subscription can't be created before the tenant has a webhook
	// secret provisioned (SetWebhookSecretUseCase), since the new
	// subscription needs to embed/sign with it (WooCommerce: request body
	// `secret` field; MyParcel: `?token=` query param -- see that use
	// case's doc comment for why both reuse the same
	// WebhookSecretEncrypted field despite verifying differently).
	ErrWebhookSecretNotSet = errs.New("CONNECTION_WEBHOOK_SECRET_NOT_SET", "integrations: connection has no webhook secret set yet")
	// ErrProviderWebhookRequestFailed distinguishes a live provider API
	// call failure (list/create/delete a webhook subscription) from an
	// internal error -- handlers map it to 502, not 500.
	ErrProviderWebhookRequestFailed = errs.NewWithCause("PROVIDER_WEBHOOK_REQUEST_FAILED", "provider webhook request failed")
	ErrEmailHostRequired            = errs.New("EMAIL_HOST_REQUIRED", "smtp host is required")
	ErrEmailPortRequired            = errs.New("EMAIL_PORT_REQUIRED", "smtp port is required")
	ErrEmailFromRequired            = errs.New("EMAIL_FROM_REQUIRED", "from address is required")
	// ErrWhatsAppTestPhoneRequired guards SetWhatsAppTestModeUseCase --
	// enabling test mode with no redirect number would silently drop every
	// outbound WhatsApp message instead of the intended "sent to me
	// instead of the customer" behavior.
	ErrWhatsAppTestPhoneRequired = errs.New("WHATSAPP_TEST_PHONE_REQUIRED", "a test phone number is required to enable whatsapp test mode")
	// ErrNotWhatsAppConnection guards SetWhatsAppTestModeUseCase against
	// being pointed at a connection of any other provider.
	ErrNotWhatsAppConnection = errs.New("NOT_A_WHATSAPP_CONNECTION", "connection is not a whatsapp connection")
)

// ErrPostNLAccountInvalid: connecting PostNL needs an API key, a customer
// code (1-4 characters) and a customer number (1-10 characters) -- the
// limits PostNL's Shipment v4 API enforces.
var ErrPostNLAccountInvalid = errs.New("POSTNL_ACCOUNT_INVALID", "PostNL needs an API key, a customer code of up to 4 characters and a customer number of up to 10 characters")

// ErrNotPostNLConnection: a PostNL-only operation was asked for a
// connection of another provider.
var ErrNotPostNLConnection = errs.New("NOT_POSTNL_CONNECTION", "this is not a PostNL connection")
