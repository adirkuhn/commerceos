package storefront

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genstorefront "github.com/adirkuhn/commerceos/api/storefront"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func (h *Handler) CreateWishlist(c *gin.Context) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	wishlist, err := h.createWishlist.Execute(ctx, *resolved.SalesChannelID)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenWishlist(*wishlist))
}

func (h *Handler) GetWishlist(c *gin.Context, token string) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	wishlist, err := h.getWishlist.Execute(ctx, token)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenWishlist(*wishlist))
}

func (h *Handler) ToggleWishlistItem(c *gin.Context, token string) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.ToggleWishlistItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	wishlist, err := h.toggleWishlistItem.Execute(ctx, token, req.ProductId)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenWishlist(*wishlist))
}

func toGenWishlist(w storefrontDomain.Wishlist) genstorefront.Wishlist {
	productIDs := w.ProductIDs
	if productIDs == nil {
		productIDs = []string{}
	}
	return genstorefront.Wishlist{Token: w.Token, ProductIds: productIDs}
}
