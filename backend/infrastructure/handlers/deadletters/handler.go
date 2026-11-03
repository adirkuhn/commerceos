// Package deadletters is the HTTP adapter for the super_admin dead-letter
// admin (api/deadletters/openapi.yaml, COM-14).
package deadletters

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go/jetstream"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"gorm.io/gorm"

	gendeadletters "github.com/adirkuhn/commerceos/api/deadletters"
	appdeadletters "github.com/adirkuhn/commerceos/application/deadletters"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/deadletter"
	"github.com/adirkuhn/commerceos/infrastructure/dlq"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ── Use-case interfaces (defined at point of use — ISP) ───────────────────────

type listUC interface {
	Execute(ctx context.Context, f deadletter.Filter) (deadletter.Page, error)
}
type getUC interface {
	Execute(ctx context.Context, stream deadletter.Stream, seq uint64) (deadletter.DeadLetter, error)
}
type actionUC interface {
	Execute(ctx context.Context, stream deadletter.Stream, seq uint64) error
}

type Handler struct {
	list    listUC
	get     getUC
	retry   actionUC
	discard actionUC
}

func NewHandler(list listUC, get getUC, retry, discard actionUC) *Handler {
	return &Handler{list: list, get: get, retry: retry, discard: discard}
}

var _ gendeadletters.ServerInterface = (*Handler)(nil)

// Register is the production wiring: JetStream-backed store, use cases,
// mounted on rg (which must already run middleware.JWT).
func Register(rg gin.IRouter, db *gorm.DB, js jetstream.JetStream, outbox eventrecord.Enqueuer) error {
	store := dlq.NewStore(js)
	return Mount(rg, NewHandler(
		appdeadletters.NewListDeadLettersUseCase(store),
		appdeadletters.NewGetDeadLetterUseCase(store),
		appdeadletters.NewRetryDeadLetterUseCase(db, store, store, outbox),
		appdeadletters.NewDiscardDeadLetterUseCase(db, store, outbox),
	))
}

// Mount guards every route to super_admin, validates each request against
// the OpenAPI contract (violations are 422 VALIDATION_ERROR in the standard
// error envelope), then registers h.
func Mount(rg gin.IRouter, h *Handler) error {
	spec, err := gendeadletters.GetSpec()
	if err != nil {
		return fmt.Errorf("deadletters: load openapi spec: %w", err)
	}
	g := rg.Group("")
	g.Use(middleware.RequireRole(middleware.RoleSuperAdmin))
	g.Use(ginmiddleware.OapiRequestValidatorWithOptions(spec, &ginmiddleware.Options{
		SilenceServersWarning: true,
		ErrorHandler: func(c *gin.Context, message string, status int) {
			if status != http.StatusNotFound {
				status = http.StatusUnprocessableEntity
			}
			middleware.Fail(c, status, "VALIDATION_ERROR", message)
		},
	}))
	gendeadletters.RegisterHandlersWithOptions(g, h, gendeadletters.GinServerOptions{
		ErrorHandler: func(c *gin.Context, err error, _ int) {
			middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		},
	})
	return nil
}

func (h *Handler) ListDeadLetters(c *gin.Context, params gendeadletters.ListDeadLettersParams) {
	f := deadletter.Filter{Stream: deadletter.Stream(params.Stream)}
	if params.Subject != nil {
		f.Subject = *params.Subject
	}
	if params.Consumer != nil {
		f.Consumer = *params.Consumer
	}
	if params.TenantId != nil {
		f.TenantID = *params.TenantId
	}
	if params.Cursor != nil {
		f.After = uint64(*params.Cursor) // contract: minimum 0
	}
	if params.Limit != nil {
		f.Limit = *params.Limit
	}
	page, err := h.list.Execute(c.Request.Context(), f)
	if err != nil {
		fail(c, err)
		return
	}
	items := make([]gendeadletters.DeadLetterSummary, len(page.Items))
	for i, d := range page.Items {
		items[i] = toSummary(d)
	}
	out := gendeadletters.DeadLetterPage{Items: items, Total: int64(page.Total)}
	if page.NextCursor > 0 {
		next := int64(page.NextCursor)
		out.NextCursor = &next
	}
	c.JSON(http.StatusOK, gendeadletters.DeadLetterListResponse{Data: out})
}

func (h *Handler) GetDeadLetter(c *gin.Context, stream gendeadletters.Stream, seq gendeadletters.Seq) {
	d, err := h.get.Execute(c.Request.Context(), deadletter.Stream(stream), uint64(seq))
	if err != nil {
		fail(c, err)
		return
	}
	s := toSummary(d)
	c.JSON(http.StatusOK, gendeadletters.DeadLetterResponse{Data: gendeadletters.DeadLetter{
		Stream: s.Stream, Sequence: s.Sequence, Subject: s.Subject, Consumer: s.Consumer, ConsumerKnown: s.ConsumerKnown,
		EventId: s.EventId, TenantId: s.TenantId, Error: s.Error, NumDelivered: s.NumDelivered,
		DeadLetteredAt: s.DeadLetteredAt, StoredAt: s.StoredAt, Data: string(d.Data),
	}})
}

func (h *Handler) RetryDeadLetter(c *gin.Context, stream gendeadletters.Stream, seq gendeadletters.Seq) {
	if err := h.retry.Execute(c.Request.Context(), deadletter.Stream(stream), uint64(seq)); err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gendeadletters.EmptyResponse{})
}

func (h *Handler) DiscardDeadLetter(c *gin.Context, stream gendeadletters.Stream, seq gendeadletters.Seq) {
	if err := h.discard.Execute(c.Request.Context(), deadletter.Stream(stream), uint64(seq)); err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gendeadletters.EmptyResponse{})
}

func toSummary(d deadletter.DeadLetter) gendeadletters.DeadLetterSummary {
	return gendeadletters.DeadLetterSummary{
		Stream: gendeadletters.Stream(d.Stream), Sequence: int64(d.Sequence), Subject: d.Subject,
		Consumer: d.Consumer, ConsumerKnown: d.ConsumerKnown(), EventId: d.EventID, TenantId: d.TenantID,
		Error: d.Error, NumDelivered: d.NumDelivered, DeadLetteredAt: d.DeadLetteredAt, StoredAt: d.StoredAt,
	}
}

// fail maps domain errors to statuses in one place.
func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, deadletter.ErrNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, deadletter.ErrForbidden):
		middleware.FailErr(c, http.StatusForbidden, "FORBIDDEN", err)
	case errors.Is(err, deadletter.ErrInvalidStream):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	case errors.Is(err, deadletter.ErrRetryFailed):
		middleware.FailErr(c, http.StatusBadGateway, "DEAD_LETTER_RETRY_FAILED", err)
	default:
		log.Printf("deadletters: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected error")
	}
}
