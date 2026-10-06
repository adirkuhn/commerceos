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
	"github.com/adirkuhn/commerceos/internal/tenant"
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

func TestJWT_superAdminWithTenantID_stillSetsTenantContext(t *testing.T) {
	// lm2 always resolves a company on login, even for super_admin — the
	// tenant must still land in context (for code that needs a concrete
	// tenant), alongside the bypass flag (for RLS).
	claims := validClaims()
	claims.Role = middleware.RoleSuperAdmin
	claims.TenantID = "tenant-456"
	token := makeToken(t, claims)

	r := gin.New()
	r.Use(middleware.JWT([]byte(testSecret)))
	r.GET("/test", func(c *gin.Context) {
		ctx := c.Request.Context()
		c.JSON(http.StatusOK, gin.H{
			"tenant_id":      tenant.IDFromContext(ctx),
			"is_super_admin": tenant.IsSuperAdmin(ctx),
		})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"tenant_id":"tenant-456"`)
	assert.Contains(t, w.Body.String(), `"is_super_admin":true`)
}

func writeGuardRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.JWT([]byte(testSecret)))
	r.Use(middleware.RequireRoleForWrites(middleware.RoleCustomerOwner, middleware.RoleCustomerAdmin, middleware.RoleSuperAdmin))
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	r.GET("/settings", ok)
	r.PUT("/settings", ok)
	r.PATCH("/settings", ok)
	r.POST("/settings", ok)
	r.DELETE("/settings", ok)
	return r
}

func TestRequireRoleForWrites_readOnlyRoleCanRead(t *testing.T) {
	claims := validClaims()
	claims.Role = middleware.RoleCustomerUser
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	req.Header.Set("Authorization", "Bearer "+makeToken(t, claims))

	writeGuardRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRoleForWrites_readOnlyRoleCannotWrite(t *testing.T) {
	claims := validClaims()
	claims.Role = middleware.RoleCustomerUser
	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/settings", nil)
		req.Header.Set("Authorization", "Bearer "+makeToken(t, claims))

		writeGuardRouter().ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code, method)
	}
}

func TestRequireRoleForWrites_allowedRoleCanWrite(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/settings", nil)
	req.Header.Set("Authorization", "Bearer "+makeToken(t, validClaims()))

	writeGuardRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestJWT_propagatesRoleToRequestContext(t *testing.T) {
	r := gin.New()
	r.Use(middleware.JWT([]byte(testSecret)))
	r.GET("/role", func(c *gin.Context) {
		c.String(http.StatusOK, string(tenant.RoleFromContext(c.Request.Context())))
	})
	claims := validClaims()
	claims.Role = middleware.RoleCustomerAdmin
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/role", nil)
	req.Header.Set("Authorization", "Bearer "+makeToken(t, claims))

	r.ServeHTTP(w, req)

	assert.Equal(t, "customer_admin", w.Body.String())
}

func TestJWT_propagatesUserIDToRequestContext(t *testing.T) {
	r := gin.New()
	r.Use(middleware.JWT([]byte(testSecret)))
	r.GET("/me", func(c *gin.Context) {
		c.String(http.StatusOK, tenant.UserIDFromContext(c.Request.Context()))
	})
	claims := validClaims()
	claims.UserID = "user-42"
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+makeToken(t, claims))

	r.ServeHTTP(w, req)

	assert.Equal(t, "user-42", w.Body.String())
}
