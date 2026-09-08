// Package stripe implements domain/integrations.PaymentLinkClient against
// Stripe's official Go SDK (github.com/stripe/stripe-go), using a Checkout
// Session in "payment" mode as the "payment link" -- Stripe's own Payment
// Links API (POST /v1/payment_links) requires a pre-existing Price object,
// which this codebase has no equivalent of (a WhatsApp cart's price is
// whatever the SKU was selling for at ORDER-command time, never a stable
// catalog Price); a Checkout Session's inline price_data supports a one-off
// ad-hoc amount instead, which is what CreatePaymentLink actually needs.
package stripe

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/stripe/stripe-go/v82"

	"github.com/adirkuhn/commerceos/domain/integrations"
)

// Config holds one Stripe connection's secret key -- BaseURL/HTTPClient
// are overridable only so tests can point the SDK at an httptest.Server
// instead of Stripe's real API.
type Config struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// Client wraps a single tenant's *stripe.Client -- constructed fresh per
// connection (see ClientFactory.NewClient), never a shared/global
// stripe.Client, since each tenant has its own secret key and stripe-go's
// package-level stripe.Key global would race across tenants.
type Client struct {
	sc *stripe.Client
}

func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" && cfg.HTTPClient == nil {
		return &Client{sc: stripe.NewClient(cfg.APIKey)}
	}
	backendConfig := &stripe.BackendConfig{HTTPClient: cfg.HTTPClient}
	if cfg.BaseURL != "" {
		backendConfig.URL = stripe.String(cfg.BaseURL)
	}
	backend := stripe.GetBackendWithConfig(stripe.APIBackend, backendConfig)
	return &Client{sc: stripe.NewClient(cfg.APIKey, stripe.WithBackends(&stripe.Backends{API: backend}))}
}

func (c *Client) CreatePaymentLink(ctx context.Context, req integrations.PaymentLinkRequest) (integrations.PaymentLink, error) {
	params := &stripe.CheckoutSessionCreateParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:        stripe.String(req.SuccessURL),
		CancelURL:         stripe.String(req.CancelURL),
		ClientReferenceID: stripe.String(req.Reference),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
					Currency:   stripe.String(strings.ToLower(req.CurrencyCode)),
					UnitAmount: stripe.Int64(req.AmountMinorUnits),
					ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
						Name:        stripe.String(req.Description),
						Description: stripe.String(req.Description),
					},
				},
			},
		},
	}

	session, err := c.sc.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return integrations.PaymentLink{}, fmt.Errorf("stripe: create checkout session: %w", err)
	}
	return integrations.PaymentLink{URL: session.URL, ExternalID: session.ID}, nil
}

// Refund implements domain/integrations.PaymentLinkClient.Refund. Stripe's
// refund API needs a PaymentIntent (or Charge) id, not a Checkout Session
// id -- but externalPaymentID is the SESSION id (session.ID, see
// StripeEventParser and PaymentLink.ExternalID's doc), so this retrieves
// the session first to resolve its PaymentIntent. reason is stored as
// refund metadata rather than Stripe's own Reason param, whose values are
// a fixed enum (duplicate/fraudulent/requested_by_customer) that a picking
// staff's free-text note doesn't fit.
func (c *Client) Refund(ctx context.Context, externalPaymentID string, amountMinorUnits int64, reason string) (string, error) {
	session, err := c.sc.V1CheckoutSessions.Retrieve(ctx, externalPaymentID, nil)
	if err != nil {
		return "", fmt.Errorf("stripe: retrieve checkout session: %w", err)
	}
	if session.PaymentIntent == nil || session.PaymentIntent.ID == "" {
		return "", fmt.Errorf("stripe: checkout session %s has no payment intent to refund", externalPaymentID)
	}

	params := &stripe.RefundCreateParams{
		PaymentIntent: stripe.String(session.PaymentIntent.ID),
		Amount:        stripe.Int64(amountMinorUnits),
	}
	if reason != "" {
		params.Metadata = map[string]string{"reason": reason}
	}
	refund, err := c.sc.V1Refunds.Create(ctx, params)
	if err != nil {
		return "", fmt.Errorf("stripe: create refund: %w", err)
	}
	return refund.ID, nil
}

// CreatePaymentIntent implements domain/integrations.CardPaymentClient --
// AutomaticPaymentMethods lets Stripe decide which payment methods to
// surface in the Elements widget based on what's enabled in the connected
// account's Dashboard, rather than this codebase hardcoding "card".
func (c *Client) CreatePaymentIntent(ctx context.Context, amountMinorUnits int64, currencyCode string) (integrations.CardPaymentIntent, error) {
	params := &stripe.PaymentIntentCreateParams{
		Amount:                  stripe.Int64(amountMinorUnits),
		Currency:                stripe.String(strings.ToLower(currencyCode)),
		AutomaticPaymentMethods: &stripe.PaymentIntentCreateAutomaticPaymentMethodsParams{Enabled: stripe.Bool(true)},
	}
	intent, err := c.sc.V1PaymentIntents.Create(ctx, params)
	if err != nil {
		return integrations.CardPaymentIntent{}, fmt.Errorf("stripe: create payment intent: %w", err)
	}
	return integrations.CardPaymentIntent{ClientSecret: intent.ClientSecret, ID: intent.ID}, nil
}

// RetrievePaymentIntent implements domain/integrations.CardPaymentClient --
// the server-side source of truth application/storefront's checkout
// re-checks against before ever trusting a client-confirmed card payment.
func (c *Client) RetrievePaymentIntent(ctx context.Context, id string) (integrations.CardPaymentStatus, error) {
	intent, err := c.sc.V1PaymentIntents.Retrieve(ctx, id, nil)
	if err != nil {
		return integrations.CardPaymentStatus{}, fmt.Errorf("stripe: retrieve payment intent: %w", err)
	}
	return integrations.CardPaymentStatus{
		ID: intent.ID, Status: string(intent.Status),
		AmountMinorUnits: intent.Amount, CurrencyCode: strings.ToUpper(string(intent.Currency)),
	}, nil
}

// ClientFactory implements domain/integrations.PaymentLinkClientFactory.
type ClientFactory struct{}

func NewClientFactory() ClientFactory { return ClientFactory{} }

func (ClientFactory) NewClient(apiKey string) (integrations.PaymentLinkClient, error) {
	return NewClient(Config{APIKey: apiKey}), nil
}

// NewCardPaymentClient implements domain/integrations.CardPaymentClientFactory
// -- same underlying *Client as NewClient, exposed through the narrower
// CardPaymentClient interface for callers that only need the PaymentIntent
// capability.
func (ClientFactory) NewCardPaymentClient(apiKey string) (integrations.CardPaymentClient, error) {
	return NewClient(Config{APIKey: apiKey}), nil
}
