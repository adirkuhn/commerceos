// Package moneybird is the Moneybird REST API v2 adapter for
// domain/moneybird.Client.
package moneybird

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	mb "github.com/adirkuhn/commerceos/domain/moneybird"
)

const baseURL = "https://moneybird.com/api/v2"

// sharedLimiter keeps the whole process under Moneybird's 150 requests per
// 5 minutes per IP: a burst of 10, then one call every 2.5s (~130/5min).
var sharedLimiter = newLimiter(10, 2500*time.Millisecond)

type factory struct{ http *http.Client }

// NewFactory builds clients against the live Moneybird API.
func NewFactory() mb.ClientFactory {
	return factory{http: &http.Client{Timeout: 20 * time.Second}}
}

func (f factory) New(apiToken, administrationID string) mb.Client {
	return newClient(baseURL, apiToken, administrationID, f.http, sharedLimiter)
}

type client struct {
	base    string
	token   string
	adminID string
	http    *http.Client
	limiter *limiter
}

func newClient(base, token, adminID string, httpClient *http.Client, l *limiter) *client {
	return &client{base: base, token: token, adminID: adminID, http: httpClient, limiter: l}
}

// do calls path (without ".json"), under the administration when the
// client has one, and decodes the JSON response into out (nil: ignored).
func (c *client) do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	u := c.base
	if c.adminID != "" {
		u += "/" + url.PathEscape(c.adminID)
	}
	u += path + ".json"
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	if err := c.limiter.Wait(ctx); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", mb.ErrRequestFailed, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%w: %w", mb.ErrRequestFailed, err)
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return mb.ErrRateLimited
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return mb.ErrTokenInvalid
	case resp.StatusCode >= 400:
		return fmt.Errorf("%w: %s %s: %d %s", mb.ErrRequestFailed, method, path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%w: decoding %s: %w", mb.ErrRequestFailed, path, err)
	}
	return nil
}

func (c *client) ListAdministrations(ctx context.Context) ([]mb.Administration, error) {
	var rows []struct{ ID, Name, Currency string }
	if err := c.do(ctx, http.MethodGet, "/administrations", nil, nil, &rows); err != nil {
		return nil, err
	}
	out := make([]mb.Administration, len(rows))
	for i, r := range rows {
		out[i] = mb.Administration{ID: r.ID, Name: r.Name, Currency: r.Currency}
	}
	return out, nil
}

func (c *client) ListSalesTaxRates(ctx context.Context) ([]mb.TaxRate, error) {
	var rows []struct {
		ID, Name, Percentage string
		Active               bool
	}
	q := url.Values{"filter": {"tax_rate_type:sales_invoice"}}
	if err := c.do(ctx, http.MethodGet, "/tax_rates", q, nil, &rows); err != nil {
		return nil, err
	}
	out := []mb.TaxRate{}
	for _, r := range rows {
		if r.Active {
			out = append(out, mb.TaxRate{ID: r.ID, Name: r.Name, Percentage: r.Percentage})
		}
	}
	return out, nil
}

func (c *client) ListRevenueLedgerAccounts(ctx context.Context) ([]mb.LedgerAccount, error) {
	var rows []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		AccountType string `json:"account_type"`
	}
	if err := c.do(ctx, http.MethodGet, "/ledger_accounts", nil, nil, &rows); err != nil {
		return nil, err
	}
	out := []mb.LedgerAccount{}
	for _, r := range rows {
		if r.AccountType == "revenue" {
			out = append(out, mb.LedgerAccount{ID: r.ID, Name: r.Name, AccountType: r.AccountType})
		}
	}
	return out, nil
}
