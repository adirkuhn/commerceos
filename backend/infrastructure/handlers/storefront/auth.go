package storefront

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// authenticate resolves the bearer key to a tenant only -- used by routes
// (GetCartRecommendations) that don't need a specific sales channel.
func (h *Handler) authenticate(c *gin.Context) (apikeysDomain.ResolvedAPIKey, bool) {
	key := bearerToken(c.GetHeader("Authorization"))
	if key == "" {
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
		return apikeysDomain.ResolvedAPIKey{}, false
	}
	resolved, err := h.resolveAPIKey.Execute(c.Request.Context(), key)
	if err != nil {
		// Deliberately the same message as the missing-header case above --
		// never lets a caller distinguish "no key" from "wrong/revoked key".
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid api key")
		return apikeysDomain.ResolvedAPIKey{}, false
	}
	return resolved, true
}

// authenticateChannel additionally requires the resolved key to be scoped
// to a sales channel -- every cart/checkout route needs to know which
// internal storefront it's for (see domain/storefront's package doc
// comment), so a tenant-wide key (e.g. the WooCommerce plugin's) is
// rejected here rather than silently operating on an arbitrary channel.
func (h *Handler) authenticateChannel(c *gin.Context) (apikeysDomain.ResolvedAPIKey, bool) {
	resolved, ok := h.authenticate(c)
	if !ok {
		return resolved, false
	}
	if resolved.SalesChannelID == nil {
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "this key is not scoped to a sales channel")
		return apikeysDomain.ResolvedAPIKey{}, false
	}
	return resolved, true
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimPrefix(header, prefix)
}
