package promotions

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrPromotionNotFound     = errs.New("PROMOTION_NOT_FOUND", "promotion not found")
	ErrPromotionNameRequired = errs.New("PROMOTION_NAME_REQUIRED", "promotion name is required")
	ErrInvalidPromotionType  = errs.New("PROMOTION_INVALID_TYPE", "promotion type must be percentage or fixed_amount")
	// ErrInvalidPromotionStatus guards Promotion.Status against any value
	// outside the three declared constants.
	ErrInvalidPromotionStatus = errs.New("PROMOTION_INVALID_STATUS", "promotion status must be draft, active, or ended")
	// ErrPromotionEndBeforeStart guards Promotion.EndsAt, when set: it must
	// be strictly after StartsAt.
	ErrPromotionEndBeforeStart = errs.New("PROMOTION_END_BEFORE_START", "promotion end date must be after the start date")
	// ErrPromotionRequiresAtLeastOneSKU guards the line set: a promotion
	// with no SKUs attached discounts nothing.
	ErrPromotionRequiresAtLeastOneSKU = errs.New("PROMOTION_SKU_REQUIRED", "promotion must include at least one sku")
	// ErrDuplicatePromotionSKU guards the line set: the same SKU can't
	// appear twice in one promotion (which discount would even apply?).
	ErrDuplicatePromotionSKU = errs.New("PROMOTION_DUPLICATE_SKU", "sku appears more than once in this promotion")
	// ErrInvalidDiscountValue guards PromotionSKU.DiscountValue against its
	// DiscountType: percentage must be 1-10000 basis points (0.01%-100%),
	// fixed_amount must be positive.
	ErrInvalidDiscountValue = errs.New("PROMOTION_INVALID_DISCOUNT_VALUE", "discount value is invalid for the given discount type")
	// ErrClearanceBatchNotFound guards PromotionSKU.BatchID: the batch must
	// exist and belong to the line's own SKU.
	ErrClearanceBatchNotFound = errs.New("CLEARANCE_BATCH_NOT_FOUND", "batch not found for this sku")
	// ErrClearanceBatchFinished rejects a clearance line whose batch is
	// already sold out, removed or expired -- it would end immediately.
	ErrClearanceBatchFinished = errs.New("CLEARANCE_BATCH_FINISHED", "batch has no stock left or has already expired")
)
