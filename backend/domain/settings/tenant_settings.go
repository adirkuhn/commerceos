package settings

import (
	"time"

	"github.com/adirkuhn/commerceos/domain/locale"
)

// TenantSettings holds tenant-wide localization defaults. A tenant with no
// row yet is represented by a zero-value TenantSettings (empty strings) —
// there is no ErrNotFound; Get always succeeds.
type TenantSettings struct {
	TenantID        string
	DefaultCurrency string
	Timezone        string
	DateFormat      DateFormat
	// Language is the company default: the admin language for users who
	// picked none, and the storefront language new sales channels start
	// with. "" on a never-saved tenant means English.
	Language locale.Locale
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
	// RecommendationsMinSupport is the minimum number of orders an itemset
	// must appear in before application/reports.GetCartRecommendationsUseCase
	// treats it as a real pattern rather than coincidence -- FP-Growth's
	// own minSupport threshold. Quality lever, not a speed one: Apriori and
	// FP-Growth mine identical itemsets for the same minSupport, this is
	// what actually separates a real cross-sell signal from two unrelated
	// shoppers who happened to buy the same two things once. Same zero-
	// value-means-unset convention as every other field here: 0 on a
	// never-saved tenant means "unset", and GetCartRecommendationsUseCase
	// falls back to its own documented default rather than treating an
	// explicit 0 (every itemset counts, however rare) as a real merchant
	// choice -- FP-Growth itself also floors any value below 1 to 1.
	RecommendationsMinSupport int
	// RecommendationsLookbackDays is how far back
	// application/reports.GetCartRecommendationsUseCase looks for order
	// history when mining cross-sell itemsets -- a seasonal or slow-moving
	// catalog benefits from a longer window (more data to find real
	// patterns in), a fast-moving one from a shorter window (recent
	// behavior, not stale patterns from a since-discontinued line). Same
	// zero-value-means-unset convention as every other field here: 0 on a
	// never-saved tenant means "unset", and GetCartRecommendationsUseCase
	// falls back to its own documented default rather than treating an
	// explicit 0 (no history at all) as a real merchant choice.
	RecommendationsLookbackDays int
	// PickingMode is how the picking module works -- see PickingMode.
	PickingMode PickingMode
	// ExpiryAlert configures the daily stock-close-to-expiry report -- see
	// ExpiryAlert.
	ExpiryAlert ExpiryAlert
	UpdatedAt   time.Time
}
