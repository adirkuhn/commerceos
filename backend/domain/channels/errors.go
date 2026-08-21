package channels

import "errors"

var (
	ErrListingNotFound = errors.New("product channel listing not found")
	// ErrCannotPushVariableProduct guards push/sync: every Variant being
	// pushed (the product's only Variant, for a single-SKU product, or
	// each one, for a multi-variant product in either Woo push mode) must
	// resolve to exactly one sellable SKU. A Variant with zero or 2+ SKUs
	// has no single price/stock source of truth to push.
	ErrCannotPushVariableProduct = errors.New("every variant must have exactly one SKU to be pushed to a channel")
	// ErrCannotPushBulkPoolSKU guards push/sync: a bulk pool (unit_type=
	// kilo) SKU is never itself sellable -- it's an internal weight ledger
	// (see pms.SKU.UnitType/PortionedFromSKUID), so it's always excluded
	// from what gets pushed. Returned only when EVERY variant being
	// pushed turns out to be a bulk pool, leaving nothing sellable.
	ErrCannotPushBulkPoolSKU = errors.New("product has no sellable (non-bulk-pool) variant to push to a channel")
	ErrInvalidSyncEntityType = errors.New("invalid sync entity type")
	ErrInvalidSyncDirection  = errors.New("invalid sync direction")
	ErrInvalidSyncInterval   = errors.New("sync interval must be a positive number of minutes")
	// ErrCannotEnableUnrunnableEntity guards SyncSchedule: categories is a
	// schema-legal entity type (so the schedule grid can show it as
	// "coming soon") but has no sync use case wired to run it yet, and
	// orders/export specifically (this codebase only ever imports Woo
	// orders, never pushes CommerceOS-created ones back) -- a schedule for
	// either can be saved only disabled.
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
	ErrSyncScheduleNotFound   = errors.New("sync schedule not found")
	ErrSyncLogNotFound        = errors.New("sync log entry not found")
	// ErrSyncLogHasNoListing guards Retry: a log entry whose listing was
	// since deleted (see SyncLogEntry.ListingID's doc) has nothing left to
	// retry against.
	ErrSyncLogHasNoListing   = errors.New("sync log entry's listing no longer exists")
	ErrSalesChannelNotFound  = errors.New("sales channel not found")
	ErrInvalidChannelType    = errors.New("invalid sales channel type")
	ErrInvalidEnrollmentMode = errors.New("invalid enrollment mode")
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
