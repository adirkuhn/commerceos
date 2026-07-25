package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

const testSecret = "test-secret-key"

func init() {
	gin.SetMode(gin.TestMode)
}

func makeToken(t *testing.T, claims middleware.Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(testSecret))
	require.NoError(t, err)
	return s
}

func validClaims() middleware.Claims {
	return middleware.Claims{
		UserID:   "user-123",
		TenantID: "tenant-456",
		Role:     middleware.RoleCustomerOwner,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
}

func newRouter(secret string) *gin.Engine {
	r := gin.New()
	r.Use(middleware.JWT([]byte(secret)))
	r.GET("/test", func(c *gin.Context) {
		claims := middleware.ClaimsFromGin(c)
		c.JSON(http.StatusOK, gin.H{"user_id": claims.UserID, "role": claims.Role})
	})
	return r
}

func TestJWT_valid_token(t *testing.T) {
	token := makeToken(t, validClaims())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	newRouter(testSecret).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestJWT_missing_header(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	newRouter(testSecret).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWT_wrong_secret(t *testing.T) {
	token := makeToken(t, validClaims())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	newRouter("wrong-secret").ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWT_expired_token(t *testing.T) {
	claims := validClaims()
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))
	token := makeToken(t, claims)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	newRouter(testSecret).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWT_malformed_bearer(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Token abc123") // wrong scheme

	newRouter(testSecret).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireRole_matching(t *testing.T) {
	token := makeToken(t, validClaims())

	r := gin.New()
	r.Use(middleware.JWT([]byte(testSecret)))
	r.Use(middleware.RequireRole(middleware.RoleCustomerOwner, middleware.RoleCustomerAdmin))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRole_forbidden(t *testing.T) {
	claims := validClaims()
	claims.Role = middleware.RoleCustomerUser
	token := makeToken(t, claims)

	r := gin.New()
	r.Use(middleware.JWT([]byte(testSecret)))
	r.Use(middleware.RequireRole(middleware.RoleCustomerOwner, middleware.RoleCustomerAdmin))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestJWT_super_admin_sets_bypass(t *testing.T) {
	claims := validClaims()
	claims.Role = middleware.RoleSuperAdmin
	claims.TenantID = ""
	token := makeToken(t, claims)

	r := gin.New()
	r.Use(middleware.JWT([]byte(testSecret)))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"role": middleware.ClaimsFromGin(c).Role})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
