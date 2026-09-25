// Package myparcel is a small, self-built client for MyParcel's Shipments
// API (https://developer.myparcel.nl/api-reference/). No approved
// third-party Go SDK exists for this API -- built to spec against the docs
// fetched during design (05.authentication.html, 06.shipments.html,
// 07.shipment-related-object-definitions.html, 08.delivery-options.html,
// 10.webhooks.html, 11.webhook-object-definitions.html, 13.orders.html).
//
// CreateShipment's response envelope was originally guessed wrong
// (data.shipments[]{id,barcode}) and produced a spurious "empty response"
// error on every real create -- despite MyParcel actually creating the
// shipment -- because the real envelope is data.ids[]{id,reference_identifier}
// (confirmed against developer.myparcel.nl/api-reference/06.shipments.html
// and 07.shipment-related-object-definitions.html's ShipmentIds object).
// Barcode isn't in that response at all; CreateShipment now makes a
// follow-up GET /shipments/{id} for it (data.search_results.shipments[],
// confirmed against 06.shipments.html's search example) -- see
// getShipmentByID's doc comment for why a failure there is non-fatal.
//
// Orders/Fulfilment's response envelope (data.orders[]{uuid,status}) and
// request's reference field (there IS NO per-shipment reference_identifier
// on this endpoint, only the order-level external_identifier) are both
// confirmed against 13.orders.html. Its price/vat/price_after_vat and
// order_lines[] fields are also confirmed present and required by that
// page's own example request, but the page does NOT state their unit
// (minor units/cents vs whole currency) -- this client sends them in minor
// units, matching every other money field in this codebase and MyParcel's
// own confirmed Shipments-insurance convention, but that specific choice
// is unconfirmed against a real sandbox response.
//
// Things still NOT fully confirmed against a real sandbox response and
// needing verification before this goes live:
//   - The auth header: two independent doc-page fetches described
//     `Authorization: bearer base64(apiKey)`, but one page separately
//     mentioned a header named `X-MyParcel-Authorization` carrying the same
//     value. This client sends `Authorization: bearer ...` (authHeader
//     below is the one place to change if a real key gets a 401).
//   - The physical page-size query param on GetLabel (A4 sheet vs a single
//     A6 label) -- this client sends `?format=A6` on the shipment_labels
//     GET request (MyParcel's docs describe a "format" query param for this
//     exact endpoint, distinct from the Accept header's pdf/zpl content
//     type), but the param name/values weren't confirmed against a real
//     sandbox response during this build.
//   - The GET/POST /webhook_subscriptions response envelope
//     (webhookSubscriptionListResponse) -- modeled as
//     `data.webhook_subscriptions[].{id,hook,url}`, matching the request
//     body's own confirmed `data.webhook_subscriptions[].{hook,url}` shape
//     (developer.myparcel.nl/api-reference/10.webhooks.html's own example --
//     the request body WAS a real bug: this client originally sent
//     `data.hooks[]`, which MyParcel rejects with a 422 schema-validation
//     error), but the exact response field names still aren't confirmed
//     against a real sandbox response.
//   - The order_lines[]/price minor-units assumption above.
//   - GetDeliveryOptions' carrier query param -- see that method's own doc
//     comment.
//   - GetDeliveryOptions' platform query param -- a real 400 ("platform veld
//     is verplicht" / "platform field is required") confirmed this endpoint
//     needs it, but the accepted value is only inferred as "myparcel" from
//     MyParcel's own platform-slug convention elsewhere, not confirmed
//     against a real sandbox response.
package myparcel

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Config holds one MyParcel connection's credentials + endpoint. BaseURL
// defaults to MyParcel's real production endpoint when empty -- overridable
// only so tests can point at an httptest.Server.
type Config struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

const (
	defaultBaseURL = "https://api.myparcel.nl"

	// authHeader is the single place this client's auth header name lives --
	// see the package doc comment's first flagged item.
	authHeader = "Authorization"

	shipmentAccept = "application/vnd.shipment+json;charset=utf-8;version=1.1"

	// orderAccept is the Orders/Fulfilment API's content type -- plain
	// JSON, unlike Shipments' versioned vnd.shipment+json. Per the docs
	// fetched during design (13.orders.html); not yet confirmed against a
	// real sandbox response -- see the package doc comment's caveats.
	orderAccept = "application/json;charset=utf-8"
)

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

// APIError wraps a non-2xx MyParcel response.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("myparcel: request failed (%d): %s", e.StatusCode, e.Body)
}

func (c *Client) authValue() string {
	return "bearer " + base64.StdEncoding.EncodeToString([]byte(c.cfg.APIKey))
}

func (c *Client) doJSON(ctx context.Context, method, path, accept string, reqBody, respBody any) error {
	_, err := c.doJSONRaw(ctx, method, path, accept, reqBody, respBody)
	return err
}

// doJSONRaw is doJSON plus the raw response body -- only GetDeliveryOptions
// needs the raw bytes (to log them when MyParcel returns zero priced slots,
// see that method's doc comment), every other caller stays on doJSON.
func (c *Client) doJSONRaw(ctx context.Context, method, path, accept string, reqBody, respBody any) ([]byte, error) {
	var bodyReader io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("myparcel: encode request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("myparcel: build request: %w", err)
	}
	req.Header.Set(authHeader, c.authValue())
	req.Header.Set("Accept", accept)
	if reqBody != nil {
		req.Header.Set("Content-Type", accept)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("myparcel: request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	if respBody != nil && len(body) > 0 {
		if err := json.Unmarshal(body, respBody); err != nil {
			return nil, fmt.Errorf("myparcel: decode response for %s %s: %w", method, path, err)
		}
	}
	return body, nil
}
