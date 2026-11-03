package woocommerce

import (
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
// WordPress's own JSON error body can.
type authFailureTransport struct {
	next http.RoundTripper
}

func newAuthFailureTransport(next http.RoundTripper) http.RoundTripper {
	return &authFailureTransport{next: next}
}

func (t *authFailureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.next.RoundTrip(req)
	if err != nil || !isRefusal(resp.StatusCode) {
		return resp, err
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
