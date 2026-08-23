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

	offerAccept   = "application/vnd.retailer.v11+json"
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

// --- Offer API (v11, synchronous) ---

type BundlePrice struct {
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
}

type Pricing struct {
	BundlePrices []BundlePrice `json:"bundlePrices"`
}

type Fulfilment struct {
	Method string `json:"method"`
}

type Condition struct {
	Category string `json:"category"`
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

type offerResponse struct {
	OfferID string `json:"offerId"`
}

// CreateOffer creates a new Bol offer for req.EAN, returning the offer id.
// Synchronous per Bol's v11 docs -- no process-status polling needed.
func (c *Client) CreateOffer(ctx context.Context, req CreateOfferRequest) (string, error) {
	var resp offerResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v11/offers", offerAccept, req, &resp); err != nil {
		return "", err
	}
	return resp.OfferID, nil
}

// UpdateOffer patches an existing offer (price/stock/etc changes).
func (c *Client) UpdateOffer(ctx context.Context, offerID string, req CreateOfferRequest) error {
	return c.doJSON(ctx, http.MethodPatch, "/v11/offers/"+url.PathEscape(offerID), offerAccept, req, nil)
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
