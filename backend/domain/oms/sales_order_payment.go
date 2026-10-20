package oms

import "github.com/adirkuhn/commerceos/domain/errs"

// SplitPaymentMethod is a SalesOrder's PaymentMethod when it was paid in
// more than one part -- the parts themselves are its Payments.
const SplitPaymentMethod = "split"

// ErrPaymentsTotalMismatch guards ReceivePOSSalesOrderUseCase: the parts of
// a (split) payment must add up to the order's total, or the order would
// record money that was never taken, or miss money that was.
var ErrPaymentsTotalMismatch = errs.New("PAYMENTS_TOTAL_MISMATCH", "payments must add up to the order total")

// SalesOrderPayment is one part of what paid a SalesOrder -- a POS sale
// can be paid in several parts with any mix of methods (card+card,
// cash+card, ...). Method is the payment method's name as the source calls
// it (e.g. "Cash", "Revolut"); TransactionID is the payment processor's
// reference, empty when there is none (cash).
type SalesOrderPayment struct {
	Method           string
	AmountMinorUnits int64
	TransactionID    string
}

// SalesOrderPayments are a SalesOrder's payment parts, in the order they
// were taken. Empty for an order whose source reports no breakdown (every
// source except a POS that sends one) -- PaymentMethod/TransactionID on the
// order are then the only record of how it was paid.
type SalesOrderPayments []SalesOrderPayment

// RequireTotal fails with ErrPaymentsTotalMismatch unless the parts add up
// to total. No parts at all is fine -- there's no breakdown to check.
func (p SalesOrderPayments) RequireTotal(total int64) error {
	if len(p) == 0 {
		return nil
	}
	var sum int64
	for _, payment := range p {
		sum += payment.AmountMinorUnits
	}
	if sum != total {
		return ErrPaymentsTotalMismatch
	}
	return nil
}

// Method is the order-level payment method these parts add up to: the
// single part's method, SplitPaymentMethod for several, empty for none.
func (p SalesOrderPayments) Method() string {
	switch len(p) {
	case 0:
		return ""
	case 1:
		return p[0].Method
	default:
		return SplitPaymentMethod
	}
}

// TransactionID is the order-level transaction reference: the single
// part's, empty otherwise -- several parts have no one reference.
func (p SalesOrderPayments) TransactionID() string {
	if len(p) != 1 {
		return ""
	}
	return p[0].TransactionID
}
