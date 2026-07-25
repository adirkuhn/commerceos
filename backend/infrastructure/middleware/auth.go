package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/adirkuhn/commerceos/internal/tenant"
)

type Role string

const (
	RoleSuperAdmin    Role = "super_admin"
	RoleCustomerOwner Role = "customer_owner"
	RoleCustomerAdmin Role = "customer_admin"
	RoleCustomerUser  Role = "customer_user"
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
		ctx := c.Request.Context()
		if claims.Role == RoleSuperAdmin {
			ctx = tenant.WithSuperAdmin(ctx)
		} else {
			ctx = tenant.WithTenantID(ctx, claims.TenantID)
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

func ClaimsFromGin(c *gin.Context) *Claims {
	v, exists := c.Get(claimsGinKey)
	if !exists {
		return nil
	}
	claims, _ := v.(*Claims)
	return claims
}
