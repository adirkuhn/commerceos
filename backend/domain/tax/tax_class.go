package tax

import "time"

// TaxClass groups a set of per-country rates under one name (e.g.
// "Standard", "Reduced rate", "Zero rate", or a tenant's own custom class)
// -- mirrors WooCommerce's tax class model. A pms.SKU references one by ID;
// a SKU with no class set uses whichever TaxClass has IsDefault=true for
// the tenant, so the common case (every product taxed the same way) needs
// no per-SKU assignment at all.
type TaxClass struct {
	ID        string
	TenantID  string
	Name      string
	IsDefault bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TaxRate is one TaxClass's rate for one country. RateBasisPoints is an
// integer (1 bp = 0.01%, e.g. 2100 = 21.00%) rather than a float, same
// rationale as this codebase's *MinorUnits money fields -- a value that
// ends up multiplied into money shouldn't carry float rounding error.
type TaxRate struct {
	ID              string
	TenantID        string
	TaxClassID      string
	Country         string
	RateBasisPoints int32
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
