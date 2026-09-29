package channels

import "time"

type ChannelType string

const (
	ChannelTypeWooCommerce ChannelType = "woocommerce"
	// ChannelTypeBol is the Bol.com marketplace adapter (Offer API v11 +
	// Product Content API v10) -- see application/channels.PushProductToBolUseCase.
	ChannelTypeBol ChannelType = "bol"
	// ChannelTypeInternal is CommerceOS' own storefront channel -- served
	// headless over /storefront/v1, or additionally as a hosted themed
	// storefront (see StorefrontMode).
	ChannelTypeInternal ChannelType = "internal"
	// ChannelTypeWhatsApp gates which products a customer can query
	// (stock/price/expiry) or order over the WhatsApp integration --
	// enrollment here reuses the exact same EnrollmentMode/override
	// mechanism as every other channel, see
	// application/whatsapp.LookupProductInfoUseCase.
	ChannelTypeWhatsApp ChannelType = "whatsapp"
)

func (t ChannelType) Valid() bool {
	switch t {
	case ChannelTypeWooCommerce, ChannelTypeBol, ChannelTypeInternal, ChannelTypeWhatsApp:
		return true
	default:
		return false
	}
}

// RequiresConnection reports whether this channel type pushes through an
// integrations.Connection -- true for every channel type implemented today.
// A future built-in storefront could be its own product's source of truth
// instead, hence the type already allows a nil ConnectionID for it.
func (t ChannelType) RequiresConnection() bool {
	return t != ChannelTypeInternal
}

type EnrollmentMode string

const (
	EnrollmentModeAll      EnrollmentMode = "all"
	EnrollmentModeCategory EnrollmentMode = "category"
	EnrollmentModeTag      EnrollmentMode = "tag"
	EnrollmentModeManual   EnrollmentMode = "manual"
)

func (m EnrollmentMode) Valid() bool {
	switch m {
	case EnrollmentModeAll, EnrollmentModeCategory, EnrollmentModeTag, EnrollmentModeManual:
		return true
	default:
		return false
	}
}

// SalesChannel is the "which products belong here" layer on top of an
// integrations.Connection (which stays the "how to push" layer, unchanged).
// A product is enrolled in a channel when it matches EnrollmentMode's rule,
// unless a SalesChannelProductOverride exists for that (channel, product)
// pair, which always wins over the default rule. Enrollment is never
// stored per product -- it's evaluated live (see application/channels'
// isEnrolled) so changing a channel's rule takes effect immediately for
// every matching product, past and future, with no backfill.
type SalesChannel struct {
	ID       string
	TenantID string
	Name     string
	Type     ChannelType
	// ConnectionID is required when Type.RequiresConnection() is true
	// (every type implemented today).
	ConnectionID   *string
	EnrollmentMode EnrollmentMode
	// EnrollmentCategoryID is used (and required) only when EnrollmentMode
	// is EnrollmentModeCategory.
	EnrollmentCategoryID *string
	// EnrollmentTag is used (and required) only when EnrollmentMode is
	// EnrollmentModeTag. Single tag by design -- anything more specific
	// than "one tag" goes through a manual per-product override instead.
	EnrollmentTag string
	// PaymentConnectionID references an integrations.Connection whose
	// Provider is a payment integration (Stripe/Revolut -- see
	// integrations.Provider.IsPaymentProvider) -- which one this channel
	// generates a payment link through when an order is placed on it (see
	// application/whatsapp's order-placement flow). nil means "no payment
	// method enabled" -- "disabling" the
	// payment method is exactly this field going back to nil, "enabling"
	// is setting it to one of the tenant's connected payment integrations.
	// Meaningful only for ChannelTypeWhatsApp and ChannelTypeInternal today
	// (both generate their own checkout payment link), but kept generic on
	// SalesChannel rather than type-specific, same convention as
	// Connection's Bol-only fields.
	PaymentConnectionID *string
	// PriceAdjustmentBasisPoints discounts every SKU's effective price when
	// shown/sold through this channel -- nil means no adjustment (retail
	// price as-is). Basis points as a DISCOUNT off list price (e.g. 2000 =
	// 20% off), same convention as tax.TaxRate.RateBasisPoints: an integer
	// so a value that ends up multiplied into money never carries float
	// rounding error. Meaningful only for ChannelTypeInternal today -- this
	// is what lets a tenant run a B2B storefront at wholesale prices
	// alongside a B2C one at retail, without any customer-group/login
	// concept (see application/storefront, the only reader). Valid range
	// 0-10000 (0%-100%), enforced at write time (see
	// application/channels.validateSalesChannelInput).
	PriceAdjustmentBasisPoints *int
	// QuantityBreaks are additional per-line-quantity discount tiers on top
	// of PriceAdjustmentBasisPoints -- "buy N+ of one SKU, get this bigger
	// discount instead of the channel's flat rate." A short list wholly
	// owned by the channel (no independent existence, no ordering that
	// matters), same JSONB whole-document convention as
	// domain/storefront.CartItem. Meaningful only for ChannelTypeInternal
	// today, same as PriceAdjustmentBasisPoints -- this is the other half
	// of "B2B/wholesale pricing" alongside the flat per-channel discount:
	// a bulk buyer gets a bigger cut the more of one SKU they add to a
	// single cart line, still with no customer-group/login concept.
	QuantityBreaks []QuantityBreak
	// CurrencyCode is this channel's fixed selling currency -- "" means
	// "use the tenant's default currency" (settings.TenantSettings), which
	// is what every channel had implicitly before multi-currency existed
	// and is still the only behavior for every channel that doesn't opt
	// in. Set once at creation (see
	// application/channels.CreateSalesChannelUseCase) and immutable after
	// -- changing a live channel's currency out from under carts/orders
	// already priced in the old one would corrupt them, so re-pricing a
	// channel means creating a new one. A non-"" value requires a
	// fx.ExchangeRate already configured for it (checked at creation, not
	// here, since that's a cross-domain lookup) -- see
	// application/pms.ResolvePriceForCurrency for how a SKU/coupon's price
	// actually gets computed in this currency.
	CurrencyCode string
	// DefaultWarehouseID is which CommerceOS warehouse this channel's
	// placed orders attribute their stock decrement to -- mirrors
	// integrations.Connection.DefaultWarehouseID (used by WooCommerce/Bol
	// imports), but for a channel type with no connection to hang it off
	// (ChannelTypeInternal, ChannelTypeWhatsApp). sales_orders.warehouse_id
	// is a NOT NULL FK, so a channel of either type can't actually process
	// an order until this is set -- see
	// application/storefront.CheckoutCartUseCase and
	// application/whatsapp.OrderFlowUseCase, the two callers that resolve
	// it and fail clearly (not a raw DB error) when it's nil.
	DefaultWarehouseID *string
	// StorefrontMode -- see StorefrontMode's own doc comment.
	StorefrontMode StorefrontMode
	// SignInURL is a headless storefront's page that completes shopper
	// sign-in (see ResolveSignInURL); nil when not configured.
	SignInURL *string
	// StorefrontLocale -- see StorefrontLocale's own doc comment. The
	// storefront's default language.
	StorefrontLocale StorefrontLocale
	// StorefrontLocales is every language the storefront offers, default
	// first (see ResolveStorefrontLocales); read it through
	// OfferedStorefrontLocales, which also covers channels saved before
	// this list existed.
	StorefrontLocales []StorefrontLocale
	Active            bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// QuantityBreak is one "at this quantity or more, this bigger discount"
// tier -- see SalesChannel.QuantityBreaks.
type QuantityBreak struct {
	MinQuantity         int64
	DiscountBasisPoints int
}

// EffectivePriceMinorUnits applies PriceAdjustmentBasisPoints (nil = no
// adjustment) to priceMinorUnits, rounding down -- the one place this
// channel's flat discount is actually computed, called for every price a
// storefront shows before a cart quantity is known (product listing/
// detail pages -- see application/storefront). Cart/checkout pricing,
// where the actual line quantity IS known, uses
// EffectivePriceMinorUnitsForQuantity instead.
func (c SalesChannel) EffectivePriceMinorUnits(priceMinorUnits int64) int64 {
	if c.PriceAdjustmentBasisPoints == nil || *c.PriceAdjustmentBasisPoints == 0 {
		return priceMinorUnits
	}
	return priceMinorUnits * (10000 - int64(*c.PriceAdjustmentBasisPoints)) / 10000
}

// EffectivePriceMinorUnitsForQuantity picks the BEST (largest) discount
// between the channel's flat PriceAdjustmentBasisPoints and any
// QuantityBreak whose MinQuantity is met by quantity -- a bulk tier is
// always an upgrade over the flat rate, never a worse price than it.
func (c SalesChannel) EffectivePriceMinorUnitsForQuantity(priceMinorUnits int64, quantity int64) int64 {
	best := 0
	if c.PriceAdjustmentBasisPoints != nil {
		best = *c.PriceAdjustmentBasisPoints
	}
	for _, b := range c.QuantityBreaks {
		if quantity >= b.MinQuantity && b.DiscountBasisPoints > best {
			best = b.DiscountBasisPoints
		}
	}
	if best == 0 {
		return priceMinorUnits
	}
	return priceMinorUnits * (10000 - int64(best)) / 10000
}

// SalesChannelProductOverride force-includes or force-excludes one product
// from one channel regardless of the channel's default EnrollmentMode rule
// -- this is what "add"/"remove" a single product means; "add all"/"add by
// tag"/"add by category" instead change the channel's own EnrollmentMode
// (see UpdateSalesChannelUseCase), needing no per-product rows at all.
type SalesChannelProductOverride struct {
	ID        string
	TenantID  string
	ChannelID string
	ProductID string
	Included  bool
	CreatedAt time.Time
}
