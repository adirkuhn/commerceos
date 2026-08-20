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
	// DefaultTargetMarginBasisPoints is the tenant-wide fallback target
	// margin used to compute a SKU's recommended price when its category
	// has no override (see pms.Category.TargetMarginBasisPoints and
	// application/pms.EffectiveTargetMarginBasisPoints). Basis points
	// (0-9999); 0 on a never-saved tenant means "unset", same zero-value
	// convention as every other field here -- callers needing a real
	// default before any settings are ever saved check UpdatedAt.IsZero().
	DefaultTargetMarginBasisPoints int
	UpdatedAt                      time.Time
}
