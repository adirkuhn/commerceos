package scanning_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genscanning "github.com/adirkuhn/commerceos/api/scanning"
	appiam "github.com/adirkuhn/commerceos/application/iam"
	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/infrastructure/handlers/scanning"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

type stubGeneratePairingCode struct {
	out      *iamDomain.ScannerPairingCode
	err      error
	gotInput appiam.GeneratePairingCodeInput
}

func (s *stubGeneratePairingCode) Execute(_ context.Context, in appiam.GeneratePairingCodeInput) (*iamDomain.ScannerPairingCode, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &iamDomain.ScannerPairingCode{
		ID: "pairing-1", Code: "123456", UserID: in.UserID, Role: in.Role,
		Status: iamDomain.ScannerPairingCodeStatusPending, ExpiresAt: time.Now().Add(5 * time.Minute), CreatedAt: time.Now(),
	}, nil
}

type stubGetPairingCodeStatus struct {
	out *iamDomain.ScannerPairingCode
	err error
}

func (s *stubGetPairingCodeStatus) Execute(_ context.Context, _ string) (*iamDomain.ScannerPairingCode, error) {
	return s.out, s.err
}

func newTestRouter(h *scanning.Handler) *gin.Engine {
	r := gin.New()
	// Inject claims the way middleware.JWT would, so handler tests exercise
	// the real "read claims from gin context" path without needing a real token.
	r.Use(func(c *gin.Context) {
		c.Set("jwt_claims", &middleware.Claims{UserID: "user-1", TenantID: "tenant-1", Role: middleware.RoleCustomerOwner})
		c.Next()
	})
	genscanning.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(rg *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, &bytes.Buffer{})
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func TestGeneratePairingCodeHandler_usesCallersClaims(t *testing.T) {
	generate := &stubGeneratePairingCode{}
	h := scanning.NewHandler(generate, &stubGetPairingCodeStatus{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/iam/scanner-pairing")

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "user-1", generate.gotInput.UserID)
	assert.Equal(t, "customer_owner", generate.gotInput.Role)
	assert.Contains(t, w.Body.String(), "123456")
}

func TestGetPairingCodeStatusHandler_returnsStatus(t *testing.T) {
	status := &stubGetPairingCodeStatus{out: &iamDomain.ScannerPairingCode{ID: "pairing-1", Status: iamDomain.ScannerPairingCodeStatusClaimed}}
	h := scanning.NewHandler(&stubGeneratePairingCode{}, status)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/iam/scanner-pairing/pairing-1")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "claimed")
}

func TestGetPairingCodeStatusHandler_notFound(t *testing.T) {
	status := &stubGetPairingCodeStatus{err: iamDomain.ErrPairingCodeNotFound}
	h := scanning.NewHandler(&stubGeneratePairingCode{}, status)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/iam/scanner-pairing/missing")

	require.Equal(t, http.StatusNotFound, w.Code)
}
