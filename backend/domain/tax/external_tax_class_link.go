package tax

import "time"

// ExternalTaxClassLink maps one external channel's own tax-class identifier
// (e.g. WooCommerce's tax_class slug -- "" means its Standard rate) to one
// of this tenant's TaxClass rows. Import auto-creates one the first time it
// sees a new external slug; staff can repoint ExternalTaxClassLink.TaxClassID
// afterward without touching the class itself (see Settings > Tax's
// external-links section).
type ExternalTaxClassLink struct {
	ID       string
	TenantID string
	// Provider identifies the source channel -- "woocommerce" today, kept
	// as its own field (not hardcoded) so a future channel's own tax
	// classes can share this same table.
	Provider     string
	ExternalSlug string
	TaxClassID   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
