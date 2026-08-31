// Package pdok is a small client for PDOK's Locatieserver free-text
// search (https://api.pdok.nl/bzk/locatieserver/search/v3_1/free) --
// public, keyless, backed by the Dutch national BAG address registry.
// NL-only. No equivalent exists in MyParcel's own API reachable with a
// merchant API key (see domain/shipping.AddressLookupClient's doc
// comment).
//
// The endpoint is a *fuzzy* free-text search -- confirmed against real
// responses during this build that it always returns its single best
// guess even for a postal code + house number combination that doesn't
// exist (e.g. querying "9999ZZ 999" still returns a real address in a
// different city entirely, at a real but unrelated postcode). So Lookup's
// entire job is verifying the top hit's own postcode/huisnummer actually
// equal what was asked for before trusting it -- never just relaying
// whatever comes back first.
package pdok

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

const defaultBaseURL = "https://api.pdok.nl/bzk/locatieserver/search/v3_1/free"

// Config's BaseURL defaults to PDOK's real endpoint when empty --
// overridable only so tests can point at an httptest.Server.
type Config struct {
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
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{cfg: cfg, http: cfg.HTTPClient}
}

type freeSearchResponse struct {
	Response struct {
		Docs []struct {
			Postcode   string `json:"postcode"`
			HuisNLT    string `json:"huis_nlt"`
			Straatnaam string `json:"straatnaam"`
			Woonplaats string `json:"woonplaatsnaam"`
		} `json:"docs"`
	} `json:"response"`
}

// normalizePostcode strips spaces and uppercases -- PDOK's own postcode
// field has no space ("1035SH"); a human might type "1035 SH".
func normalizePostcode(s string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(s), " ", ""))
}

// normalizeHouseNumber lowercases and trims -- huis_nlt carries any letter
// suffix (e.g. "31a"), and case shouldn't count as a mismatch.
func normalizeHouseNumber(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func (c *Client) Lookup(ctx context.Context, postalCode, houseNumber string) (*shippingDomain.AddressLookupResult, error) {
	q := url.Values{}
	q.Set("q", postalCode+" "+houseNumber)
	q.Set("fq", "type:adres")
	q.Set("rows", "1")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("pdok: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pdok: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("pdok: request failed (%d)", resp.StatusCode)
	}

	var body freeSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("pdok: decode response: %w", err)
	}
	if len(body.Response.Docs) == 0 {
		return nil, shippingDomain.ErrAddressNotFound
	}

	doc := body.Response.Docs[0]
	if normalizePostcode(doc.Postcode) != normalizePostcode(postalCode) || normalizeHouseNumber(doc.HuisNLT) != normalizeHouseNumber(houseNumber) {
		return nil, shippingDomain.ErrAddressNotFound
	}

	return &shippingDomain.AddressLookupResult{Street: doc.Straatnaam, City: doc.Woonplaats}, nil
}
