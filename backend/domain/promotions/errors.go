package promotions

import "errors"

var (
	ErrPromotionNotFound     = errors.New("promotion not found")
	ErrPromotionNameRequired = errors.New("promotion name is required")
	ErrInvalidPromotionType  = errors.New("promotion type must be percentage or fixed_amount")
	// ErrInvalidPromotionStatus guards Promotion.Status against any value
	// outside the three declared constants.
	ErrInvalidPromotionStatus = errors.New("promotion status must be draft, active, or ended")
	// ErrPromotionEndBeforeStart guards Promotion.EndsAt, when set: it must
	// be strictly after StartsAt.
	ErrPromotionEndBeforeStart = errors.New("promotion end date must be after the start date")
	// ErrPromotionRequiresAtLeastOneSKU guards the line set: a promotion
	// with no SKUs attached discounts nothing.
	ErrPromotionRequiresAtLeastOneSKU = errors.New("promotion must include at least one sku")
	// ErrDuplicatePromotionSKU guards the line set: the same SKU can't
	// appear twice in one promotion (which discount would even apply?).
	ErrDuplicatePromotionSKU = errors.New("sku appears more than once in this promotion")
	// ErrInvalidDiscountValue guards PromotionSKU.DiscountValue against its
	// DiscountType: percentage must be 1-10000 basis points (0.01%-100%),
	// fixed_amount must be positive.
	ErrInvalidDiscountValue = errors.New("discount value is invalid for the given discount type")
	// ErrClearanceBatchNotFound guards PromotionSKU.BatchID: the batch must
	// exist and belong to the line's own SKU.
	ErrClearanceBatchNotFound = errors.New("batch not found for this sku")
	// ErrClearanceBatchFinished rejects a clearance line whose batch is
	// already sold out, removed or expired -- it would end immediately.
	ErrClearanceBatchFinished = errors.New("batch has no stock left or has already expired")
)
