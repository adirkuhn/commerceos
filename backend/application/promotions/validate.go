package promotions

import (
	"context"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

// skuExistenceChecker narrows pms.SKURepository to the one method this
// package needs (ISP) -- defined here at point of use rather than importing
// the whole domain/pms.SKURepository interface, same pattern as
// application/tax's skuTaxClassChecker.
type skuExistenceChecker interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.SKU, error)
}

// PromotionLineInput is one SKU's discount within a promotion -- shared
// shape for CreatePromotionUseCase and UpdatePromotionUseCase.
type PromotionLineInput struct {
	SKUID         string
	DiscountType  promotionsDomain.PromotionType
	DiscountValue int64
}

// validatePromotionType is the shared invariant for Create/UpdatePromotion:
// only the two v1-supported mechanisms are accepted.
func validatePromotionType(t promotionsDomain.PromotionType) error {
	switch t {
	case promotionsDomain.PromotionTypePercentage, promotionsDomain.PromotionTypeFixedAmount:
		return nil
	default:
		return promotionsDomain.ErrInvalidPromotionType
	}
}

// validatePromotionStatus is the shared invariant for Create/UpdatePromotion.
func validatePromotionStatus(s promotionsDomain.PromotionStatus) error {
	switch s {
	case promotionsDomain.PromotionStatusDraft, promotionsDomain.PromotionStatusActive, promotionsDomain.PromotionStatusEnded:
		return nil
	default:
		return promotionsDomain.ErrInvalidPromotionStatus
	}
}

// validateDiscountValue is the shared invariant for a PromotionSKU line:
// percentage is 1-10000 basis points (0.01%-100%), fixed_amount must be
// positive. Assumes discountType has already passed validatePromotionType.
func validateDiscountValue(discountType promotionsDomain.PromotionType, value int64) error {
	switch discountType {
	case promotionsDomain.PromotionTypePercentage:
		if value < 1 || value > 10000 {
			return promotionsDomain.ErrInvalidDiscountValue
		}
	case promotionsDomain.PromotionTypeFixedAmount:
		if value < 1 {
			return promotionsDomain.ErrInvalidDiscountValue
		}
	}
	return nil
}

// validateLines is the shared invariant for Create/UpdatePromotion: at
// least one line, no duplicate SKU, and each line's own type/value/SKU
// existence checked individually.
func validateLines(ctx context.Context, skuChecker skuExistenceChecker, lines []PromotionLineInput) error {
	if len(lines) == 0 {
		return promotionsDomain.ErrPromotionRequiresAtLeastOneSKU
	}
	seen := make(map[string]bool, len(lines))
	for _, line := range lines {
		if seen[line.SKUID] {
			return promotionsDomain.ErrDuplicatePromotionSKU
		}
		seen[line.SKUID] = true
		if err := validatePromotionType(line.DiscountType); err != nil {
			return err
		}
		if err := validateDiscountValue(line.DiscountType, line.DiscountValue); err != nil {
			return err
		}
		if _, err := skuChecker.FindByID(ctx, line.SKUID); err != nil {
			return err
		}
	}
	return nil
}
