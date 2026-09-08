package reviews

import "errors"

var (
	ErrReviewNotFound        = errors.New("review not found")
	ErrCustomerNameRequired  = errors.New("customer name is required")
	ErrCustomerEmailRequired = errors.New("customer email is required")
	ErrInvalidRating         = errors.New("rating must be between 1 and 5")
	ErrReviewBodyRequired    = errors.New("review body is required")
)
