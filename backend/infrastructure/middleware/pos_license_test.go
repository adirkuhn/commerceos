package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/infrastructure/lm2license"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubLicenseVerifier struct {
	payload *lm2license.LicensePayload
	err     error
}

func (s stubLicenseVerifier) VerifyAndGetPayload(_ string) (*lm2license.LicensePayload, error) {
	return s.payload, s.err
}

func validLicensePayload() *lm2license.LicensePayload {
	return &lm2license.LicensePayload{CompanyID: "tenant-456", Status: "active"}
}

func newPOSRouter(verifier middleware.LicenseVerifier) *gin.Engine {
	r := gin.New()
	r.Use(middleware.POSLicense(verifier))
	r.GET("/test", func(c *gin.Context) {
		payload := middleware.POSLicenseFromGin(c)
		c.JSON(http.StatusOK, gin.H{"company_id": payload.CompanyID, "tenant_id": tenant.IDFromContext(c.Request.Context())})
	})
	return r
}

func TestPOSLicense_validLicense(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer license-abc")

	newPOSRouter(stubLicenseVerifier{payload: validLicensePayload()}).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"tenant_id":"tenant-456"`)
}

func TestPOSLicense_missingBearerPrefix(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "license-abc")

	newPOSRouter(stubLicenseVerifier{payload: validLicensePayload()}).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPOSLicense_noAuthorizationHeader(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	newPOSRouter(stubLicenseVerifier{payload: validLicensePayload()}).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPOSLicense_verificationFails(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer bad-license")

	newPOSRouter(stubLicenseVerifier{err: lm2license.ErrInvalidLicense}).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPOSLicense_inactiveLicenseRejectedEvenWithAValidSignature(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer suspended-license")

	newPOSRouter(stubLicenseVerifier{payload: &lm2license.LicensePayload{CompanyID: "tenant-456", Status: "suspended"}}).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPOSLicense_expiredLicenseRejected(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer expired-license")

	newPOSRouter(stubLicenseVerifier{payload: &lm2license.LicensePayload{CompanyID: "tenant-456", Status: "active", ExpiresAt: &past}}).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
