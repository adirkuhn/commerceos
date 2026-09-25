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
