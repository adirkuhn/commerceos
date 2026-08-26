// Package bol is a small, self-built client for Bol.com's Retailer API
// (OAuth2 client_credentials, Offer API v11, Product Content API v10,
// Commissions API v10). No approved third-party Go SDK exists for this API
// -- built to spec against https://api.bol.com/retailer/public/Retailer-API/
// fetched during design. The exact Commissions endpoint path and the exact
// mandatory Product Content attribute set per category were NOT fully
// resolvable from the docs fetched (Redoc-only detail); both are marked
// where they matter below and need verification against a real Bol sandbox
// before this goes live -- see GetCommission and SubmitProductContent's doc
// comments.
package bol

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config holds one Bol connection's credentials + endpoints. TokenURL/
// BaseURL/SharedBaseURL default to Bol's real production endpoints when
// empty -- overridable only so tests can point at an httptest.Server.
type Config struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	BaseURL      string
	// SharedBaseURL is Bol's cross-resource "shared" API (currently just
	// process-status) -- a genuinely different path root from BaseURL
	// (/shared/... not /retailer/...), confirmed against the real spec at
	// api.bol.com/registry/api-definitions/process-status/process-status-v10.yaml.
	// Not a sibling of /retailer under the same prefix, so it can't be
	// expressed as a path relative to BaseURL.
	SharedBaseURL string
	HTTPClient    *http.Client
}

const (
	defaultTokenURL      = "https://login.bol.com/token"
	defaultBaseURL       = "https://api.bol.com/retailer"
	defaultSharedBaseURL = "https://api.bol.com/shared"

	contentAccept = "application/vnd.retailer.v10+json"
)

type Client struct {
	cfg  Config
	http *http.Client

	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

func NewClient(cfg Config) *Client {
	if cfg.TokenURL == "" {
		cfg.TokenURL = defaultTokenURL
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.SharedBaseURL == "" {
		cfg.SharedBaseURL = defaultSharedBaseURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{cfg: cfg, http: cfg.HTTPClient}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// AccessToken returns a cached token if it's still valid for at least
// tokenRefreshMargin, otherwise fetches a new one. Bol issues no refresh
// token (see design research) -- expiry just means "call /token again."
const tokenRefreshMargin = 15 * time.Second

func (c *Client) AccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Add(tokenRefreshMargin).Before(c.tokenExp) {
		return c.token, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.TokenURL, strings.NewReader("grant_type=client_credentials"))
	if err != nil {
		return "", fmt.Errorf("bol: build token request: %w", err)
	}
	basic := base64.StdEncoding.EncodeToString([]byte(c.cfg.ClientID + ":" + c.cfg.ClientSecret))
	req.Header.Set("Authorization", "Basic "+basic)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("bol: token request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bol: token request failed (%d): %s", resp.StatusCode, string(body))
	}

	var tok tokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", fmt.Errorf("bol: decode token response: %w", err)
	}
	c.token = tok.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	return c.token, nil
}

// APIError wraps a non-2xx Bol response, kept as its own type so callers
// (PushProductToBolUseCase) can classify specific failures via IsUnknownEAN
// without string-matching the raw body directly.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("bol: request failed (%d): %s", e.StatusCode, e.Body)
}

// IsUnknownEAN reports whether err is a Bol API error indicating the EAN
// isn't in Bol's catalog yet -- Bol's exact violation-reason wording for
// this case wasn't confirmed against a real sandbox response during this
// build (only the 400/violations response SHAPE was), so this matches on
// "ean" and "unknown" case-insensitively in the body rather than a specific
// reason string. Verify this against a real unknown-EAN response and
// tighten the match if it's too loose/too strict.
func IsUnknownEAN(err error) bool {
	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}
	lower := strings.ToLower(apiErr.Body)
	return strings.Contains(lower, "ean") && strings.Contains(lower, "unknown")
}

func (c *Client) doJSON(ctx context.Context, method, path, accept string, reqBody, respBody any) error {
	return c.doJSONWithBase(ctx, c.cfg.BaseURL, method, path, accept, reqBody, respBody)
}

// doJSONWithBase is doJSON with an explicit base URL, for the one resource
// (process-status) that lives under a different path root than
// Config.BaseURL -- see SharedBaseURL's doc comment.
func (c *Client) doJSONWithBase(ctx context.Context, base, method, path, accept string, reqBody, respBody any) error {
	token, err := c.AccessToken(ctx)
	if err != nil {
		return err
	}

	var bodyReader io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("bol: encode request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, base+path, bodyReader)
	if err != nil {
		return fmt.Errorf("bol: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", accept)
	if reqBody != nil {
		req.Header.Set("Content-Type", accept)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("bol: request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	if respBody != nil && len(body) > 0 {
		if err := json.Unmarshal(body, respBody); err != nil {
			return fmt.Errorf("bol: decode response for %s %s: %w", method, path, err)
		}
	}
	return nil
}

// --- Offer API (v11, current -- fully synchronous). Confirmed against
// Bol's real, currently-published Offers API v11.5.0 spec
// (api.bol.com/registry/api-definitions/offers/offers-v11.yaml), found via
// api.bol.com/retailer/public/Retailer-API's own docs navigation, which
// lists "Offers API v11 (current)" as a distinct, separately-versioned API
// from "v10 (deprecated)". This client previously targeted v10 (a wrong
// downgrade from an earlier, also-wrong /v11/offers guess): it accepted
// requests (202 + a real processStatusId) but polling /process-status/{id}
// 404'd in prod -- the deprecated version's documented async completion
// tracking doesn't actually resolve. v11 has no such problem: POST/PATCH
// return the result directly, nothing to poll.
//
// v11's Condition is a discriminated union (NewCondition/
// SecondhandCondition/RefurbishedCondition, keyed by category) -- only
// Category is modeled here since this codebase only ever pushes NEW
// condition products; State/Comment/Grade/Margin would need adding if
// SECONDHAND/REFURBISHED support is ever built.

const offerAccept = "application/vnd.retailer.v11+json"

type BundlePrice struct {
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
}

type Pricing struct {
	BundlePrices []BundlePrice `json:"bundlePrices"`
}

type Fulfilment struct {
	Method string `json:"method"` // FBR or FBB
	// Schedule is required by Bol when Method is FBR (confirmed via a real
	// 400: "Field 'schedule' is required when 'method' is FBR" -- not
	// visible as schema-required in the static v11.5.0 spec, same kind of
	// business-required-but-not-schema-required gap as economicOperatorId).
	// Enum: BOL_DELIVERY_PROMISE (bol calculates it), MY_DELIVERY_PROMISE
	// (retailer's own profile via the separate Delivery Promise API, not
	// integrated here), SHIPPING_VIA_BOL. Always BOL_DELIVERY_PROMISE for
	// now -- the simplest correct choice with no extra setup required.
	Schedule string `json:"schedule,omitempty"`
	// DeliveryPromise is required by Bol whenever Schedule is
	// BOL_DELIVERY_PROMISE (confirmed via a real 400: "Field
	// 'deliveryPromise' is required when 'method' = FBR and 'schedule' is
	// BOL_DELIVERY_PROMISE" -- again not schema-required in the static
	// spec).
	DeliveryPromise *DeliveryPromise `json:"deliveryPromise,omitempty"`
}

type DeliveryPromise struct {
	MinimumDaysToCustomer int `json:"minimumDaysToCustomer"`
	MaximumDaysToCustomer int `json:"maximumDaysToCustomer"`
}

type Condition struct {
	Category string `json:"category"`
}

type Stock struct {
	Amount            int  `json:"amount"`
	ManagedByRetailer bool `json:"managedByRetailer"`
}

type CreateOfferRequest struct {
	EAN                 string     `json:"ean"`
	UnknownProductTitle string     `json:"unknownProductTitle,omitempty"`
	EconomicOperatorID  string     `json:"economicOperatorId,omitempty"`
	Reference           string     `json:"reference,omitempty"`
	OnHoldByRetailer    bool       `json:"onHoldByRetailer,omitempty"`
	Condition           Condition  `json:"condition"`
	Pricing             Pricing    `json:"pricing"`
	Fulfilment          Fulfilment `json:"fulfilment"`
	Stock               *Stock     `json:"stock,omitempty"`
}

type offerIDResponse struct {
	OfferID string `json:"offerId"`
}

// CreateOffer creates a new Bol offer for req.EAN, returning the new
// offer's id -- synchronous (201 Created) under v11, no polling.
func (c *Client) CreateOffer(ctx context.Context, req CreateOfferRequest) (string, error) {
	var resp offerIDResponse
	if err := c.doJSON(ctx, http.MethodPost, "/offers", offerAccept, req, &resp); err != nil {
		return "", err
	}
	return resp.OfferID, nil
}

// PatchOfferRequest updates an existing offer. Deliberately narrower than
// CreateOfferRequest -- Bol's real PatchOfferRequest schema has no
// condition field at all (an offer's condition can't be changed after
// creation).
type PatchOfferRequest struct {
	EconomicOperatorID string      `json:"economicOperatorId,omitempty"`
	Reference          string      `json:"reference,omitempty"`
	Pricing            *Pricing    `json:"pricing,omitempty"`
	Fulfilment         *Fulfilment `json:"fulfilment,omitempty"`
	Stock              *Stock      `json:"stock,omitempty"`
}

// UpdateOffer patches an existing offer's price/stock/etc -- PATCH, not
// PUT (confirmed against the real v11 spec: Bol has no PUT for offers).
// Synchronous (204 No Content).
func (c *Client) UpdateOffer(ctx context.Context, offerID string, req PatchOfferRequest) error {
	return c.doJSON(ctx, http.MethodPatch, "/offers/"+url.PathEscape(offerID), offerAccept, req, nil)
}

type offerListResponse struct {
	Offers []offerSummary `json:"offers"`
}

type offerSummary struct {
	OfferID string `json:"offerId"`
	EAN     string `json:"ean"`
}

// FindOfferByEAN looks up whether an offer already exists on Bol for this
// EAN -- lets PushProductToBolUseCase link and update an offer that
// predates this integration (or survived a lost local listing row)
// instead of blindly POSTing and hitting Bol's 409 Conflict for a
// duplicate offer. Empty string + false (not an error) means no offer
// exists yet for this EAN.
func (c *Client) FindOfferByEAN(ctx context.Context, ean string) (string, bool, error) {
	var resp offerListResponse
	path := "/offers?eans=" + url.QueryEscape(ean)
	if err := c.doJSON(ctx, http.MethodGet, path, offerAccept, nil, &resp); err != nil {
		return "", false, err
	}
	if len(resp.Offers) == 0 {
		return "", false, nil
	}
	return resp.Offers[0].OfferID, true, nil
}

// NotForSaleReasons and its nested types mirror the real v11 spec exactly
// (api.bol.com/registry/api-definitions/offers/offers-v11.yaml,
// operationId get-not-for-sale-reasons, schema NotForSaleReasons) --
// confirmed 2026-08-25 while investigating why a real offer stayed
// "offline" with no error anywhere in this client. Reason is a
// discriminated union in the spec (oneOf ContentReason/PricingReason/
// StockReason/... keyed by "category"); this client only cares about
// Category/Code/Description/MissingAttributes, which every variant has in
// common (MissingAttributes is empty for non-CONTENT reasons), so it's
// modeled as one flat struct rather than a Go type switch -- simpler, and
// nothing here needs the reason-specific fields the other variants might
// carry.
type NotForSaleReasons struct {
	OfferID   string                     `json:"offerId"`
	Countries []NotForSaleReasonsCountry `json:"countries"`
}

type NotForSaleReasonsCountry struct {
	CountryCode string   `json:"countryCode"`
	Reasons     []Reason `json:"reasons"`
}

type Reason struct {
	Category          string             `json:"category"`
	Code              string             `json:"code"`
	Description       string             `json:"description"`
	MissingAttributes []MissingAttribute `json:"missingAttributes,omitempty"`
}

type MissingAttribute struct {
	Name string `json:"name"`
}

// GetNotForSaleReasons reports why an offer is currently not for sale
// (offline), per country -- empty Countries means the offer is for sale
// everywhere. Read-only, safe to call after every push to catch content
// Bol silently rejected (e.g. a still-missing mandatory attribute) that a
// 202-accepted content submission would never otherwise surface.
func (c *Client) GetNotForSaleReasons(ctx context.Context, offerID string) (NotForSaleReasons, error) {
	var resp NotForSaleReasons
	path := "/offers/" + url.PathEscape(offerID) + "/not-for-sale-reasons"
	if err := c.doJSON(ctx, http.MethodGet, path, offerAccept, nil, &resp); err != nil {
		return NotForSaleReasons{}, err
	}
	return resp, nil
}

// --- Product Content API (v10, async -- upload then poll a report) ---

type AttributeValue struct {
	Value string `json:"value"`
	// UnitID is Bol's unit-of-measure code for this value (e.g.
	// "unece.unit.MLT" for ml, "unece.unit.MMT" for mm) -- required for
	// number-typed attributes that have a units list in Bol's data model
	// (Capacity, Width/Height/Length, Weight, Number Pieces In Package).
	// Confirmed against the real spec's AttributeValue schema (unitId,
	// example "mm") and per-attribute unit lists in
	// productdatamodel.s-bol.com/v10/datamodel_v10_en.json. Empty/omitted
	// for text, boolean, and LOV-select attributes, which carry no unit.
	UnitID string `json:"unitId,omitempty"`
}

type ContentAttribute struct {
	ID     string           `json:"id"`
	Values []AttributeValue `json:"values"`
}

// Asset.Labels is an array in Bol's real spec (maxItems 2, example
// ["FRONT"]), not a bare string -- confirmed 2026-08-25 alongside the
// product-content path fix. A bare string previously silently sent the
// wrong JSON shape ("labels":"FRONT" instead of "labels":["FRONT"]), the
// same silent-drop failure mode as a wrong attribute id.
type Asset struct {
	URL    string   `json:"url"`
	Labels []string `json:"labels"`
}

type ProductContentRequest struct {
	Language   string             `json:"language"`
	Attributes []ContentAttribute `json:"attributes"`
	Assets     []Asset            `json:"assets,omitempty"`
}

type processStatusResponse struct {
	ProcessStatusID string `json:"processStatusId"`
	Status          string `json:"status"`
}

type PollOptions struct {
	Interval    time.Duration
	MaxAttempts int
}

func (o PollOptions) withDefaults() PollOptions {
	if o.Interval <= 0 {
		o.Interval = 2 * time.Second
	}
	if o.MaxAttempts <= 0 {
		o.MaxAttempts = 30
	}
	return o
}

// SubmitProductContent uploads new/updated product content, then polls
// process-status until it reaches a terminal status. Bol requires a GPC Code
// + enrichment-level-0/1 mandatory attributes for a genuinely new EAN --
// this client does not validate or invent those; the caller
// (PushProductToBolUseCase) is responsible for populating req.Attributes
// correctly. The exact mandatory-attribute set is category-dependent (Bol's
// "chunks" taxonomy) and was NOT confirmed against a real sandbox during
// this build -- verify before relying on this for a genuinely new category.
//
// Path is POST /content/products (plural). Confirmed 2026-08-25 by
// downloading and inspecting the real, currently-published spec directly --
// api.bol.com/registry/api-definitions/product-content/product-content-v10.yaml,
// operationId post-product-content, path "/retailer/content/products"
// (server root is bare api.bol.com; this client's BaseURL already includes
// /retailer, hence the relative "/content/products" here). Two earlier
// guesses both 404'd in prod: "/product-content" (wrong resource name) and
// then singular "/content/product" (right resource, wrong pluralization --
// taken from a third-party generated client's docs, which was stale/wrong).
// Lesson: don't trust community client docs for this API, download the
// registry spec directly.
//
// Process-status polling hits a genuinely different path root: confirmed
// against api.bol.com/registry/api-definitions/process-status/process-status-v10.yaml,
// path "/shared/process-status/{id}" -- under /shared, not /retailer, so it
// cannot be expressed relative to BaseURL and uses SharedBaseURL instead.
func (c *Client) SubmitProductContent(ctx context.Context, req ProductContentRequest, opts PollOptions) error {
	opts = opts.withDefaults()

	var upload processStatusResponse
	if err := c.doJSON(ctx, http.MethodPost, "/content/products", contentAccept, req, &upload); err != nil {
		return fmt.Errorf("bol: submit product content: %w", err)
	}
	if upload.ProcessStatusID == "" {
		return nil // some responses may be synchronous already-successful, nothing to poll
	}

	for attempt := 0; attempt < opts.MaxAttempts; attempt++ {
		var report processStatusResponse
		path := "/process-status/" + url.PathEscape(upload.ProcessStatusID)
		if err := c.doJSONWithBase(ctx, c.cfg.SharedBaseURL, http.MethodGet, path, contentAccept, nil, &report); err != nil {
			return fmt.Errorf("bol: poll product content report: %w", err)
		}
		switch strings.ToUpper(report.Status) {
		case "SUCCESS":
			return nil
		case "FAILURE", "TIMEOUT":
			return fmt.Errorf("bol: product content upload %s ended with status %s", upload.ProcessStatusID, report.Status)
		}
		time.Sleep(opts.Interval)
	}
	return fmt.Errorf("bol: product content upload %s did not reach a terminal status after %d attempts", upload.ProcessStatusID, opts.MaxAttempts)
}

// --- Commissions API (v10) ---

type Commission struct {
	EAN         string  `json:"ean"`
	FixedAmount float64 `json:"fixedAmount"`
	Percentage  float64 `json:"percentage"`
}

// GetCommission fetches the live commission for one EAN at unitPrice.
// The exact endpoint path below (GET /commission/{ean}?unit-price=) follows
// Bol's general Retailer API path conventions but was NOT confirmed
// verbatim against the Redoc spec during this build (that page only
// resolved to a Redoc anchor, not a literal path) -- verify against a real
// sandbox call before relying on this in production; if the real path
// differs, only this method's path/query needs to change, nothing else in
// this client depends on it.
func (c *Client) GetCommission(ctx context.Context, ean string, unitPrice float64) (Commission, error) {
	var resp Commission
	path := "/commission/" + url.PathEscape(ean) + "?unit-price=" + strconv.FormatFloat(unitPrice, 'f', 2, 64)
	if err := c.doJSON(ctx, http.MethodGet, path, contentAccept, nil, &resp); err != nil {
		return Commission{}, err
	}
	return resp, nil
}

// --- Orders API (v10) ---
// Confirmed against Bol's real Retailer API v10 OpenAPI spec (downloaded
// and inspected directly, unlike GetCommission's path above which was a
// verified-correct guess): GET /orders (paginated, 50/page, reduced shape
// -- no shipment/billing details, no per-item price) and GET /orders/{id}
// (single order, full shape). Neither supports a "modified since" cursor
// -- only a status filter -- so ImportBolOrdersUseCase polls status=OPEN
// every run rather than tracking an incremental cursor.

type reducedOrdersResponse struct {
	Orders []reducedOrder `json:"orders"`
}

type reducedOrder struct {
	OrderID             string             `json:"orderId"`
	OrderPlacedDateTime time.Time          `json:"orderPlacedDateTime"`
	OrderItems          []reducedOrderItem `json:"orderItems"`
}

type reducedOrderItem struct {
	OrderItemID         string `json:"orderItemId"`
	EAN                 string `json:"ean"`
	Quantity            int    `json:"quantity"`
	QuantityShipped     int    `json:"quantityShipped"`
	QuantityCancelled   int    `json:"quantityCancelled"`
	CancellationRequest bool   `json:"cancellationRequest"`
}

type orderResponse struct {
	OrderID             string          `json:"orderId"`
	OrderPlacedDateTime time.Time       `json:"orderPlacedDateTime"`
	ShipmentDetails     shipmentDetails `json:"shipmentDetails"`
	OrderItems          []orderItem     `json:"orderItems"`
}

type shipmentDetails struct {
	FirstName   string `json:"firstName"`
	Surname     string `json:"surname"`
	StreetName  string `json:"streetName"`
	HouseNumber string `json:"houseNumber"`
	ZipCode     string `json:"zipCode"`
	City        string `json:"city"`
	CountryCode string `json:"countryCode"`
	Email       string `json:"email"`
}

type orderItem struct {
	OrderItemID         string       `json:"orderItemId"`
	CancellationRequest bool         `json:"cancellationRequest"`
	Offer               orderOffer   `json:"offer"`
	Product             orderProduct `json:"product"`
	Quantity            int          `json:"quantity"`
	QuantityShipped     int          `json:"quantityShipped"`
	QuantityCancelled   int          `json:"quantityCancelled"`
	UnitPrice           float64      `json:"unitPrice"`
}

type orderOffer struct {
	Reference string `json:"reference"`
}

type orderProduct struct {
	EAN   string `json:"ean"`
	Title string `json:"title"`
}

// ListOrders returns one page of orders (Bol's fixed 50/page) filtered by
// status ("OPEN", "SHIPPED", or "ALL"). Empty slice (not an error) means no
// more pages -- caller stops paging.
func (c *Client) ListOrders(ctx context.Context, page int, status string) ([]reducedOrder, error) {
	var resp reducedOrdersResponse
	path := "/orders?page=" + strconv.Itoa(page) + "&status=" + url.QueryEscape(status) + "&fulfilment-method=FBR"
	if err := c.doJSON(ctx, http.MethodGet, path, contentAccept, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// GetOrder returns one order's full detail, including ShipmentDetails and
// per-item pricing -- neither present in ListOrders' reduced shape.
func (c *Client) GetOrder(ctx context.Context, orderID string) (*orderResponse, error) {
	var resp orderResponse
	path := "/orders/" + url.PathEscape(orderID)
	if err := c.doJSON(ctx, http.MethodGet, path, contentAccept, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- Economic Operator API (v1) ---
// Confirmed against Bol's real Economic Operator API v1 spec
// (api.bol.com/registry/api-definitions/economic-operators/economic-operators-v1.yaml,
// same /retailer base path this client already uses, different media type
// -- economicOperatorAccept below, not contentAccept). Only the list
// endpoint is used here (a "fetch from Bol" convenience for the
// Economic Operator ID field); create/update/delete/get-by-id are not
// needed by this codebase.

const economicOperatorAccept = "application/vnd.economic-operator.v1+json"

type economicOperatorResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type economicOperatorsPageResponse struct {
	Operators []economicOperatorResponse `json:"operators"`
}

// ListEconomicOperators returns every GPSR economic operator registered
// against this Bol account. Requires the economic-operators.read (or
// .manage) OAuth scope -- a 403 here means the app registration behind
// this connection's client_id/secret hasn't been granted that scope in
// Bol's Partner Platform, not a bug in this client.
func (c *Client) ListEconomicOperators(ctx context.Context) ([]economicOperatorResponse, error) {
	var resp economicOperatorsPageResponse
	if err := c.doJSON(ctx, http.MethodGet, "/economic-operators", economicOperatorAccept, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Operators, nil
}
