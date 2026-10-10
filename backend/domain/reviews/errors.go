package reviews

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrReviewNotFound        = errs.New("REVIEW_NOT_FOUND", "review not found")
	ErrCustomerNameRequired  = errs.New("REVIEW_CUSTOMER_NAME_REQUIRED", "customer name is required")
	ErrCustomerEmailRequired = errs.New("REVIEW_CUSTOMER_EMAIL_REQUIRED", "customer email is required")
	ErrInvalidRating         = errs.New("REVIEW_INVALID_RATING", "rating must be between 1 and 5")
	ErrReviewBodyRequired    = errs.New("REVIEW_BODY_REQUIRED", "review body is required")
)
