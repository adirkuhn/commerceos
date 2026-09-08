package channels

import "errors"

var (
	ErrListingNotFound = errors.New("product channel listing not found")
	// ErrInvalidPaymentConnection guards SetSalesChannelPaymentMethodUseCase:
	// a channel's PaymentConnectionID must reference a connection whose
	// Provider is a payment integration (Stripe/Revolut) and whose Status
	// is Connected -- never a disconnected or non-payment connection.
	ErrInvalidPaymentConnection = errors.New("payment_connection_id must reference a connected payment integration")
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
	// ErrInvalidPriceAdjustment guards SalesChannel.PriceAdjustmentBasisPoints
	// against an out-of-range value -- 0-10000 (0%-100% discount) is the
	// only valid range.
	ErrInvalidPriceAdjustment = errors.New("price adjustment must be between 0 and 10000 basis points")
	// ErrConnectionRequired guards a channel type where
	// ChannelType.RequiresConnection() is true (every type implemented
	// today) against being created/updated with no ConnectionID.
	ErrConnectionRequired = errors.New("a connection is required for this channel type")
	// ErrEnrollmentCategoryRequired/ErrEnrollmentTagRequired guard
	// EnrollmentModeCategory/EnrollmentModeTag against being set with no
	// category/tag to match against.
	ErrEnrollmentCategoryRequired   = errors.New("a category is required for category-based enrollment")
	ErrEnrollmentTagRequired        = errors.New("a tag is required for tag-based enrollment")
	ErrExternalCategoryLinkNotFound = errors.New("external category link not found")
	// ErrBolMissingEAN guards PushProductToBolUseCase: Bol requires an EAN
	// on every offer (pms.SKU.Barcode, or BolProductOverride.EANOverride)
	// -- there's no "unknown EAN" path for an offer that has no EAN at
	// all, only for one whose EAN isn't yet in Bol's catalog.
	ErrBolMissingEAN = errors.New("product's SKU has no barcode/EAN set, required to push to Bol")
	// ErrBolCommissionRequestFailed wraps a live call to Bol's Commissions
	// API that failed (network error, non-2xx, auth failure) -- mapped to
	// 502 UPSTREAM_ERROR, same treatment as
	// integrationsDomain.ErrWooRequestFailed, since it's Bol's API that's
	// unavailable, not a client mistake.
	ErrBolCommissionRequestFailed = errors.New("failed to fetch commission from Bol")
	// ErrChannelTypeConnectionMismatch guards Create/UpdateSalesChannelUseCase:
	// a channel's Type must match its Connection's actual Provider -- a
	// SalesChannel pointed at a Bol connection but typed 'woocommerce' (or
	// vice versa) silently never matches in EnrollProductInChannelsUseCase/
	// RunScheduledBolExportUseCase's `ch.Type == ChannelTypeBol` checks,
	// which is exactly how this went unnoticed: the channel looked
	// correctly configured in the UI (right name, right connection) but
	// its Type was wrong.
	ErrChannelTypeConnectionMismatch = errors.New("channel type does not match the connection's provider")
	// ErrProductAlreadyLinkedToConnection guards LinkExistingWooProductUseCase
	// -- the manual "this CommerceOS product is actually this Woo product"
	// admin action refuses to run against a product that already has a
	// product_channel_listings row for the given connection, since that
	// would create a second, conflicting row nothing else expects to see
	// (see the table's own doc: only (connection_id, external_id) is
	// actually unique at the DB level -- app logic is what prevents two
	// rows for the same product+connection otherwise).
	ErrProductAlreadyLinkedToConnection = errors.New("this product already has a listing for this connection")
	// ErrProductNotSyncable guards every initial channel push (Woo and Bol):
	// a Draft/Archived product must never be created on an external store.
	// An already-linked product leaving Active is handled differently -- see
	// pms.Product.IsSyncableToChannels and
	// application/channels.SyncProductToChannelsUseCase's gating, not this
	// error.
	ErrProductNotSyncable = errors.New("product is not active, cannot push to a sales channel")
)
