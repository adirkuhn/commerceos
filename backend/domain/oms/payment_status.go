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
