package channels

import "errors"

var (
	ErrListingNotFound = errors.New("product channel listing not found")
	// ErrCannotPushVariableProduct guards push-to-channel: only single-SKU
	// (has_variants=false) products map onto a WooCommerce "simple"
	// product today. Pushing a multi-variant product would need Woo
	// "variable" products + per-variation calls — not built yet.
	ErrCannotPushVariableProduct = errors.New("products with variants cannot be pushed to a channel yet")
)
