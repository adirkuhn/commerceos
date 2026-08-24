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
)
