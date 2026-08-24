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
// BaseURL default to Bol's real production endpoints when empty --
// overridable only so tests can point at an httptest.Server.
type Config struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	BaseURL      string
	HTTPClient   *http.Client
}

const (
	defaultTokenURL = "https://login.bol.com/token"
	defaultBaseURL  = "https://api.bol.com/retailer"

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

	req, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+path, bodyReader)
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

// --- Offer API (v10, async -- POST/PUT return 202 + a ProcessStatus,
// same shape as Product Content below, poll to a terminal status and read
// entityId for the real offer id). Previously implemented against a wrong
// guessed path/version/method (/v11/offers, PATCH) that also assumed a
// synchronous {"offerId":...} response -- confirmed wrong against the real
// downloaded Retailer API v10 spec: real path is /offers (POST create, PUT
// update by id), real header is v10 not v11, real response is 202 Accepted.
// That mismatch meant CreateOffer/UpdateOffer never errored but also never
// created a real offer -- silent no-op in prod. ---

type BundlePrice struct {
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
}

type Pricing struct {
	BundlePrices []BundlePrice `json:"bundlePrices"`
}

type Fulfilment struct {
	Method string `json:"method"`
	// DeliveryCode is required by Bol for method FBR (see this const
	// block's Retailer API v10 header) -- omitted for FBB, where Bol
	// controls delivery timing itself.
	DeliveryCode string `json:"deliveryCode,omitempty"`
}

type Condition struct {
	// Name is required by Bol (enum NEW/AS_NEW/GOOD/REASONABLE/MODERATE).
	// Category is optional -- Bol derives NEW/SECONDHAND from Name when
	// omitted, so this client never sets it separately.
	Name string `json:"name"`
}

type Stock struct {
	Amount            int  `json:"amount"`
	ManagedByRetailer bool `json:"managedByRetailer"`
}

type CreateOfferRequest struct {
	EAN                   string     `json:"ean"`
	UnknownProductTitle   string     `json:"unknownProductTitle,omitempty"`
	EconomicOperatorID    string     `json:"economicOperatorId"`
	Reference             string     `json:"reference,omitempty"`
	OnHoldByRetailer      bool       `json:"onHoldByRetailer,omitempty"`
	Condition             Condition  `json:"condition"`
	Pricing               Pricing    `json:"pricing"`
	CountryAvailabilities []string   `json:"countryAvailabilities,omitempty"`
	Fulfilment            Fulfilment `json:"fulfilment"`
	Stock                 *Stock     `json:"stock,omitempty"`
}

// CreateOffer creates a new Bol offer for req.EAN. The POST only returns a
// ProcessStatus (202 Accepted) -- this polls it to a terminal status and
// returns the resulting entityId as the real offer id.
func (c *Client) CreateOffer(ctx context.Context, req CreateOfferRequest, opts PollOptions) (string, error) {
	var accepted processStatusResponse
	if err := c.doJSON(ctx, http.MethodPost, "/offers", contentAccept, req, &accepted); err != nil {
		return "", err
	}
	return c.pollOfferProcessStatus(ctx, accepted.ProcessStatusID, opts)
}

// UpdateOffer replaces an existing offer's price/stock/etc (PUT, not
// PATCH -- Bol has no partial-update verb for offers). Same async
// ProcessStatus response as CreateOffer.
func (c *Client) UpdateOffer(ctx context.Context, offerID string, req CreateOfferRequest, opts PollOptions) error {
	var accepted processStatusResponse
	if err := c.doJSON(ctx, http.MethodPut, "/offers/"+url.PathEscape(offerID), contentAccept, req, &accepted); err != nil {
		return err
	}
	_, err := c.pollOfferProcessStatus(ctx, accepted.ProcessStatusID, opts)
	return err
}

// pollOfferProcessStatus mirrors SubmitProductContent's polling loop below,
// but reads entityId on SUCCESS since offer creation/update needs the
// resulting offer id back, unlike a product content upload.
func (c *Client) pollOfferProcessStatus(ctx context.Context, processStatusID string, opts PollOptions) (string, error) {
	opts = opts.withDefaults()
	if processStatusID == "" {
		return "", fmt.Errorf("bol: offer request accepted with no process status id")
	}

	for attempt := 0; attempt < opts.MaxAttempts; attempt++ {
		var report processStatusResponse
		path := "/process-status/" + url.PathEscape(processStatusID)
		if err := c.doJSON(ctx, http.MethodGet, path, contentAccept, nil, &report); err != nil {
			return "", fmt.Errorf("bol: poll offer process status: %w", err)
		}
		switch strings.ToUpper(report.Status) {
		case "SUCCESS":
			return report.EntityID, nil
		case "FAILURE", "TIMEOUT":
			msg := report.ErrorMessage
			if msg == "" {
				msg = report.Status
			}
			return "", fmt.Errorf("bol: offer request %s ended with status %s: %s", processStatusID, report.Status, msg)
		}
		time.Sleep(opts.Interval)
	}
	return "", fmt.Errorf("bol: offer request %s did not reach a terminal status after %d attempts", processStatusID, opts.MaxAttempts)
}

// --- Product Content API (v10, async -- upload then poll a report) ---

type AttributeValue struct {
	Value string `json:"value"`
}

type ContentAttribute struct {
	ID     string           `json:"id"`
	Values []AttributeValue `json:"values"`
}

type Asset struct {
	URL    string `json:"url"`
	Labels string `json:"labels"`
}

type ProductContentRequest struct {
	Language   string             `json:"language"`
	Attributes []ContentAttribute `json:"attributes"`
	Assets     []Asset            `json:"assets,omitempty"`
}

type processStatusResponse struct {
	ProcessStatusID string `json:"processStatusId"`
	EntityID        string `json:"entityId"`
	Status          string `json:"status"`
	ErrorMessage    string `json:"errorMessage"`
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

// SubmitProductContent uploads new/updated product content, then polls the
// upload report until it reaches a terminal status. Bol requires a GPC Code
// + enrichment-level-0/1 mandatory attributes for a genuinely new EAN --
// this client does not validate or invent those; the caller
// (PushProductToBolUseCase) is responsible for populating req.Attributes
// correctly. The exact mandatory-attribute set is category-dependent (Bol's
// "chunks" taxonomy) and was NOT confirmed against a real sandbox during
// this build -- verify before relying on this for a genuinely new category.
func (c *Client) SubmitProductContent(ctx context.Context, req ProductContentRequest, opts PollOptions) error {
	opts = opts.withDefaults()

	var upload processStatusResponse
	if err := c.doJSON(ctx, http.MethodPost, "/product-content", contentAccept, req, &upload); err != nil {
		return fmt.Errorf("bol: submit product content: %w", err)
	}
	if upload.ProcessStatusID == "" {
		return nil // some responses may be synchronous already-successful, nothing to poll
	}

	for attempt := 0; attempt < opts.MaxAttempts; attempt++ {
		var report processStatusResponse
		path := "/product-content/" + url.PathEscape(upload.ProcessStatusID)
		if err := c.doJSON(ctx, http.MethodGet, path, contentAccept, nil, &report); err != nil {
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
