package scannerauth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genscannerauth "github.com/adirkuhn/commerceos/api/scannerauth"
	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/infrastructure/handlers/scannerauth"
)

func init() { gin.SetMode(gin.TestMode) }

type stubClaimPairingCode struct {
	out      string
	err      error
	gotInput string
}

func (s *stubClaimPairingCode) Execute(_ context.Context, code string) (string, error) {
	s.gotInput = code
	return s.out, s.err
}

func newTestRouter(h *scannerauth.Handler) *gin.Engine {
	r := gin.New()
	genscannerauth.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(rg *gin.Engine, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/iam/scanner-pairing/claim", &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func TestClaimPairingCodeHandler_success(t *testing.T) {
	claim := &stubClaimPairingCode{out: "signed-jwt-token"}
	h := scannerauth.NewHandler(claim)
	w := doRequest(newTestRouter(h), map[string]any{"code": "123456"})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "123456", claim.gotInput)
	assert.Contains(t, w.Body.String(), "signed-jwt-token")
}

func TestClaimPairingCodeHandler_invalidCodeReturns422(t *testing.T) {
	claim := &stubClaimPairingCode{err: iamDomain.ErrPairingCodeInvalid}
	h := scannerauth.NewHandler(claim)
	w := doRequest(newTestRouter(h), map[string]any{"code": "000000"})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
