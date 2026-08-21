package pms

import pmsDomain "github.com/adirkuhn/commerceos/domain/pms"

// EffectiveTargetMarkupBasisPoints resolves the markup used to compute a
// product's recommended price: its category's override when one is set,
// otherwise the tenant-wide default. category may be nil (product has no
// category, or its category couldn't be loaded).
func EffectiveTargetMarkupBasisPoints(category *pmsDomain.Category, tenantDefaultBasisPoints int) int {
	if category != nil && category.TargetMarkupBasisPoints != nil {
		return *category.TargetMarkupBasisPoints
	}
	return tenantDefaultBasisPoints
}

// RecommendedPriceMinorUnits computes cost * (1 + markup), integer math
// with round-half-up, same smallest-currency-unit, no-float convention as
// every other money calculation in this codebase. Unlike margin, markup has
// no upper bound (validated only for >= 0 at the settings/category write
// boundary, see pms.ErrInvalidTargetMarkupBasisPoints), so unlike the old
// margin-based formula this never divides by a value that could reach
// zero or go negative. markupBasisPoints of 0 just returns
// costPriceMinorUnits unchanged (no markup).
func RecommendedPriceMinorUnits(costPriceMinorUnits int64, markupBasisPoints int) int64 {
	if markupBasisPoints <= 0 {
		return costPriceMinorUnits
	}
	numerator := costPriceMinorUnits * int64(10000+markupBasisPoints)
	return (numerator + 5000) / 10000
}
