package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/adirkuhn/commerceos/infrastructure/lm2license"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// LicenseVerifier is satisfied by *lm2license.TokenService -- narrowed to
// the one method this middleware needs (ISP).
type LicenseVerifier interface {
	VerifyAndGetPayload(license string) (*lm2license.LicensePayload, error)
}

const posLicenseGinKey = "pos_license_payload"

// POSLicense authenticates a POS desktop request directly off its
// lm2-issued license (Bearer token) -- no separate CommerceOS credential.
// Verifies the RS256 signature via LicenseVerifier, then enforces
// status/expiry itself via IsUsable (VerifyAndGetPayload deliberately
// doesn't -- see LicensePayload.IsUsable's doc comment). CompanyID becomes
// the request's tenant, same propagation JWT does for human callers.
func POSLicense(verifier LicenseVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if !strings.HasPrefix(raw, "Bearer ") {
			Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer license")
			return
		}

		license := strings.TrimPrefix(raw, "Bearer ")
		payload, err := verifier.VerifyAndGetPayload(license)
		if err != nil || !payload.IsUsable(time.Now()) {
			Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or inactive license")
			return
		}

		c.Set(posLicenseGinKey, payload)

		ctx := tenant.WithTenantID(c.Request.Context(), payload.CompanyID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func POSLicenseFromGin(c *gin.Context) *lm2license.LicensePayload {
	v, exists := c.Get(posLicenseGinKey)
	if !exists {
		return nil
	}
	payload, _ := v.(*lm2license.LicensePayload)
	return payload
}
