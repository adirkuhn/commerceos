package integrations

import "context"

// PaymentLinkRequest is what a caller asks a payment integration to
// generate a checkout URL for -- provider-agnostic, deliberately no
// provider-specific fields (Stripe's Checkout Session, Revolut's Merchant
// order, and any future provider all map onto this same shape).
type PaymentLinkRequest struct {
	AmountMinorUnits int64
	// CurrencyCode is an uppercase ISO 4217 code (e.g. "EUR"), same
	// convention as pms.SKU.CurrencyCode -- each adapter lowercases it
	// itself if its API requires that (Stripe does).
	CurrencyCode string
	// Description is shown to the payer on the provider's own checkout
	// page -- e.g. "Order #WA-1A2B3C4D".
	Description string
	// Reference is the caller's own identifier (a SalesOrder's Number),
	// passed through as the provider's reconciliation field (Stripe's
	// client_reference_id) where one exists.
	Reference string
	// SuccessURL/CancelURL are where the provider redirects the payer
	// after a payment attempt -- required by Stripe's Checkout Session,
	// optional for Revolut's order (its own hosted page needs neither to
	// function, it just has nothing to send the payer back to without
	// one).
	SuccessURL string
	CancelURL  string
}

// PaymentLink is the result of successfully generating one -- URL is what
// gets sent to the payer, ExternalID is the provider's own session/order
// id (kept for future reconciliation, e.g. matching an incoming webhook to
// the SalesOrder it paid for -- not wired up yet).
type PaymentLink struct {
	URL        string
	ExternalID string
}

// PaymentLinkClient is the payment integration contract -- Stripe
// (infrastructure/stripe) and Revolut (infrastructure/revolut) both
// implement it, and any future payment provider slots in as a third
// implementation with no change to a caller.
type PaymentLinkClient interface {
	CreatePaymentLink(ctx context.Context, req PaymentLinkRequest) (PaymentLink, error)
	// Refund issues a refund against externalPaymentID -- the same id
	// PaymentLink.ExternalID captured at checkout time (persisted as
	// SalesOrder.PaymentExternalID once a payment webhook confirms it, see
	// application/payments.WebhookDispatchHandler). amountMinorUnits is the
	// amount to refund, in the same minor-unit currency the original charge
	// was made in. Returns the provider's own refund reference on success.
	// Called by application/oms.CompletePickingUseCase's PickScanActionRefund
	// replay -- see that type's doc comment for what happens when this
	// errors (never blocks picking completion).
	Refund(ctx context.Context, externalPaymentID string, amountMinorUnits int64, reason string) (refundReference string, err error)
}

// PaymentLinkClientFactory builds a PaymentLinkClient scoped to one
// connection's secret key -- same shape as MyParcelClientFactory/
// EvolutionAPIClientFactory, for the same reason (DI, see CLAUDE.md's
// SOLID D rule): the use case that needs to create a payment link never
// constructs a concrete client itself.
type PaymentLinkClientFactory interface {
	NewClient(apiKey string) (PaymentLinkClient, error)
}

// CardPaymentIntent is the result of starting a PaymentIntent -- ClientSecret
// is handed to the storefront client to confirm the card charge via Stripe
// Elements/Stripe.js, ID is kept for server-side verification afterward
// (see CardPaymentClient.RetrievePaymentIntent).
type CardPaymentIntent struct {
	ClientSecret string
	ID           string
}

// CardPaymentStatus is what a server-side re-check of a PaymentIntent
// reports -- application/storefront's checkout never trusts a
// client-asserted "already paid"; it re-fetches this from the provider and
// only proceeds when Status is the provider's own "succeeded" value and
// the amount/currency match the order being placed.
type CardPaymentStatus struct {
	ID               string
	Status           string
	AmountMinorUnits int64
	CurrencyCode     string
}

// CardPaymentClient is a Stripe-specific capability (embedded card fields
// via Stripe Elements/Stripe.js) -- deliberately its own narrow interface
// rather than added to PaymentLinkClient, which every payment provider
// (including Revolut, which has no equivalent client-side-confirm SDK)
// must implement. Only infrastructure/stripe implements this.
type CardPaymentClient interface {
	CreatePaymentIntent(ctx context.Context, amountMinorUnits int64, currencyCode string) (CardPaymentIntent, error)
	RetrievePaymentIntent(ctx context.Context, id string) (CardPaymentStatus, error)
}

// CardPaymentClientFactory mirrors PaymentLinkClientFactory's shape/reason
// (DI, see CLAUDE.md's SOLID D rule) for CardPaymentClient instead. Named
// NewCardPaymentClient (not NewClient) so a single concrete factory type
// (infrastructure/stripe.ClientFactory) can implement both this and
// PaymentLinkClientFactory at once -- one Go type can't have two methods
// named NewClient differing only by return type.
type CardPaymentClientFactory interface {
	NewCardPaymentClient(apiKey string) (CardPaymentClient, error)
}
