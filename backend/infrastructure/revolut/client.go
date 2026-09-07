// Package revolut is a small, self-built client for Revolut's Merchant API
// (https://developer.revolut.com/docs/merchant/) -- no official Go SDK
// exists for it. Built from published docs, NOT confirmed against a real
// Revolut Business account -- same caveat convention as
// infrastructure/myparcel and infrastructure/evolutionapi's package doc
// comments. Confirm the exact request/response shape against a real
// account before relying on this in production; specifically:
//   - The base URL (this client uses https://merchant.revolut.com/api/1.0,
//     production -- Revolut also documents a sandbox host under a
//     different domain, not wired up here).
//   - The create-order response actually returning a usable "checkout_url"
//     field directly -- if it instead only returns a public_id, the
//     checkout URL has to be constructed from it
//     (https://checkout.revolut.com/payment-link/{public_id}), which this
//     client does as a fallback but has not verified is the real, current
//     format.
package revolut

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/adirkuhn/commerceos/domain/integrations"
)

const defaultBaseURL = "https://merchant.revolut.com/api/1.0"

type Config struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

type Client struct {
	cfg  Config
	http *http.Client
}

func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{cfg: cfg, http: cfg.HTTPClient}
}

type createOrderRequest struct {
	Amount              int64  `json:"amount"`
	Currency            string `json:"currency"`
	Description         string `json:"description,omitempty"`
	MerchantOrderExtRef string `json:"merchant_order_ext_ref,omitempty"`
	RedirectURL         string `json:"redirect_url,omitempty"`
}

type createOrderResponse struct {
	ID          string `json:"id"`
	PublicID    string `json:"public_id"`
	CheckoutURL string `json:"checkout_url"`
}

func (c *Client) CreatePaymentLink(ctx context.Context, req integrations.PaymentLinkRequest) (integrations.PaymentLink, error) {
	reqBody, err := json.Marshal(createOrderRequest{
		Amount: req.AmountMinorUnits, Currency: req.CurrencyCode,
		Description: req.Description, MerchantOrderExtRef: req.Reference, RedirectURL: req.SuccessURL,
	})
	if err != nil {
		return integrations.PaymentLink{}, fmt.Errorf("revolut: encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/orders", bytes.NewReader(reqBody))
	if err != nil {
		return integrations.PaymentLink{}, fmt.Errorf("revolut: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return integrations.PaymentLink{}, fmt.Errorf("revolut: create order: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return integrations.PaymentLink{}, fmt.Errorf("revolut: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return integrations.PaymentLink{}, fmt.Errorf("revolut: create order: status %d: %s", resp.StatusCode, string(respBody))
	}

	var order createOrderResponse
	if err := json.Unmarshal(respBody, &order); err != nil {
		return integrations.PaymentLink{}, fmt.Errorf("revolut: decode response: %w", err)
	}

	checkoutURL := order.CheckoutURL
	if checkoutURL == "" && order.PublicID != "" {
		// Fallback construction -- see this package's doc comment's second
		// flagged caveat.
		checkoutURL = "https://checkout.revolut.com/payment-link/" + order.PublicID
	}
	return integrations.PaymentLink{URL: checkoutURL, ExternalID: order.ID}, nil
}

type refundOrderRequest struct {
	Amount int64  `json:"amount"`
	Reason string `json:"reason,omitempty"`
}

type refundOrderResponse struct {
	ID string `json:"id"`
}

// Refund implements domain/integrations.PaymentLinkClient.Refund against
// Revolut's documented POST /orders/{id}/refund endpoint -- same
// not-confirmed-against-a-real-account caveat as CreatePaymentLink (see
// this package's doc comment); externalPaymentID is the order id
// CreatePaymentLink returned as PaymentLink.ExternalID. Unlike Stripe's
// refund call, this doesn't send a currency -- Revolut's docs describe the
// refund as always in the original order's own currency, needing no
// override, but that's unverified here same as everything else in this
// package.
func (c *Client) Refund(ctx context.Context, externalPaymentID string, amountMinorUnits int64, reason string) (string, error) {
	reqBody, err := json.Marshal(refundOrderRequest{Amount: amountMinorUnits, Reason: reason})
	if err != nil {
		return "", fmt.Errorf("revolut: encode refund request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/orders/"+externalPaymentID+"/refund", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("revolut: build refund request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("revolut: refund order: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("revolut: read refund response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("revolut: refund order: status %d: %s", resp.StatusCode, string(respBody))
	}

	var refund refundOrderResponse
	if err := json.Unmarshal(respBody, &refund); err != nil {
		return "", fmt.Errorf("revolut: decode refund response: %w", err)
	}
	return refund.ID, nil
}

// ClientFactory implements domain/integrations.PaymentLinkClientFactory.
type ClientFactory struct{}

func NewClientFactory() ClientFactory { return ClientFactory{} }

func (ClientFactory) NewClient(apiKey string) (integrations.PaymentLinkClient, error) {
	return NewClient(Config{APIKey: apiKey}), nil
}
