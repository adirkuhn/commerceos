// Package newsletter serves the admin Newsletter API (api/newsletter).
package newsletter

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	gennewsletter "github.com/adirkuhn/commerceos/api/newsletter"
	appnewsletter "github.com/adirkuhn/commerceos/application/newsletter"
	"github.com/adirkuhn/commerceos/domain/newsletter"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

type UseCases struct {
	List interface {
		Execute(ctx context.Context, in appnewsletter.ListInput) ([]newsletter.Subscriber, int64, error)
	}
	Export interface {
		Execute(ctx context.Context, out io.Writer) error
	}
	Erase interface {
		Execute(ctx context.Context, id string) error
	}
}

func NewUseCases(d appnewsletter.Deps) UseCases {
	return UseCases{
		List: appnewsletter.NewListSubscribersUseCase(d), Export: appnewsletter.NewExportSubscribersUseCase(d),
		Erase: appnewsletter.NewEraseSubscriberUseCase(d),
	}
}

type Handler struct{ uc UseCases }

func NewHandler(uc UseCases) *Handler { return &Handler{uc: uc} }

var _ gennewsletter.ServerInterface = (*Handler)(nil)

// Register mounts on a JWT group. Subscriber emails are personal data:
// every route (reading included) needs owner/admin.
func Register(rg gin.IRouter, h *Handler) {
	g := rg.Group("")
	g.Use(middleware.RequireRole(middleware.RoleCustomerOwner, middleware.RoleCustomerAdmin, middleware.RoleSuperAdmin))
	gennewsletter.RegisterHandlers(g, h)
}

func fail(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, newsletter.ErrSubscriberNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	default:
		log.Printf("newsletter: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return true
}

func (h *Handler) ListNewsletterSubscribers(c *gin.Context, p gennewsletter.ListNewsletterSubscribersParams) {
	in := appnewsletter.ListInput{Page: 1, PerPage: 50}
	if p.Page != nil {
		in.Page = *p.Page
	}
	if p.PerPage != nil {
		in.PerPage = *p.PerPage
	}
	if p.Search != nil {
		in.Search = *p.Search
	}
	if p.Status != nil {
		st := newsletter.Status(*p.Status)
		in.Status = &st
	}
	list, total, err := h.uc.List.Execute(c.Request.Context(), in)
	if fail(c, err) {
		return
	}
	data := make([]gennewsletter.NewsletterSubscriber, len(list))
	for i, s := range list {
		id, _ := uuid.Parse(s.ID)
		row := gennewsletter.NewsletterSubscriber{
			Id: id, Email: s.Email, Status: gennewsletter.NewsletterSubscriberStatus(s.Status), ConsentAt: s.ConsentAt,
			ConfirmedAt: s.ConfirmedAt, UnsubscribedAt: s.UnsubscribedAt, CreatedAt: s.CreatedAt,
		}
		if ch, err := uuid.Parse(s.SalesChannelID); err == nil {
			row.SalesChannelId = &ch
		}
		data[i] = row
	}
	t := int(total)
	c.JSON(http.StatusOK, gennewsletter.SubscriberListResponse{Data: &data, Total: &t})
}

func (h *Handler) ExportNewsletterSubscribers(c *gin.Context) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="newsletter-subscribers.csv"`)
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusOK)
	if err := h.uc.Export.Execute(c.Request.Context(), c.Writer); err != nil {
		// Headers are already sent; the truncated file is the signal.
		log.Printf("newsletter: export failed: %v", err)
	}
}

func (h *Handler) EraseNewsletterSubscriber(c *gin.Context, id uuid.UUID) {
	if fail(c, h.uc.Erase.Execute(c.Request.Context(), id.String())) {
		return
	}
	middleware.OK(c, nil)
}
