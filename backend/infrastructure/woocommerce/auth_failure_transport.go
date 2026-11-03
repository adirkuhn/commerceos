package woocommerce

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// maxBodySnippet caps how much of a non-WordPress refusal body ends up in
// the error detail -- enough to recognise a firewall page, never a dump.
const maxBodySnippet = 160

// maxRefusalBody caps how much of a refusal body is read at all.
const maxRefusalBody = 64 << 10

// authFailureTransport turns a store's 401/403 into a typed domain error
// (see integrationsDomain.ErrWooAuthRejected and its siblings), for every
// request this adapter makes -- the SDK's and our own raw ones alike. A bare
// "403" can't tell a wrong key from a firewall ban (2026-11-03: CrowdSec);
// WordPress's own JSON error body can. It also turns WooCommerce's
// "invalid product id" answer into integrationsDomain.ErrWooProductNotFound,
// so a product deleted in the store is recognised (COM-18).
type authFailureTransport struct {
	next http.RoundTripper
}

func newAuthFailureTransport(next http.RoundTripper) http.RoundTripper {
	return &authFailureTransport{next: next}
}

func (t *authFailureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.next.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if mayBeMissingProduct(resp.StatusCode) {
		return classifyMissingProduct(resp)
	}
	if !isRefusal(resp.StatusCode) {
		return resp, nil
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxRefusalBody))
	if readErr != nil {
		return nil, fmt.Errorf("%w: status %d", integrationsDomain.ErrWooRequestFailed, resp.StatusCode)
	}
	return nil, classifyRefusal(resp.StatusCode, body)
}

func isRefusal(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

// missingProductCodes are WooCommerce's "no such product/variation" error
// codes (WC REST controllers: woocommerce_rest_{post_type}_invalid_id) --
// 404 on read, 400 on update.
var missingProductCodes = map[string]bool{
	"woocommerce_rest_product_invalid_id":           true,
	"woocommerce_rest_product_variation_invalid_id": true,
}

func mayBeMissingProduct(status int) bool {
	return status == http.StatusNotFound || status == http.StatusBadRequest
}

// classifyMissingProduct turns WooCommerce's invalid-id answer into
// ErrWooProductNotFound; any other 400/404 is handed back untouched, its
// body restored for the caller to read.
func classifyMissingProduct(resp *http.Response) (*http.Response, error) {
	// read whole: a non-matching body is handed back to the caller intact
	body, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("%w: status %d", integrationsDomain.ErrWooRequestFailed, resp.StatusCode)
	}
	var wp wordPressError
	if json.Unmarshal(body, &wp) == nil && missingProductCodes[wp.Code] {
		return nil, withDetail(integrationsDomain.ErrWooProductNotFound, resp.StatusCode, wp.Message)
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

// wordPressError is the body WordPress's REST API sends with every error.
type wordPressError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func classifyRefusal(status int, body []byte) error {
	var wp wordPressError
	if json.Unmarshal(body, &wp) != nil || wp.Code == "" {
		return withDetail(integrationsDomain.ErrWooBlocked, status, snippet(body))
	}
	if status == http.StatusUnauthorized {
		return withDetail(integrationsDomain.ErrWooAuthRejected, status, wp.Message)
	}
	return withDetail(integrationsDomain.ErrWooPermissionDenied, status, wp.Message)
}

func withDetail(sentinel error, status int, reason string) error {
	if reason == "" {
		return fmt.Errorf("%w: status %d", sentinel, status)
	}
	return fmt.Errorf("%w: status %d: %s", sentinel, status, reason)
}

func snippet(body []byte) string {
	s := strings.Join(strings.Fields(string(body)), " ")
	if len(s) <= maxBodySnippet {
		return s
	}
	return strings.ToValidUTF8(s[:maxBodySnippet], "") + "…"
}
