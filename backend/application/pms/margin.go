package pms

import pmsDomain "github.com/adirkuhn/commerceos/domain/pms"

// EffectiveTargetMarginBasisPoints resolves the margin used to compute a
// product's recommended price: its category's override when one is set,
// otherwise the tenant-wide default. category may be nil (product has no
// category, or its category couldn't be loaded).
func EffectiveTargetMarginBasisPoints(category *pmsDomain.Category, tenantDefaultBasisPoints int) int {
	if category != nil && category.TargetMarginBasisPoints != nil {
		return *category.TargetMarginBasisPoints
	}
	return tenantDefaultBasisPoints
}

// RecommendedPriceMinorUnits computes cost / (1 - margin), integer math
// with round-half-up on the final division -- same smallest-currency-unit,
// no-float convention as every other money calculation in this codebase.
// marginBasisPoints must be < 10000 (validated at the settings/category
// write boundary, see pms.ErrInvalidTargetMarginBasisPoints); a
// marginBasisPoints of 0 just returns costPriceMinorUnits unchanged (no
// markup).
func RecommendedPriceMinorUnits(costPriceMinorUnits int64, marginBasisPoints int) int64 {
	if marginBasisPoints <= 0 {
		return costPriceMinorUnits
	}
	denominator := int64(10000 - marginBasisPoints)
	numerator := costPriceMinorUnits * 10000
	return (numerator + denominator/2) / denominator
}
