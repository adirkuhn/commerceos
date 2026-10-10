package tax

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrTaxClassNotFound     = errs.New("TAX_CLASS_NOT_FOUND", "tax class not found")
	ErrTaxRateNotFound      = errs.New("TAX_RATE_NOT_FOUND", "tax rate not found")
	ErrTaxClassNameRequired = errs.New("TAX_CLASS_NAME_REQUIRED", "tax class name is required")
	// ErrCannotDeleteDefaultTaxClass guards TaxClass deletion: a SKU with no
	// tax_class_id set falls back to whichever class IsDefault=true, so
	// deleting the default out from under every unassigned SKU would leave
	// them with no resolvable class at all.
	ErrCannotDeleteDefaultTaxClass = errs.New("TAX_CLASS_DEFAULT_UNDELETABLE", "cannot delete the default tax class")
	// ErrTaxClassInUse guards TaxClass deletion: at least one SKU
	// explicitly references this class by ID.
	ErrTaxClassInUse       = errs.New("TAX_CLASS_IN_USE", "tax class is assigned to one or more SKUs")
	ErrInvalidCountryCode  = errs.New("TAX_RATE_INVALID_COUNTRY", "country must be a 2-letter ISO code")
	ErrInvalidTaxRateValue = errs.New("TAX_RATE_NEGATIVE", "tax rate must not be negative")

	ErrExternalTaxClassLinkNotFound = errs.New("EXTERNAL_TAX_CLASS_LINK_NOT_FOUND", "external tax class link not found")
	// ErrProviderRequired guards Provider only -- ExternalSlug legitimately
	// can be "" (e.g. WooCommerce's own Standard rate has an empty
	// tax_class slug), so it has no required-non-empty check.
	ErrProviderRequired = errs.New("TAX_PROVIDER_REQUIRED", "provider is required")
)
