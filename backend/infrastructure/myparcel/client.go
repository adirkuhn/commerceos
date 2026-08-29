// Package myparcel is a small, self-built client for MyParcel's Shipments
// API (https://developer.myparcel.nl/api-reference/). No approved
// third-party Go SDK exists for this API -- built to spec against the docs
// fetched during design (05.authentication.html, 06.shipments.html,
// 07.shipment-related-object-definitions.html, 10.webhooks.html,
// 11.webhook-object-definitions.html, 13.orders.html).
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
package myparcel

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/adirkuhn/commerceos/domain/integrations"
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
	// InsuranceCurrency is required by MyParcel whenever insured (see
	// insuranceOption). Empty falls back to "EUR" -- CreateShipment's own
	// default when a caller doesn't have a tenant currency to hand (e.g.
	// never configured Settings -> Localization -> Default Currency).
	InsuranceCurrency string
}

// defaultInsuranceCurrency is CreateShipment's fallback when the caller
// passes no InsuranceCurrency -- matches this client's original hardcoded
// behavior for a tenant with no Settings -> Localization -> Default
// Currency configured yet (domain/settings.TenantSettings' zero value).
const defaultInsuranceCurrency = "EUR"

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

// OrderResult is this package's own response shape for CreateOrder.
type OrderResult struct {
	UUID   string
	Status string
}

type LabelResult struct {
	Data        []byte
	ContentType string
}

type TrackTraceResult struct {
	StatusCode int
	StatusMain string
}

// WebhookSubscription is this package's own response shape for a
// registered webhook subscription -- adapter.go maps it to
// domain/integrations.WebhookSubscription at the boundary.
type WebhookSubscription struct {
	ID   string
	Hook string
	URL  string
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

// shipmentIdsEnvelope is POST /shipments' real response envelope --
// data.ids[]{id,reference_identifier} -- see the package doc comment for
// why this isn't data.shipments[]{id,barcode} (the original, wrong guess).
type shipmentIdsEnvelope struct {
	Data struct {
		IDs []struct {
			ID                  int64  `json:"id"`
			ReferenceIdentifier string `json:"reference_identifier"`
		} `json:"ids"`
	} `json:"data"`
}

// shipmentSearchEnvelope is GET /shipments/{id}'s response envelope --
// data.search_results.shipments[], confirmed against
// 06.shipments.html's search example. Used only by getShipmentByID's
// post-create barcode fetch; other fields in the real response (recipient,
// options, price, etc) are intentionally not modeled here since nothing
// needs them yet.
type shipmentSearchEnvelope struct {
	Data struct {
		SearchResults struct {
			Shipments []struct {
				ID      int64  `json:"id"`
				Barcode string `json:"barcode"`
				Status  int    `json:"status"`
			} `json:"shipments"`
		} `json:"search_results"`
	} `json:"data"`
}

// getShipmentByID fetches a just-created shipment's barcode (and current
// status) -- CreateShipment's own response never includes them (see the
// package doc comment). Errors here are handled by the caller as non-fatal:
// the shipment id is already confirmed to exist on MyParcel's side by this
// point, so treating a barcode-fetch failure as a full CreateShipment
// failure would make the caller re-POST on retry and create a genuine
// second MyParcel shipment -- exactly the duplicate-creation bug this fix
// is for. A missing barcode gets backfilled later by
// ReconcileShipmentsUseCase's poll (ShipmentRepository.FindStaleNonTerminal).
func (c *Client) getShipmentByID(ctx context.Context, id int64) (barcode string, status int, err error) {
	var resp shipmentSearchEnvelope
	path := "/shipments/" + strconv.FormatInt(id, 10)
	if err := c.doJSON(ctx, http.MethodGet, path, shipmentAccept, nil, &resp); err != nil {
		return "", 0, err
	}
	if len(resp.Data.SearchResults.Shipments) == 0 {
		return "", 0, fmt.Errorf("myparcel: get shipment %d: empty response", id)
	}
	s := resp.Data.SearchResults.Shipments[0]
	return s.Barcode, s.Status, nil
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
		currency := in.Options.InsuranceCurrency
		if currency == "" {
			currency = defaultInsuranceCurrency
		}
		options.Insurance = &insuranceOption{Amount: in.Options.InsuranceAmountMinorUnits, Currency: currency}
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

	var resp shipmentIdsEnvelope
	if err := c.doJSON(ctx, http.MethodPost, "/shipments", shipmentAccept, req, &resp); err != nil {
		return ShipmentResult{}, err
	}
	if len(resp.Data.IDs) == 0 {
		return ShipmentResult{}, fmt.Errorf("myparcel: create shipment: empty response")
	}
	id := resp.Data.IDs[0].ID

	barcode, _, err := c.getShipmentByID(ctx, id)
	if err != nil {
		// The shipment id is confirmed real -- see getShipmentByID's doc
		// comment for why this doesn't fail the whole create.
		return ShipmentResult{ShipmentID: id}, nil
	}
	return ShipmentResult{ShipmentID: id, Barcode: barcode}, nil
}

// --- Orders (Fulfilment API) ---

// OrderInput is this package's own request shape for CreateOrder -- kept
// separate from domain/integrations.CreateOrderInput the same way
// ShipmentInput is kept separate from CreateShipmentInput (see the comment
// above ShipmentInput); adapter.go maps between the two.
// OrderLine is one line item on the order -- see the package doc comment
// for the confirmed field names/nesting and the unconfirmed minor-units
// assumption on Price/VAT/PriceAfterVAT.
type OrderLine struct {
	Quantity                int
	SKU                     string
	Name                    string
	PriceMinorUnits         int
	VATMinorUnits           int
	PriceAfterVATMinorUnits int
}

type OrderInput struct {
	ExternalIdentifier string
	OrderDate          time.Time
	AccountID          int
	ShopID             int
	Carrier            int
	Recipient          Recipient
	PackageType        int
	// PriceMinorUnits/VATMinorUnits/PriceAfterVATMinorUnits are the order's
	// own totals -- required by MyParcel's Orders API (see the package doc
	// comment); PriceAfterVATMinorUnits should equal
	// PriceMinorUnits+VATMinorUnits.
	PriceMinorUnits         int
	VATMinorUnits           int
	PriceAfterVATMinorUnits int
	OrderLines              []OrderLine
}

type orderShipment struct {
	Carrier   int             `json:"carrier"`
	Recipient recipient       `json:"recipient"`
	Options   shipmentOptions `json:"options"`
}

type orderLineProduct struct {
	SKU  string `json:"sku"`
	Name string `json:"name"`
}

type orderLine struct {
	Quantity      int              `json:"quantity"`
	Price         int              `json:"price"`
	VAT           int              `json:"vat"`
	PriceAfterVAT int              `json:"price_after_vat"`
	Product       orderLineProduct `json:"product"`
}

type createOrderRequest struct {
	ExternalIdentifier string        `json:"external_identifier"`
	OrderDate          string        `json:"order_date"`
	AccountID          int           `json:"account_id"`
	ShopID             int           `json:"shop_id"`
	Price              int           `json:"price"`
	VAT                int           `json:"vat"`
	PriceAfterVAT      int           `json:"price_after_vat"`
	Shipment           orderShipment `json:"shipment"`
	OrderLines         []orderLine   `json:"order_lines"`
}

type ordersEnvelope struct {
	Data struct {
		Orders []struct {
			UUID   string `json:"uuid"`
			Status string `json:"status"`
		} `json:"orders"`
	} `json:"data"`
}

// CreateOrder exports one order via MyParcel's Orders/Fulfilment API
// (POST /fulfilment/orders) -- see the package doc comment's caveats;
// this endpoint specifically has not been exercised against a real
// sandbox response during this build (needs verification with real
// credentials before relying on it in production, same open item as the
// rest of this integration -- see project memory on Bol/MyParcel
// verification).
func (c *Client) CreateOrder(ctx context.Context, in OrderInput) (OrderResult, error) {
	req := struct {
		Data struct {
			Orders []createOrderRequest `json:"orders"`
		} `json:"data"`
	}{}
	lines := make([]orderLine, len(in.OrderLines))
	for i, l := range in.OrderLines {
		lines[i] = orderLine{
			Quantity: l.Quantity, Price: l.PriceMinorUnits, VAT: l.VATMinorUnits, PriceAfterVAT: l.PriceAfterVATMinorUnits,
			Product: orderLineProduct{SKU: l.SKU, Name: l.Name},
		}
	}
	req.Data.Orders = []createOrderRequest{{
		ExternalIdentifier: in.ExternalIdentifier,
		OrderDate:          in.OrderDate.Format("2006-01-02 15:04:05"),
		AccountID:          in.AccountID,
		ShopID:             in.ShopID,
		Price:              in.PriceMinorUnits, VAT: in.VATMinorUnits, PriceAfterVAT: in.PriceAfterVATMinorUnits,
		Shipment: orderShipment{
			Carrier: in.Carrier,
			Recipient: recipient{
				Person: in.Recipient.Person, Company: in.Recipient.Company,
				Street: in.Recipient.Street, Number: in.Recipient.Number, NumberSuffix: in.Recipient.NumberSuffix,
				PostalCode: in.Recipient.PostalCode, City: in.Recipient.City, CC: in.Recipient.CC,
				Email: in.Recipient.Email, Phone: in.Recipient.Phone,
			},
			Options: shipmentOptions{PackageType: in.PackageType},
		},
		OrderLines: lines,
	}}

	var resp ordersEnvelope
	if err := c.doJSON(ctx, http.MethodPost, "/fulfilment/orders", orderAccept, req, &resp); err != nil {
		return OrderResult{}, err
	}
	if len(resp.Data.Orders) == 0 {
		return OrderResult{}, fmt.Errorf("myparcel: create order: empty response")
	}
	o := resp.Data.Orders[0]
	return OrderResult{UUID: o.UUID, Status: o.Status}, nil
}

// GetLabel fetches a shipment's label. format is "pdf" or "zpl" -- see
// domain/shipping.LabelFormatPDF/LabelFormatZPL -- the response Content-Type
// (Accept header), fixed at shipment-creation time. size is the physical
// page format for this fetch (domain/shipping.LabelSizeA4/LabelSizeA6),
// sent as the `format` query param -- see the package doc comment's third
// flagged item; unconfirmed against a real sandbox response. Empty size
// omits the query param, leaving MyParcel's own default.
func (c *Client) GetLabel(ctx context.Context, myParcelShipmentID int64, format, size string) (LabelResult, error) {
	accept := "application/pdf"
	if format == "zpl" {
		accept = "application/vnd.shipment_label+zpl"
	}

	url := c.cfg.BaseURL + "/shipment_labels/" + strconv.FormatInt(myParcelShipmentID, 10)
	if size != "" {
		url += "?format=" + size
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

// webhookSubscriptionRequest is POST /webhook_subscriptions' request body
// -- confirmed against developer.myparcel.nl/api-reference/10.webhooks.html's
// own example: `{"data":{"webhook_subscriptions":[{"hook":...,"url":...}]}}`.
// Was previously `data.hooks[]`, which MyParcel rejects with a 422 ("the
// property hooks is not defined and the definition does not allow
// additional properties").
type webhookSubscriptionRequest struct {
	Data struct {
		WebhookSubscriptions []struct {
			Hook string `json:"hook"`
			URL  string `json:"url"`
		} `json:"webhook_subscriptions"`
	} `json:"data"`
}

// webhookSubscriptionListResponse is GET/POST /webhook_subscriptions'
// response envelope -- the exact nesting (data.webhook_subscriptions[])
// mirrors the create request's own data.hooks[] nesting convention and the
// id/hook/url field names documented for the resource, but -- like the
// three items already flagged in this package's doc comment -- was NOT
// confirmed against a real sandbox response during this build.
type webhookSubscriptionListResponse struct {
	Data struct {
		WebhookSubscriptions []struct {
			ID   int    `json:"id"`
			Hook string `json:"hook"`
			URL  string `json:"url"`
		} `json:"webhook_subscriptions"`
	} `json:"data"`
}

// RegisterWebhook registers this shop's callback URL for every hook type
// this codebase knows about (integrations.AllMyParcelHookTypes), not just
// shipment_status_change. MyParcel: "you can only have one webhook of each
// type at a time for a shop, so adding the same webhook overwrites any
// existing one" -- so this is safe to call more than once for the same
// connection/URL, and also overwrites any stale subscription (e.g. one left
// behind by a previous integration such as a WooCommerce plugin) pointing
// at a different delivery URL for the same hook.
//
// Every hook type is attempted even if an earlier one fails, so one bad
// hook can't silently leave the rest unregistered; all failures are joined
// into a single returned error naming which hook(s) failed.
func (c *Client) RegisterWebhook(ctx context.Context, callbackURL string) error {
	var errs []error
	for _, hook := range integrations.AllMyParcelHookTypes {
		if _, err := c.CreateWebhookSubscription(ctx, string(hook), callbackURL); err != nil {
			errs = append(errs, fmt.Errorf("hook %s: %w", hook, err))
		}
	}
	return errors.Join(errs...)
}

// CreateWebhookSubscription is RegisterWebhook generalized to any hook type
// -- see domain/integrations.MyParcelClient.CreateWebhookSubscription's doc
// comment for why this exists separately (the provider-webhooks admin
// feature, which lets an admin pick any of MyParcel's own hook types, not
// just shipment_status_change).
func (c *Client) CreateWebhookSubscription(ctx context.Context, hook, deliveryURL string) (WebhookSubscription, error) {
	var req webhookSubscriptionRequest
	req.Data.WebhookSubscriptions = []struct {
		Hook string `json:"hook"`
		URL  string `json:"url"`
	}{{Hook: hook, URL: deliveryURL}}
	var resp webhookSubscriptionListResponse
	if err := c.doJSON(ctx, http.MethodPost, "/webhook_subscriptions", "application/json", req, &resp); err != nil {
		return WebhookSubscription{}, err
	}
	if len(resp.Data.WebhookSubscriptions) == 0 {
		// MyParcel accepted the request but the response envelope wasn't
		// what this client expected (see webhookSubscriptionListResponse's
		// doc comment) -- return what we asked for rather than an empty
		// struct, since the subscription was still created.
		return WebhookSubscription{Hook: hook, URL: deliveryURL}, nil
	}
	s := resp.Data.WebhookSubscriptions[0]
	return WebhookSubscription{ID: strconv.Itoa(s.ID), Hook: s.Hook, URL: s.URL}, nil
}

// ListWebhookSubscriptions lists every subscription currently registered
// for this shop/API key, across all hook types.
func (c *Client) ListWebhookSubscriptions(ctx context.Context) ([]WebhookSubscription, error) {
	var resp webhookSubscriptionListResponse
	if err := c.doJSON(ctx, http.MethodGet, "/webhook_subscriptions", "application/json", nil, &resp); err != nil {
		return nil, err
	}
	out := make([]WebhookSubscription, len(resp.Data.WebhookSubscriptions))
	for i, s := range resp.Data.WebhookSubscriptions {
		out[i] = WebhookSubscription{ID: strconv.Itoa(s.ID), Hook: s.Hook, URL: s.URL}
	}
	return out, nil
}

// DeleteWebhookSubscription removes one subscription by its MyParcel id.
func (c *Client) DeleteWebhookSubscription(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/webhook_subscriptions/"+id, "application/json", nil, nil)
}
