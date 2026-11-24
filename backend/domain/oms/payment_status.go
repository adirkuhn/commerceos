package oms

// PaymentStatus answers "did this order's payment succeed?" -- derived
// from Status and DatePaid (never stored), so it can't drift from them.
type PaymentStatus string

const (
	PaymentStatusAwaitingPayment PaymentStatus = "awaiting_payment"
	// PaymentStatusAwaitingConfirmation is WooCommerce's on-hold: an
	// offline/delayed method (bank transfer, SEPA) the store must confirm.
	PaymentStatusAwaitingConfirmation PaymentStatus = "awaiting_confirmation"
	PaymentStatusPaid                 PaymentStatus = "paid"
	PaymentStatusFailed               PaymentStatus = "failed"
	PaymentStatusRefunded             PaymentStatus = "refunded"
	PaymentStatusPartiallyRefunded    PaymentStatus = "partially_refunded"
	PaymentStatusCancelledUnpaid      PaymentStatus = "cancelled_unpaid"
)

// PaymentStatuses lists every PaymentStatus.
func PaymentStatuses() []PaymentStatus {
	return []PaymentStatus{
		PaymentStatusAwaitingPayment, PaymentStatusAwaitingConfirmation, PaymentStatusPaid, PaymentStatusFailed,
		PaymentStatusRefunded, PaymentStatusPartiallyRefunded, PaymentStatusCancelledUnpaid,
	}
}

// PaymentStatus follows WooCommerce's status meanings (see
// SalesOrderStatus): processing/shipped/completed are paid, pending awaits
// payment, on-hold awaits the store's confirmation. A DatePaid wins over
// pending/on-hold/cancelled -- the payment did land -- but never over
// failed or a refund.
func (o SalesOrder) PaymentStatus() PaymentStatus {
	switch o.Status {
	case SalesOrderStatusFailed:
		return PaymentStatusFailed
	case SalesOrderStatusRefunded:
		return PaymentStatusRefunded
	case SalesOrderStatusPartiallyRefunded:
		return PaymentStatusPartiallyRefunded
	case SalesOrderStatusProcessing, SalesOrderStatusShipped, SalesOrderStatusCompleted:
		return PaymentStatusPaid
	}
	if o.DatePaid != nil {
		return PaymentStatusPaid
	}
	switch o.Status {
	case SalesOrderStatusOnHold:
		return PaymentStatusAwaitingConfirmation
	case SalesOrderStatusCancelled:
		return PaymentStatusCancelledUnpaid
	default:
		return PaymentStatusAwaitingPayment
	}
}

// MakesPayingCustomer reports whether this order makes its customer a
// paying customer: it was paid (a partial refund still counts) and was
// neither cancelled nor fully refunded. The customer repository computes
// the same rule in SQL (see PayingCustomerSQL); keep the two together.
func (o SalesOrder) MakesPayingCustomer() bool {
	if o.Status == SalesOrderStatusCancelled {
		return false
	}
	switch o.PaymentStatus() {
	case PaymentStatusPaid, PaymentStatusPartiallyRefunded:
		return true
	default:
		return false
	}
}

// PaymentReference is the id to look the payment up by: the channel's own
// TransactionID (e.g. Woo's gateway charge id) when known, else the
// provider id CommerceOS itself recorded (PaymentExternalID, e.g. a
// storefront card payment's Stripe PaymentIntent).
func (o SalesOrder) PaymentReference() string {
	if o.TransactionID != "" {
		return o.TransactionID
	}
	return o.PaymentExternalID
}

// ExpectsPaymentRecord reports whether an order in this status went
// through a payment -- so a channel order in it with no transaction id and
// no paid date is missing its payment info rather than simply unpaid.
// On-hold counts: an offline payment is still recorded once confirmed.
func (s SalesOrderStatus) ExpectsPaymentRecord() bool {
	switch s {
	case SalesOrderStatusProcessing, SalesOrderStatusOnHold, SalesOrderStatusShipped,
		SalesOrderStatusCompleted, SalesOrderStatusRefunded, SalesOrderStatusPartiallyRefunded:
		return true
	default:
		return false
	}
}

// PaymentRecordExpectedStatuses lists every status ExpectsPaymentRecord
// accepts -- for a repository filtering in SQL.
func PaymentRecordExpectedStatuses() []SalesOrderStatus {
	var out []SalesOrderStatus
	for _, s := range SalesOrderStatuses() {
		if s.ExpectsPaymentRecord() {
			out = append(out, s)
		}
	}
	return out
}
