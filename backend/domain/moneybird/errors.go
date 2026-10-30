package moneybird

import "github.com/adirkuhn/commerceos/domain/errs"

var (
	ErrNotConnected = errs.New("MONEYBIRD_NOT_CONNECTED", "Moneybird is not connected")
	// ErrNotConfigured: connected, but no administration or default
	// ledger account chosen yet.
	ErrNotConfigured    = errs.New("MONEYBIRD_NOT_CONFIGURED", "choose a Moneybird administration and ledger account first")
	ErrTokenInvalid     = errs.New("MONEYBIRD_TOKEN_INVALID", "Moneybird API token is missing or not accepted")
	ErrTaxRateNotMapped = errs.NewWithCause("MONEYBIRD_TAX_RATE_NOT_MAPPED", "a VAT rate has no Moneybird tax rate mapped")
	// ErrPaymentMethodNotMapped: the order is paid, but its payment method
	// isn't mapped -- booking the invoice without its payment would leave
	// the books showing it unpaid.
	ErrPaymentMethodNotMapped = errs.NewWithCause("MONEYBIRD_PAYMENT_METHOD_NOT_MAPPED", "the order's payment method has no Moneybird payment method mapped")
	// ErrDuplicateReference: Moneybird already has an invoice with this
	// reference that CommerceOS didn't create (e.g. another service booked
	// the order) -- never book it a second time.
	ErrDuplicateReference = errs.NewWithCause("MONEYBIRD_DUPLICATE_REFERENCE", "Moneybird already has an invoice with this reference")
	// ErrDocumentLocked: the estimate was sent/accepted in Moneybird and
	// can no longer be changed from here.
	ErrDocumentLocked = errs.NewWithCause("MONEYBIRD_DOCUMENT_LOCKED", "the Moneybird document can no longer be changed")
	// ErrDocumentNotFound: the quote/order was never sent to Moneybird.
	ErrDocumentNotFound = errs.New("MONEYBIRD_DOCUMENT_NOT_FOUND", "this was never sent to Moneybird")
	ErrNothingToBook    = errs.New("MONEYBIRD_NOTHING_TO_BOOK", "there are no lines to book")
	ErrRequestFailed    = errs.NewWithCause("MONEYBIRD_REQUEST_FAILED", "Moneybird request failed")
	ErrRateLimited      = errs.New("MONEYBIRD_RATE_LIMITED", "Moneybird is busy, try again in a few minutes")
)
