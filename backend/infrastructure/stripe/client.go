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

// ClientFactory implements domain/integrations.PaymentLinkClientFactory.
type ClientFactory struct{}

func NewClientFactory() ClientFactory { return ClientFactory{} }

func (ClientFactory) NewClient(apiKey string) (integrations.PaymentLinkClient, error) {
	return NewClient(Config{APIKey: apiKey}), nil
}
