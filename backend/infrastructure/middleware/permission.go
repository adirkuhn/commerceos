package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// RequirePermission checks that the caller has the given rule via group membership.
// super_admin bypasses all permission checks.
// Must be placed after middleware.JWT.
func RequirePermission(db *gorm.DB, rule string) gin.HandlerFunc {
	gpRepo := repositories.NewGroupPermissionRepo(db)

	return func(c *gin.Context) {
		claims := ClaimsFromGin(c)
		if claims == nil {
			Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "no claims in context")
			return
		}
		if claims.Role == RoleSuperAdmin {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		var allowed bool
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var e error
			allowed, e = gpRepo.HasPermission(ctx, claims.UserID, iamDomain.Rule(rule))
			return e
		})
		if err != nil {
			Fail(c, http.StatusInternalServerError, "INTERNAL", "permission check failed")
			return
		}
		if !allowed {
			Fail(c, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
			return
		}
		c.Next()
	}
}
