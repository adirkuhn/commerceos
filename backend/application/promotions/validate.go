package promotions

import (
	"context"
	"fmt"
	"time"

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
	SKUID string
	// BatchID, when set, makes this a clearance line -- see
	// domain/promotions.PromotionSKU.BatchID.
	BatchID       *string
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
// existence checked individually. Also collects a warning (not a rejection)
// for any line whose SKU already carries a manual sale price -- a
// promotion always wins over it once applied (see
// domain/promotions.PromotionSKU's doc), so this is the one place that
// conflict is ever surfaced to the person creating the promotion.
// UpdatePromotionUseCase calls this too but currently discards the
// warnings -- only creation surfaces them.
func validateLines(ctx context.Context, skuChecker skuExistenceChecker, lines []PromotionLineInput) ([]string, error) {
	if len(lines) == 0 {
		return nil, promotionsDomain.ErrPromotionRequiresAtLeastOneSKU
	}
	var warnings []string
	seen := make(map[string]bool, len(lines))
	for _, line := range lines {
		if seen[line.SKUID] {
			return nil, promotionsDomain.ErrDuplicatePromotionSKU
		}
		seen[line.SKUID] = true
		if err := validatePromotionType(line.DiscountType); err != nil {
			return nil, err
		}
		if err := validateDiscountValue(line.DiscountType, line.DiscountValue); err != nil {
			return nil, err
		}
		sku, err := skuChecker.FindByID(ctx, line.SKUID)
		if err != nil {
			return nil, err
		}
		if sku.SalePriceMinorUnits != nil {
			warnings = append(warnings, fmt.Sprintf("SKU %s already has a manual sale price -- this promotion will override it while active", sku.SKUCode))
		}
	}
	return warnings, nil
}

// validateClearanceBatches checks every clearance line's batch belongs to
// its SKU and isn't already finished -- a clearance that would end the
// moment it starts is almost certainly a mistake. Lines whose batch is in
// alreadyOnPromotion are skipped: UpdatePromotionUseCase passes the batches
// the promotion already had, so editing a running (or already sold-out)
// clearance isn't blocked by its own batch. Must run outside
// tenant.WithTenant -- see skuBatchLister.
func validateClearanceBatches(ctx context.Context, batches skuBatchLister, lines []PromotionLineInput, alreadyOnPromotion map[string]bool) error {
	now := time.Now()
	for _, line := range lines {
		if line.BatchID == nil || alreadyOnPromotion[*line.BatchID] {
			continue
		}
		skuBatches, err := batches.Execute(ctx, line.SKUID)
		if err != nil {
			return err
		}
		batch := clearanceBatchFor(skuBatches, *line.BatchID)
		if !batch.Found {
			return promotionsDomain.ErrClearanceBatchNotFound
		}
		if batch.IsFinished(now) {
			return promotionsDomain.ErrClearanceBatchFinished
		}
	}
	return nil
}

// hasClearanceLine reports whether any line carries a BatchID.
func hasClearanceLine(lines []PromotionLineInput) bool {
	for _, l := range lines {
		if l.BatchID != nil {
			return true
		}
	}
	return false
}
