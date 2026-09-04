package settings

import "time"

// TenantSettings holds tenant-wide localization defaults. A tenant with no
// row yet is represented by a zero-value TenantSettings (empty strings) —
// there is no ErrNotFound; Get always succeeds.
type TenantSettings struct {
	TenantID        string
	DefaultCurrency string
	Timezone        string
	DateFormat      string
	Language        string
	// PricesIncludeTax and DefaultTaxCountry configure how SKU prices and
	// manually-created Sales Orders are taxed -- see domain/tax.TaxClass.
	// Like every other field here, the zero value (false / "") on a
	// never-saved tenant means "unset", not "explicitly no tax" -- callers
	// needing a true default before any settings are ever saved (this
	// tenant's chosen default is tax-inclusive) check UpdatedAt.IsZero()
	// the same way the frontend already falls back for DefaultCurrency.
	PricesIncludeTax  bool
	DefaultTaxCountry string
	// InventorySyncRulesEnabled toggles the batch-expiry-aware channel sync
	// strategy (see application/inventory.computeStockLevel): off keeps
	// today's flat sum-of-non-expired-batches behavior. BatchExpiryThresholdDays
	// is the age (days past ExpiryDate) at which a batch is auto-flagged
	// expired by the scheduler job; 0 disables auto-expiry.
	InventorySyncRulesEnabled bool
	BatchExpiryThresholdDays  int
	// DefaultTargetMarkupBasisPoints is the tenant-wide fallback target
	// markup (profit / cost) used to compute a SKU's recommended price when
	// its category has no override (see pms.Category.TargetMarkupBasisPoints
	// and application/pms.EffectiveTargetMarkupBasisPoints). Basis points,
	// no upper bound (unlike margin, markup can exceed 100%); 0 on a
	// never-saved tenant means "unset", same zero-value convention as every
	// other field here -- callers needing a real default before any
	// settings are ever saved check UpdatedAt.IsZero().
	DefaultTargetMarkupBasisPoints int
	// BundleConfidenceThreshold is the minimum confidence (0-1, the
	// fraction of orders containing a cart item that also contain the
	// candidate) a cross-sell recommendation must clear before
	// application/reports.GetCartRecommendationsUseCase flags it
	// IsBundleCandidate for a storefront -- CommerceOS only signals the
	// flag; the discount policy itself stays the storefront's own. Same
	// zero-value-means-unset convention as every other field here: on a
	// never-saved tenant this is 0, and GetCartRecommendationsUseCase
	// falls back to its own documented default rather than treating an
	// explicit 0 (flag every candidate) as a real merchant choice.
	BundleConfidenceThreshold float64
	UpdatedAt                 time.Time
}
