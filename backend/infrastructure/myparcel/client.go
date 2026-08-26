// Package myparcel is a small, self-built client for MyParcel's Shipments
// API (https://developer.myparcel.nl/api-reference/). No approved
// third-party Go SDK exists for this API -- built to spec against the docs
// fetched during design (05.authentication.html, 06.shipments.html,
// 07.shipment-related-object-definitions.html, 10.webhooks.html,
// 11.webhook-object-definitions.html).
//
// Two things were NOT fully confirmed against a real sandbox response
// during this build and need verification before this goes live:
//   - The auth header: two independent doc-page fetches described
//     `Authorization: bearer base64(apiKey)`, but one page separately
//     mentioned a header named `X-MyParcel-Authorization` carrying the same
//     value. This client sends `Authorization: bearer ...` (authHeader
//     below is the one place to change if a real key gets a 401).
//   - The exact field MyParcel's create-shipment request uses to round-trip
//     our own SalesOrder.ID (referred to in the docs as
//     shipment_reference_identifier on the *webhook payload* side) -- this
//     client sends it as `reference_identifier` in the request body, the
//     most likely counterpart name, but this wasn't confirmed against the
//     request schema itself.
package myparcel

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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
	var bodyReader io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("myparcel: encode request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("myparcel: build request: %w", err)
	}
	req.Header.Set(authHeader, c.authValue())
	req.Header.Set("Accept", accept)
	if reqBody != nil {
		req.Header.Set("Content-Type", accept)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("myparcel: request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	if respBody != nil && len(body) > 0 {
		if err := json.Unmarshal(body, respBody); err != nil {
			return fmt.Errorf("myparcel: decode response for %s %s: %w", method, path, err)
		}
	}
	return nil
}

// --- Shipments ---

// Recipient/Options/ShipmentInput/ShipmentResult/LabelResult/TrackTraceResult
// are this package's own request/response shapes -- kept separate from
// domain/integrations' MyParcelClient port types so domain/application never
// import infrastructure types (CLAUDE.md's layering rule). adapter.go maps
// between the two at the boundary.

type Recipient struct {
	Person       string
	Company      string
	Street       string
	Number       string
	NumberSuffix string
	PostalCode   string
	City         string
	CC           string
	Email        string
	Phone        string
}

type Options struct {
	PackageType       int
	SignatureRequired bool
	OnlyRecipient     bool
	// InsuranceAmountMinorUnits 0 means not insured.
	InsuranceAmountMinorUnits int
}

type ShipmentInput struct {
	Recipient      Recipient
	Carrier        int
	Options        Options
	OrderReference string
}

type ShipmentResult struct {
	ShipmentID int64
	Barcode    string
}

type LabelResult struct {
	Data        []byte
	ContentType string
}

type TrackTraceResult struct {
	StatusCode int
	StatusMain string
}

type recipient struct {
	Person       string `json:"person"`
	Company      string `json:"company,omitempty"`
	Street       string `json:"street"`
	Number       string `json:"number,omitempty"`
	NumberSuffix string `json:"number_suffix,omitempty"`
	PostalCode   string `json:"postal_code,omitempty"`
	City         string `json:"city"`
	CC           string `json:"cc"`
	Email        string `json:"email,omitempty"`
	Phone        string `json:"phone,omitempty"`
}

type insuranceOption struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

type shipmentOptions struct {
	PackageType   int              `json:"package_type"`
	Signature     bool             `json:"signature,omitempty"`
	OnlyRecipient bool             `json:"only_recipient,omitempty"`
	Insurance     *insuranceOption `json:"insurance,omitempty"`
}

type createShipmentRequest struct {
	Carrier   int             `json:"carrier"`
	Recipient recipient       `json:"recipient"`
	Options   shipmentOptions `json:"options"`
	// ReferenceIdentifier round-trips our own SalesOrder.ID -- see the
	// package doc comment's second flagged item on the exact field name.
	ReferenceIdentifier string `json:"reference_identifier,omitempty"`
}

type shipmentsEnvelope struct {
	Data struct {
		Shipments []struct {
			ID      int64  `json:"id"`
			Barcode string `json:"barcode"`
		} `json:"shipments"`
	} `json:"data"`
}

// CreateShipment creates one MyParcel shipment. MyParcel's create-shipment
// endpoint accepts an array of shipments in one call; this client always
// sends exactly one and returns its result.
func (c *Client) CreateShipment(ctx context.Context, in ShipmentInput) (ShipmentResult, error) {
	req := struct {
		Data struct {
			Shipments []createShipmentRequest `json:"shipments"`
		} `json:"data"`
	}{}
	options := shipmentOptions{
		PackageType: in.Options.PackageType, Signature: in.Options.SignatureRequired, OnlyRecipient: in.Options.OnlyRecipient,
	}
	if in.Options.InsuranceAmountMinorUnits > 0 {
		options.Insurance = &insuranceOption{Amount: in.Options.InsuranceAmountMinorUnits, Currency: "EUR"}
	}
	req.Data.Shipments = []createShipmentRequest{{
		Carrier: in.Carrier,
		Recipient: recipient{
			Person: in.Recipient.Person, Company: in.Recipient.Company,
			Street: in.Recipient.Street, Number: in.Recipient.Number, NumberSuffix: in.Recipient.NumberSuffix,
			PostalCode: in.Recipient.PostalCode, City: in.Recipient.City, CC: in.Recipient.CC,
			Email: in.Recipient.Email, Phone: in.Recipient.Phone,
		},
		Options:             options,
		ReferenceIdentifier: in.OrderReference,
	}}

	var resp shipmentsEnvelope
	if err := c.doJSON(ctx, http.MethodPost, "/shipments", shipmentAccept, req, &resp); err != nil {
		return ShipmentResult{}, err
	}
	if len(resp.Data.Shipments) == 0 {
		return ShipmentResult{}, fmt.Errorf("myparcel: create shipment: empty response")
	}
	s := resp.Data.Shipments[0]
	return ShipmentResult{ShipmentID: s.ID, Barcode: s.Barcode}, nil
}

// GetLabel fetches a shipment's label. format is "pdf" or "zpl" -- see
// domain/shipping.LabelFormatPDF/LabelFormatZPL.
func (c *Client) GetLabel(ctx context.Context, myParcelShipmentID int64, format string) (LabelResult, error) {
	accept := "application/pdf"
	if format == "zpl" {
		accept = "application/vnd.shipment_label+zpl"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.cfg.BaseURL+"/shipment_labels/"+strconv.FormatInt(myParcelShipmentID, 10), nil)
	if err != nil {
		return LabelResult{}, fmt.Errorf("myparcel: build label request: %w", err)
	}
	req.Header.Set(authHeader, c.authValue())
	req.Header.Set("Accept", accept)

	resp, err := c.http.Do(req)
	if err != nil {
		return LabelResult{}, fmt.Errorf("myparcel: label request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return LabelResult{}, &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	return LabelResult{Data: body, ContentType: resp.Header.Get("Content-Type")}, nil
}

type trackTraceEnvelope struct {
	Data struct {
		TrackTraces []struct {
			Status struct {
				Current int    `json:"current"`
				Main    string `json:"main"`
			} `json:"status"`
		} `json:"tracktraces"`
	} `json:"data"`
}

// GetTrackTrace fetches a shipment's current tracking status. See the
// package doc comment and domain/shipping.Shipment.IsTerminal for the
// status caveats.
func (c *Client) GetTrackTrace(ctx context.Context, myParcelShipmentID int64) (TrackTraceResult, error) {
	var resp trackTraceEnvelope
	path := "/tracktraces/" + strconv.FormatInt(myParcelShipmentID, 10)
	if err := c.doJSON(ctx, http.MethodGet, path, "application/json", nil, &resp); err != nil {
		return TrackTraceResult{}, err
	}
	if len(resp.Data.TrackTraces) == 0 {
		return TrackTraceResult{}, fmt.Errorf("myparcel: get tracktrace: empty response")
	}
	tt := resp.Data.TrackTraces[0]
	return TrackTraceResult{StatusCode: tt.Status.Current, StatusMain: tt.Status.Main}, nil
}

type webhookSubscriptionRequest struct {
	Data struct {
		Hooks []struct {
			Hook string `json:"hook"`
			URL  string `json:"url"`
		} `json:"hooks"`
	} `json:"data"`
}

// RegisterWebhook registers this shop's shipment_status_change callback
// URL. MyParcel: "you can only have one webhook of each type at a time for
// a shop, so adding the same webhook overwrites any existing one" -- so
// this is safe to call more than once for the same connection/URL.
func (c *Client) RegisterWebhook(ctx context.Context, callbackURL string) error {
	var req webhookSubscriptionRequest
	req.Data.Hooks = []struct {
		Hook string `json:"hook"`
		URL  string `json:"url"`
	}{{Hook: "shipment_status_change", URL: callbackURL}}
	return c.doJSON(ctx, http.MethodPost, "/webhook_subscriptions", "application/json", req, nil)
}
