package storefront

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genstorefront "github.com/adirkuhn/commerceos/api/storefront"
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func (h *Handler) RequestMagicLink(c *gin.Context) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.RequestMagicLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	err := h.requestMagicLink.Execute(ctx, appstorefront.RequestMagicLinkInput{
		SalesChannelID: *resolved.SalesChannelID, Email: string(req.Email),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) VerifySession(c *gin.Context) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.VerifySessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	session, err := h.verifyMagicLink.Execute(ctx, *resolved.SalesChannelID, req.Token)
	if err := mapError(c, err); err != nil {
		return
	}
	account, err := h.getAccount.Execute(ctx, *resolved.SalesChannelID, session.Token)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genstorefront.Session{
		SessionToken: session.Token,
		Account:      genstorefront.Account{Email: account.Email, FirstName: account.FirstName, LastName: account.LastName},
	})
}

func (h *Handler) GetAccount(c *gin.Context, token string) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	account, err := h.getAccount.Execute(ctx, *resolved.SalesChannelID, token)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genstorefront.Account{Email: account.Email, FirstName: account.FirstName, LastName: account.LastName})
}

func (h *Handler) ListAccountOrders(c *gin.Context, token string) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	orders, err := h.listAccountOrders.Execute(ctx, *resolved.SalesChannelID, token)
	if err := mapError(c, err); err != nil {
		return
	}
	items := make([]genstorefront.AccountOrder, len(orders))
	for i, o := range orders {
		items[i] = genstorefront.AccountOrder{
			OrderNumber: o.OrderNumber, Status: string(o.Status), Fulfilled: o.Fulfilled,
			TotalMinorUnits: o.TotalMinorUnits, CurrencyCode: o.CurrencyCode, PlacedAt: o.PlacedAt,
		}
	}
	middleware.OK(c, items)
}

func (h *Handler) Logout(c *gin.Context, token string) {
	if _, ok := h.authenticateChannel(c); !ok {
		return
	}
	if err := mapError(c, h.logout.Execute(c.Request.Context(), token)); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) LookupStorefrontOrder(c *gin.Context, params genstorefront.LookupStorefrontOrderParams) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	status, err := h.lookupOrder.Execute(ctx, *resolved.SalesChannelID, params.OrderNumber, string(params.Email))
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genstorefront.StorefrontOrderStatus{
		OrderNumber: status.OrderNumber, Status: string(status.Status), Fulfilled: status.Fulfilled,
		TotalMinorUnits: status.TotalMinorUnits, CurrencyCode: status.CurrencyCode,
	})
}
