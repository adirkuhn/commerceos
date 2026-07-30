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
	ErrNoTenantContext = errors.New("no tenant in request context")
)
