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
	UpdatedAt         time.Time
}
