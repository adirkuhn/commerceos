package payments

import (
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-go/v82"
)

// StripeEventParser implements PaymentEventParser for Stripe. Only
// checkout.session.completed with payment_status "paid" is ever Handled --
// a Checkout Session using an async payment method (bank debit, etc.) can
// complete before its payment actually settles, in which case Stripe sends
// checkout.session.async_payment_succeeded separately; that event isn't
// handled here yet (this codebase's Checkout Sessions never enable async
// payment methods today, so it's not expected to occur -- extend here, not
// with a branch elsewhere, if that changes).
type StripeEventParser struct{}

func NewStripeEventParser() *StripeEventParser { return &StripeEventParser{} }

func (StripeEventParser) Parse(rawPayload []byte) (PaymentEvent, error) {
	var event stripe.Event
	if err := json.Unmarshal(rawPayload, &event); err != nil {
		return PaymentEvent{}, fmt.Errorf("stripeEventParser: decode event: %w", err)
	}
	if event.Type != stripe.EventTypeCheckoutSessionCompleted {
		return PaymentEvent{Handled: false}, nil
	}

	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return PaymentEvent{}, fmt.Errorf("stripeEventParser: decode checkout session: %w", err)
	}
	if session.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
		return PaymentEvent{Handled: false}, nil
	}

	return PaymentEvent{Handled: true, OrderNumber: session.ClientReferenceID, ExternalPaymentID: session.ID}, nil
}
