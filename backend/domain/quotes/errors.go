package quotes

import "github.com/adirkuhn/commerceos/domain/errs"

var (
	ErrQuoteNotFound   = errs.New("QUOTE_NOT_FOUND", "quote not found")
	ErrNameRequired    = errs.New("QUOTE_NAME_REQUIRED", "a quote needs a name")
	ErrNameTooLong     = errs.New("QUOTE_NAME_TOO_LONG", "a quote name can be at most 200 characters")
	ErrInvalidDiscount = errs.New("QUOTE_INVALID_DISCOUNT", "the discount must be between 0% and 100%")
	ErrInvalidCurrency = errs.New("QUOTE_INVALID_CURRENCY", "the currency must be a three-letter code")
	ErrInvalidLine     = errs.New("QUOTE_INVALID_LINE", "every quote line needs a product, a quantity and a price of at least zero")
	ErrDuplicateLine   = errs.New("QUOTE_DUPLICATE_LINE", "a product can be on a quote only once")
	ErrTooManyLines    = errs.New("QUOTE_TOO_MANY_LINES", "a quote can have at most 500 lines")
	ErrLineUnavailable = errs.New("QUOTE_LINE_UNAVAILABLE", "a product on the quote is no longer active")
	ErrNothingToOrder  = errs.New("QUOTE_NOTHING_TO_ORDER", "the quote has no line with a quantity to order")
)
