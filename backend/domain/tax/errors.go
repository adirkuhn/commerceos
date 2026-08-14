package tax

import "errors"

var (
	ErrTaxClassNotFound     = errors.New("tax class not found")
	ErrTaxRateNotFound      = errors.New("tax rate not found")
	ErrTaxClassNameRequired = errors.New("tax class name is required")
	// ErrCannotDeleteDefaultTaxClass guards TaxClass deletion: a SKU with no
	// tax_class_id set falls back to whichever class IsDefault=true, so
	// deleting the default out from under every unassigned SKU would leave
	// them with no resolvable class at all.
	ErrCannotDeleteDefaultTaxClass = errors.New("cannot delete the default tax class")
	// ErrTaxClassInUse guards TaxClass deletion: at least one SKU
	// explicitly references this class by ID.
	ErrTaxClassInUse       = errors.New("tax class is assigned to one or more SKUs")
	ErrInvalidCountryCode  = errors.New("country must be a 2-letter ISO code")
	ErrInvalidTaxRateValue = errors.New("tax rate must not be negative")

	ErrExternalTaxClassLinkNotFound = errors.New("external tax class link not found")
	// ErrProviderRequired guards Provider only -- ExternalSlug legitimately
	// can be "" (e.g. WooCommerce's own Standard rate has an empty
	// tax_class slug), so it has no required-non-empty check.
	ErrProviderRequired = errors.New("provider is required")
)
