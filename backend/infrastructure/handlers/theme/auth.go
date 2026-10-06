package theme

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func bearer(c *gin.Context) string {
	const prefix = "Bearer "
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimPrefix(h, prefix)
}

// editSession authenticates the editor's bearer token and scopes the
// request context to the session's tenant and user.
func (h *Handler) editSession(c *gin.Context) (*theme.Session, bool) {
	token := bearer(c)
	if token == "" {
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
		return nil, false
	}
	s, err := h.uc.ResolveSession.Execute(c.Request.Context(), token, theme.PurposeEdit)
	if err != nil {
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired edit session")
		return nil, false
	}
	// The merchant who opened the session is the one acting -- their user
	// and role go into the context so the events editor changes raise
	// name them, not the system.
	ctx := tenant.WithTenantID(c.Request.Context(), s.TenantID)
	ctx = tenant.WithUserID(ctx, s.UserID)
	ctx = tenant.WithRole(ctx, tenant.Role(s.Role))
	c.Request = c.Request.WithContext(ctx)
	return s, true
}

// storefront is the calling channel plus, when X-Theme-Preview is sent, the
// preview session it carries.
type storefront struct {
	channelID string
	preview   *theme.Session
}

// storefrontCaller authenticates the channel storefront key (and an
// optional preview token of the same tenant) and scopes the request
// context to that tenant.
func (h *Handler) storefrontCaller(c *gin.Context, previewToken string) (storefront, bool) {
	key := bearer(c)
	if key == "" {
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
		return storefront{}, false
	}
	resolved, err := h.uc.ResolveStorefrontKey.Execute(c.Request.Context(), key)
	if err != nil || resolved.SalesChannelID == nil {
		middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid storefront key")
		return storefront{}, false
	}
	out := storefront{channelID: *resolved.SalesChannelID}
	if previewToken != "" {
		s, err := h.uc.ResolveSession.Execute(c.Request.Context(), previewToken, theme.PurposePreview)
		if err != nil || s.TenantID != resolved.TenantID {
			middleware.Fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired preview session")
			return storefront{}, false
		}
		out.preview = s
	}
	c.Request = c.Request.WithContext(tenant.WithTenantID(c.Request.Context(), resolved.TenantID))
	return out, true
}
