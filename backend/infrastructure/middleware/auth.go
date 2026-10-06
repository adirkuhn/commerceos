package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/adirkuhn/commerceos/internal/tenant"
)

type Role = tenant.Role

const (
	RoleSuperAdmin    = tenant.RoleSuperAdmin
	RoleCustomerOwner = tenant.RoleCustomerOwner
	RoleCustomerAdmin = tenant.RoleCustomerAdmin
	RoleCustomerUser  = tenant.RoleCustomerUser
)

type Claims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     Role   `json:"role"`
	jwt.RegisteredClaims
}

const claimsGinKey = "jwt_claims"

// JWT parses and validates the Bearer token from lm2 using HS256.
func JWT(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if !strings.HasPrefix(raw, "Bearer ") {
			Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
			return
		}

		tokenStr := strings.TrimPrefix(raw, "Bearer ")
		claims := &Claims{}

		_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return secret, nil
		}, jwt.WithExpirationRequired())

		if err != nil {
			Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
			return
		}

		c.Set(claimsGinKey, claims)

		// Propagate tenant + super_admin flag into request context for use in domain layer.
		// TenantID is set regardless of role — lm2 always resolves a company
		// on login, even for super_admin — so code that needs a concrete
		// tenant (e.g. creating a tenant-owned resource) still has one.
		// super_admin additionally gets the bypass flag, which tenant.WithTenant
		// checks first, so RLS bypass is unaffected by TenantID being present.
		ctx := c.Request.Context()
		ctx = tenant.WithTenantID(ctx, claims.TenantID)
		ctx = tenant.WithRole(ctx, claims.Role)
		ctx = tenant.WithUserID(ctx, claims.UserID)
		if claims.Role == RoleSuperAdmin {
			ctx = tenant.WithSuperAdmin(ctx)
		}
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// RequireRole aborts with 403 if the caller's role is not in the allowed list.
func RequireRole(roles ...Role) gin.HandlerFunc {
	allowed := make(map[Role]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		claims := ClaimsFromGin(c)
		if claims == nil {
			Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "no claims in context")
			return
		}
		if _, ok := allowed[claims.Role]; !ok {
			Fail(c, http.StatusForbidden, "FORBIDDEN", "insufficient role")
			return
		}
		c.Next()
	}
}

// RequireRoleForWrites lets every authenticated role read (GET/HEAD) but
// requires one of roles for anything that changes state -- for route groups
// where customer_user is read-only (CLAUDE.md: customer_user is read-only
// by default) but reads must stay open to everyone.
func RequireRoleForWrites(roles ...Role) gin.HandlerFunc {
	requireRole := RequireRole(roles...)
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Next()
			return
		}
		requireRole(c)
	}
}

func ClaimsFromGin(c *gin.Context) *Claims {
	v, exists := c.Get(claimsGinKey)
	if !exists {
		return nil
	}
	claims, _ := v.(*Claims)
	return claims
}
