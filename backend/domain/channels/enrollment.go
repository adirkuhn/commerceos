package channels

import "slices"

// IsProductEnrolled decides whether a product belongs on this channel.
// Deliberately takes the product's category/tags as primitives rather than
// a pms.Product, to keep this package decoupled from pms (see the package
// doc comment). override, when non-nil, always wins over the default
// EnrollmentMode rule -- that's what a manual "add"/"remove" action means.
func (c SalesChannel) IsProductEnrolled(productCategoryID *string, productTags []string, override *SalesChannelProductOverride) bool {
	if override != nil {
		return override.Included
	}
	switch c.EnrollmentMode {
	case EnrollmentModeAll:
		return true
	case EnrollmentModeCategory:
		return c.EnrollmentCategoryID != nil && productCategoryID != nil && *c.EnrollmentCategoryID == *productCategoryID
	case EnrollmentModeTag:
		return c.EnrollmentTag != "" && slices.Contains(productTags, c.EnrollmentTag)
	case EnrollmentModeManual:
		return false
	default:
		return false
	}
}
