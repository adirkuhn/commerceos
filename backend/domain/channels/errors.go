package channels

import "errors"

var (
	ErrListingNotFound = errors.New("product channel listing not found")
	// ErrCannotPushVariableProduct guards push-to-channel: only single-SKU
	// (has_variants=false) products map onto a WooCommerce "simple"
	// product today. Pushing a multi-variant product would need Woo
	// "variable" products + per-variation calls — not built yet.
	ErrCannotPushVariableProduct = errors.New("products with variants cannot be pushed to a channel yet")
	ErrInvalidSyncEntityType     = errors.New("invalid sync entity type")
	ErrInvalidSyncDirection      = errors.New("invalid sync direction")
	ErrInvalidSyncInterval       = errors.New("sync interval must be a positive number of minutes")
	// ErrCannotEnableUnrunnableEntity guards SyncSchedule: orders/categories
	// are schema-legal entity types (so the schedule grid can show them as
	// "coming soon") but have no sync use case wired to run them yet — a
	// schedule for them can be saved only disabled.
	ErrCannotEnableUnrunnableEntity = errors.New("this entity type does not support scheduled sync yet")
	// ErrDefaultWarehouseAndCurrencyRequired guards the products/import
	// schedule specifically: Woo has no warehouse concept and its product
	// endpoint doesn't return a currency, so ImportWooProductsUseCase
	// can't run without both supplied.
	ErrDefaultWarehouseAndCurrencyRequired = errors.New("default warehouse and currency are required to enable scheduled product import")
	// ErrDefaultWarehouseRequired guards the stock/import schedule: stock
	// reconciliation adjustments need a warehouse to attribute the
	// movement to.
	ErrDefaultWarehouseRequired = errors.New("default warehouse is required to enable scheduled stock import")
	// ErrImportJobAlreadyActive guards manual import: at most one
	// pending/running ImportJob per connection — returned both by the
	// proactive check and as the mapped form of the DB's unique-index
	// backstop if two requests race.
	ErrImportJobAlreadyActive = errors.New("an import is already running for this connection")
	ErrImportJobNotFound      = errors.New("import job not found")
	ErrSalesChannelNotFound   = errors.New("sales channel not found")
	ErrInvalidChannelType     = errors.New("invalid sales channel type")
	ErrInvalidEnrollmentMode  = errors.New("invalid enrollment mode")
	// ErrConnectionRequired guards a channel type where
	// ChannelType.RequiresConnection() is true (every type implemented
	// today) against being created/updated with no ConnectionID.
	ErrConnectionRequired = errors.New("a connection is required for this channel type")
	// ErrEnrollmentCategoryRequired/ErrEnrollmentTagRequired guard
	// EnrollmentModeCategory/EnrollmentModeTag against being set with no
	// category/tag to match against.
	ErrEnrollmentCategoryRequired = errors.New("a category is required for category-based enrollment")
	ErrEnrollmentTagRequired      = errors.New("a tag is required for tag-based enrollment")
)
