package storefront

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	genstorefront "github.com/adirkuhn/commerceos/api/storefront"
	appnewsletter "github.com/adirkuhn/commerceos/application/newsletter"
	"github.com/adirkuhn/commerceos/domain/newsletter"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type NewsletterUseCases struct {
	Subscribe interface {
		Execute(ctx context.Context, in appnewsletter.SubscribeInput) error
	}
	Confirm interface {
		Execute(ctx context.Context, token string) error
	}
	Unsubscribe interface {
		Execute(ctx context.Context, token string) error
	}
}

func newsletterStatus(c *gin.Context, code int, status genstorefront.NewsletterStatusStatus) {
	c.JSON(code, genstorefront.NewsletterStatusResponse{Data: &genstorefront.NewsletterStatus{Status: status}})
}

func newsletterFail(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, newsletter.ErrInvalidEmail), errors.Is(err, newsletter.ErrConsentRequired), errors.Is(err, newsletter.ErrInvalidToken):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return true
	}
	_ = mapError(c, err)
	return true
}

func (h *Handler) SubscribeToNewsletter(c *gin.Context) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.NewsletterSubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	err := h.newsletter.Subscribe.Execute(tenant.WithTenantID(c.Request.Context(), resolved.TenantID), appnewsletter.SubscribeInput{
		SalesChannelID: *resolved.SalesChannelID, Email: req.Email, Consent: req.Consent,
	})
	if newsletterFail(c, err) {
		return
	}
	newsletterStatus(c, http.StatusAccepted, genstorefront.CheckYourEmail)
}

func (h *Handler) ConfirmNewsletter(c *gin.Context) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.NewsletterTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	if newsletterFail(c, h.newsletter.Confirm.Execute(tenant.WithTenantID(c.Request.Context(), resolved.TenantID), req.Token)) {
		return
	}
	newsletterStatus(c, http.StatusOK, genstorefront.Subscribed)
}

func (h *Handler) UnsubscribeFromNewsletter(c *gin.Context) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	var req genstorefront.NewsletterTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	if newsletterFail(c, h.newsletter.Unsubscribe.Execute(tenant.WithTenantID(c.Request.Context(), resolved.TenantID), req.Token)) {
		return
	}
	newsletterStatus(c, http.StatusOK, genstorefront.Unsubscribed)
}
