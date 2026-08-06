package channels

import "time"

type ChannelType string

const (
	ChannelTypeWooCommerce ChannelType = "woocommerce"
	// ChannelTypeInternal is reserved for a future built-in CommerceOS
	// storefront -- no push adapter exists for it yet, but the type is
	// defined now so adding it later doesn't need a SalesChannel redesign.
	ChannelTypeInternal ChannelType = "internal"
)

func (t ChannelType) Valid() bool {
	switch t {
	case ChannelTypeWooCommerce, ChannelTypeInternal:
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
	Active        bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
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
