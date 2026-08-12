package scannerauth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genscannerauth "github.com/adirkuhn/commerceos/api/scannerauth"
	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/infrastructure/handlers/scannerauth"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

type stubClaimPairingCode struct {
	out          string
	err          error
	gotInput     string
	byTokenOut   string
	byTokenErr   error
	byTokenInput string
}

func (s *stubClaimPairingCode) Execute(_ context.Context, code string) (string, error) {
	s.gotInput = code
	return s.out, s.err
}

func (s *stubClaimPairingCode) ExecuteByToken(_ context.Context, token string) (string, error) {
	s.byTokenInput = token
	return s.byTokenOut, s.byTokenErr
}

func newTestRouter(h *scannerauth.Handler) *gin.Engine {
	r := gin.New()
	genscannerauth.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func newHandler(claim *stubClaimPairingCode) *scannerauth.Handler {
	return scannerauth.NewHandler(claim, middleware.NewIPRateLimiter(10, time.Minute))
}

func doRequest(rg *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(body)
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func TestClaimPairingCodeHandler_success(t *testing.T) {
	claim := &stubClaimPairingCode{out: "signed-jwt-token"}
	h := newHandler(claim)
	w := doRequest(newTestRouter(h), "/api/v1/iam/scanner-pairing/claim", map[string]any{"code": "123456"})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "123456", claim.gotInput)
	assert.Contains(t, w.Body.String(), "signed-jwt-token")
}

func TestClaimPairingCodeHandler_invalidCodeReturns422(t *testing.T) {
	claim := &stubClaimPairingCode{err: iamDomain.ErrPairingCodeInvalid}
	h := newHandler(claim)
	w := doRequest(newTestRouter(h), "/api/v1/iam/scanner-pairing/claim", map[string]any{"code": "000000"})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestClaimPairingCodeHandler_rateLimitsRepeatedAttemptsFromSameIP(t *testing.T) {
	claim := &stubClaimPairingCode{err: iamDomain.ErrPairingCodeInvalid}
	h := scannerauth.NewHandler(claim, middleware.NewIPRateLimiter(3, time.Minute))
	router := newTestRouter(h)

	var last *httptest.ResponseRecorder
	for i := 0; i < 4; i++ {
		last = doRequest(router, "/api/v1/iam/scanner-pairing/claim", map[string]any{"code": "000000"})
	}

	assert.Equal(t, http.StatusTooManyRequests, last.Code, "4th attempt within the window must be rate-limited")
}

func TestClaimPairingCodeByTokenHandler_success(t *testing.T) {
	claim := &stubClaimPairingCode{byTokenOut: "signed-jwt-token"}
	h := newHandler(claim)
	w := doRequest(newTestRouter(h), "/api/v1/iam/scanner-pairing/claim-token", map[string]any{"token": "opaque-abc"})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "opaque-abc", claim.byTokenInput)
	assert.Contains(t, w.Body.String(), "signed-jwt-token")
}

func TestClaimPairingCodeByTokenHandler_invalidTokenReturns422(t *testing.T) {
	claim := &stubClaimPairingCode{byTokenErr: iamDomain.ErrPairingCodeInvalid}
	h := newHandler(claim)
	w := doRequest(newTestRouter(h), "/api/v1/iam/scanner-pairing/claim-token", map[string]any{"token": "does-not-exist"})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestClaimPairingCodeByTokenHandler_notRateLimited(t *testing.T) {
	claim := &stubClaimPairingCode{byTokenErr: iamDomain.ErrPairingCodeInvalid}
	h := scannerauth.NewHandler(claim, middleware.NewIPRateLimiter(3, time.Minute))
	router := newTestRouter(h)

	var last *httptest.ResponseRecorder
	for i := 0; i < 6; i++ {
		last = doRequest(router, "/api/v1/iam/scanner-pairing/claim-token", map[string]any{"token": "does-not-exist"})
	}

	assert.Equal(t, http.StatusUnprocessableEntity, last.Code, "token path must never be rate-limited -- unguessable within TTL")
}
