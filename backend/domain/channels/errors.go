package channels

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrListingNotFound = errs.New("CHANNEL_LISTING_NOT_FOUND", "product channel listing not found")
	// ErrInvalidPaymentConnection guards SetSalesChannelPaymentMethodUseCase:
	// a channel's PaymentConnectionID must reference a connection whose
	// Provider is a payment integration (Stripe/Revolut) and whose Status
	// is Connected -- never a disconnected or non-payment connection.
	ErrInvalidPaymentConnection = errs.New("CHANNEL_PAYMENT_CONNECTION_INVALID", "payment_connection_id must reference a connected payment integration")
	// ErrCannotPushVariableProduct guards push/sync: every Variant being
	// pushed (the product's only Variant, for a single-SKU product, or
	// each one, for a multi-variant product in either Woo push mode) must
	// resolve to exactly one sellable SKU. A Variant with zero or 2+ SKUs
	// has no single price/stock source of truth to push.
	ErrCannotPushVariableProduct = errs.New("PUSH_VARIANT_NEEDS_ONE_SKU", "every variant must have exactly one SKU to be pushed to a channel")
	// ErrCannotPushBulkPoolSKU guards push/sync: a bulk pool (unit_type=
	// kilo) SKU is never itself sellable -- it's an internal weight ledger
	// (see pms.SKU.UnitType/PortionedFromSKUID), so it's always excluded
	// from what gets pushed. Returned only when EVERY variant being
	// pushed turns out to be a bulk pool, leaving nothing sellable.
	ErrCannotPushBulkPoolSKU = errs.New("PUSH_NO_SELLABLE_VARIANT", "product has no sellable (non-bulk-pool) variant to push to a channel")
	ErrInvalidSyncEntityType = errs.New("SYNC_ENTITY_TYPE_INVALID", "invalid sync entity type")
	ErrInvalidSyncDirection  = errs.New("SYNC_DIRECTION_INVALID", "invalid sync direction")
	ErrInvalidSyncInterval   = errs.New("SYNC_INTERVAL_INVALID", "sync interval must be a positive number of minutes")
	// ErrCannotEnableUnrunnableEntity guards SyncSchedule: categories is a
	// schema-legal entity type (so the schedule grid can show it as
	// "coming soon") but has no sync use case wired to run it yet, and
	// orders/export specifically (this codebase only ever imports Woo
	// orders, never pushes CommerceOS-created ones back) -- a schedule for
	// either can be saved only disabled.
	ErrCannotEnableUnrunnableEntity = errs.New("SYNC_ENTITY_NOT_SCHEDULABLE", "this entity type does not support scheduled sync yet")
	// ErrDefaultWarehouseAndCurrencyRequired guards the products/import
	// schedule specifically: Woo has no warehouse concept and its product
	// endpoint doesn't return a currency, so ImportWooProductsUseCase
	// can't run without both supplied.
	ErrDefaultWarehouseAndCurrencyRequired = errs.New("SYNC_PRODUCT_IMPORT_NEEDS_DEFAULTS", "default warehouse and currency are required to enable scheduled product import")
	// ErrDefaultWarehouseRequired guards the stock/import schedule: stock
	// reconciliation adjustments need a warehouse to attribute the
	// movement to.
	ErrDefaultWarehouseRequired = errs.New("SYNC_STOCK_IMPORT_NEEDS_WAREHOUSE", "default warehouse is required to enable scheduled stock import")
	// ErrImportJobAlreadyActive guards manual import: at most one
	// pending/running ImportJob per connection — returned both by the
	// proactive check and as the mapped form of the DB's unique-index
	// backstop if two requests race.
	ErrImportJobAlreadyActive = errs.New("IMPORT_ALREADY_RUNNING", "an import is already running for this connection")
	ErrImportJobNotFound      = errs.New("IMPORT_JOB_NOT_FOUND", "import job not found")
	ErrSyncScheduleNotFound   = errs.New("SYNC_SCHEDULE_NOT_FOUND", "sync schedule not found")
	ErrSyncLogNotFound        = errs.New("SYNC_LOG_NOT_FOUND", "sync log entry not found")
	// ErrSyncLogHasNoListing guards Retry: a log entry whose listing was
	// since deleted (see SyncLogEntry.ListingID's doc) has nothing left to
	// retry against.
	ErrSyncLogHasNoListing   = errs.New("SYNC_LOG_HAS_NO_LISTING", "sync log entry's listing no longer exists")
	ErrSalesChannelNotFound  = errs.New("SALES_CHANNEL_NOT_FOUND", "sales channel not found")
	ErrInvalidChannelType    = errs.New("SALES_CHANNEL_TYPE_INVALID", "invalid sales channel type")
	ErrInvalidEnrollmentMode = errs.New("SALES_CHANNEL_ENROLLMENT_MODE_INVALID", "invalid enrollment mode")
	// ErrInvalidPriceAdjustment guards SalesChannel.PriceAdjustmentBasisPoints
	// against an out-of-range value -- 0-10000 (0%-100% discount) is the
	// only valid range.
	ErrInvalidPriceAdjustment = errs.New("SALES_CHANNEL_PRICE_ADJUSTMENT_INVALID", "price adjustment must be between 0 and 10000 basis points")
	// ErrInvalidQuantityBreak guards SalesChannel.QuantityBreaks: every
	// MinQuantity must be positive, every DiscountBasisPoints 0-10000, and
	// no two breaks may share the same MinQuantity (which one would even
	// apply is undefined otherwise).
	ErrInvalidQuantityBreak = errs.New("SALES_CHANNEL_QUANTITY_BREAK_INVALID", "each quantity break needs a positive, unique minimum quantity and a discount between 0 and 10000 basis points")
	// ErrConnectionRequired guards a channel type where
	// ChannelType.RequiresConnection() is true (every type implemented
	// today) against being created/updated with no ConnectionID.
	ErrConnectionRequired = errs.New("SALES_CHANNEL_CONNECTION_REQUIRED", "a connection is required for this channel type")
	// ErrEnrollmentCategoryRequired/ErrEnrollmentTagRequired guard
	// EnrollmentModeCategory/EnrollmentModeTag against being set with no
	// category/tag to match against.
	ErrEnrollmentCategoryRequired   = errs.New("SALES_CHANNEL_ENROLLMENT_CATEGORY_REQUIRED", "a category is required for category-based enrollment")
	ErrEnrollmentTagRequired        = errs.New("SALES_CHANNEL_ENROLLMENT_TAG_REQUIRED", "a tag is required for tag-based enrollment")
	ErrExternalCategoryLinkNotFound = errs.New("EXTERNAL_CATEGORY_LINK_NOT_FOUND", "external category link not found")
	// ErrBolMissingEAN guards PushProductToBolUseCase: Bol requires an EAN
	// on every offer (pms.SKU.Barcode, or BolProductOverride.EANOverride)
	// -- there's no "unknown EAN" path for an offer that has no EAN at
	// all, only for one whose EAN isn't yet in Bol's catalog.
	ErrBolMissingEAN = errs.New("BOL_MISSING_EAN", "product's SKU has no barcode/EAN set, required to push to Bol")
	// ErrBolCommissionRequestFailed wraps a live call to Bol's Commissions
	// API that failed (network error, non-2xx, auth failure) -- mapped to
	// 502 UPSTREAM_ERROR, same treatment as
	// integrationsDomain.ErrWooRequestFailed, since it's Bol's API that's
	// unavailable, not a client mistake.
	ErrBolCommissionRequestFailed = errs.NewWithCause("BOL_COMMISSION_REQUEST_FAILED", "failed to fetch commission from Bol")
	// ErrChannelTypeConnectionMismatch guards Create/UpdateSalesChannelUseCase:
	// a channel's Type must match its Connection's actual Provider -- a
	// SalesChannel pointed at a Bol connection but typed 'woocommerce' (or
	// vice versa) silently never matches in EnrollProductInChannelsUseCase/
	// RunScheduledBolExportUseCase's `ch.Type == ChannelTypeBol` checks,
	// which is exactly how this went unnoticed: the channel looked
	// correctly configured in the UI (right name, right connection) but
	// its Type was wrong.
	ErrChannelTypeConnectionMismatch = errs.New("CHANNEL_TYPE_CONNECTION_MISMATCH", "channel type does not match the connection's provider")
	// ErrProductAlreadyLinkedToConnection guards LinkExistingWooProductUseCase
	// -- the manual "this CommerceOS product is actually this Woo product"
	// admin action refuses to run against a product that already has a
	// product_channel_listings row for the given connection, since that
	// would create a second, conflicting row nothing else expects to see
	// (see the table's own doc: only (connection_id, external_id) is
	// actually unique at the DB level -- app logic is what prevents two
	// rows for the same product+connection otherwise).
	ErrProductAlreadyLinkedToConnection = errs.New("PRODUCT_ALREADY_LINKED_TO_CONNECTION", "this product already has a listing for this connection")
	// ErrProductNotSyncable guards every initial channel push (Woo and Bol):
	// a Draft/Archived product must never be created on an external store.
	// An already-linked product leaving Active is handled differently -- see
	// pms.Product.IsSyncableToChannels and
	// application/channels.SyncProductToChannelsUseCase's gating, not this
	// error.
	ErrProductNotSyncable = errs.New("PRODUCT_NOT_SYNCABLE", "product is not active, cannot push to a sales channel")
	// ErrInvalidCurrencyCode guards SalesChannel.CurrencyCode against a
	// malformed value -- "" (meaning "use the tenant's default currency")
	// is always valid; a non-empty value must pass fx.ValidCurrencyCode.
	ErrInvalidCurrencyCode = errs.New("SALES_CHANNEL_CURRENCY_INVALID", "currency code must be 3 uppercase letters")
	// ErrChannelCurrencyRequiresRate guards CreateSalesChannelUseCase: a
	// non-default CurrencyCode needs a fx.ExchangeRate already configured
	// for it, since that's the fallback every SKU/coupon without an
	// explicit override in that currency will be priced through.
	ErrChannelCurrencyRequiresRate = errs.New("CHANNEL_CURRENCY_REQUIRES_RATE", "no exchange rate is configured for this currency yet")
)
