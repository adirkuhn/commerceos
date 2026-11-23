// Package assistant wires the admin AI assistant API (see api/assistant's
// doc comment) to its use cases. Thin: parse, call, map.
package assistant

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/gorm"

	genassistant "github.com/adirkuhn/commerceos/api/assistant"
	appassistant "github.com/adirkuhn/commerceos/application/assistant"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type createUC interface {
	Execute(ctx context.Context, text string) (*assistant.Request, error)
}

type listUC interface {
	Execute(ctx context.Context, f assistant.ListFilter) (*appassistant.RequestPage, error)
}

type getUC interface {
	Execute(ctx context.Context, id string) (*appassistant.RequestDetail, error)
}

type replyUC interface {
	Execute(ctx context.Context, id, text string) (*assistant.Request, error)
}

type cancelUC interface {
	Execute(ctx context.Context, id string) (*assistant.Request, error)
}

type Handler struct {
	create createUC
	list   listUC
	get    getUC
	reply  replyUC
	cancel cancelUC
}

func NewHandler(create createUC, list listUC, get getUC, reply replyUC, cancel cancelUC) *Handler {
	return &Handler{create: create, list: list, get: get, reply: reply, cancel: cancel}
}

var _ genassistant.ServerInterface = (*Handler)(nil)

// Register wires the use cases on the real repositories and mounts them.
func Register(rg gin.IRouter, db *gorm.DB, outbox eventrecord.Enqueuer) error {
	requests, messages := repositories.NewAssistantRequestRepo(db), repositories.NewAssistantMessageRepo(db)
	return Mount(rg, NewHandler(
		appassistant.NewCreateRequestUseCase(db, requests, messages, outbox, time.Now),
		appassistant.NewListRequestsUseCase(db, requests),
		appassistant.NewGetRequestUseCase(db, requests, messages),
		appassistant.NewReplyToRequestUseCase(db, requests, messages, outbox, time.Now),
		appassistant.NewCancelRequestUseCase(db, requests, outbox, time.Now),
	))
}

// Mount guards every route to admin roles, validates each request against
// the OpenAPI contract (violations are 422 VALIDATION_ERROR in the standard
// error envelope), then registers h.
func Mount(rg gin.IRouter, h *Handler) error {
	spec, err := genassistant.GetSpec()
	if err != nil {
		return fmt.Errorf("assistant: load openapi spec: %w", err)
	}
	g := rg.Group("")
	g.Use(middleware.RequireRole(middleware.RoleCustomerOwner, middleware.RoleCustomerAdmin, middleware.RoleSuperAdmin))
	g.Use(ginmiddleware.OapiRequestValidatorWithOptions(spec, &ginmiddleware.Options{
		SilenceServersWarning: true,
		ErrorHandler: func(c *gin.Context, message string, status int) {
			if status != http.StatusNotFound {
				status = http.StatusUnprocessableEntity
			}
			middleware.Fail(c, status, "VALIDATION_ERROR", message)
		},
	}))
	genassistant.RegisterHandlersWithOptions(g, h, genassistant.GinServerOptions{
		ErrorHandler: func(c *gin.Context, err error, _ int) {
			middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		},
	})
	return nil
}

func (h *Handler) ListAssistantRequests(c *gin.Context, params genassistant.ListAssistantRequestsParams) {
	var f assistant.ListFilter
	if params.Status != nil {
		f.Status = assistant.Status(*params.Status)
	}
	if params.Limit != nil {
		f.Limit = *params.Limit
	}
	if params.Cursor != nil {
		f.Cursor = params.Cursor.String()
	}
	page, err := h.list.Execute(c.Request.Context(), f)
	if mapError(c, err) != nil {
		return
	}
	out := genassistant.AssistantRequestList{Requests: make([]genassistant.AssistantRequest, len(page.Requests))}
	for i, r := range page.Requests {
		out.Requests[i] = toGenRequest(r)
	}
	if page.NextCursor != "" {
		out.NextCursor = &page.NextCursor
	}
	c.JSON(http.StatusOK, genassistant.AssistantRequestListResponse{Data: out})
}

func (h *Handler) CreateAssistantRequest(c *gin.Context) {
	var req genassistant.AssistantTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	r, err := h.create.Execute(c.Request.Context(), req.Text)
	if mapError(c, err) != nil {
		return
	}
	c.JSON(http.StatusCreated, genassistant.AssistantRequestResponse{Data: toGenRequest(*r)})
}

func (h *Handler) GetAssistantRequest(c *gin.Context, id genassistant.Id) {
	d, err := h.get.Execute(c.Request.Context(), id.String())
	if mapError(c, err) != nil {
		return
	}
	out := genassistant.AssistantRequestDetail{Request: toGenRequest(d.Request), Messages: make([]genassistant.AssistantMessage, len(d.Messages))}
	for i, m := range d.Messages {
		out.Messages[i] = toGenMessage(m)
	}
	c.JSON(http.StatusOK, genassistant.AssistantRequestDetailResponse{Data: out})
}

func (h *Handler) ReplyToAssistantRequest(c *gin.Context, id genassistant.Id) {
	var req genassistant.AssistantTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	r, err := h.reply.Execute(c.Request.Context(), id.String(), req.Text)
	if mapError(c, err) != nil {
		return
	}
	c.JSON(http.StatusOK, genassistant.AssistantRequestResponse{Data: toGenRequest(*r)})
}

func (h *Handler) CancelAssistantRequest(c *gin.Context, id genassistant.Id) {
	r, err := h.cancel.Execute(c.Request.Context(), id.String())
	if mapError(c, err) != nil {
		return
	}
	c.JSON(http.StatusOK, genassistant.AssistantRequestResponse{Data: toGenRequest(*r)})
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, assistant.ErrRequestNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, assistant.ErrInvalidTransition), errors.Is(err, assistant.ErrNotCancellable):
		middleware.FailErr(c, http.StatusConflict, "CONFLICT", err)
	case errors.Is(err, assistant.ErrTextRequired), errors.Is(err, assistant.ErrTextTooLong),
		errors.Is(err, appassistant.ErrInvalidStatusFilter), errors.Is(err, appassistant.ErrNoIdentity):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	default:
		log.Printf("assistant: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}

func toGenRequest(r assistant.Request) genassistant.AssistantRequest {
	out := genassistant.AssistantRequest{
		Id: parseUUID(r.ID), Title: r.Title, Status: genassistant.AssistantRequestStatus(r.Status), Attempts: r.Attempts,
		NextAttemptAt: r.NextAttemptAt, UserId: r.UserID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.WaitReason != "" {
		reason := genassistant.AssistantWaitReason(r.WaitReason)
		out.WaitReason = &reason
	}
	if r.LastError != "" {
		out.LastError = &r.LastError
	}
	return out
}

func toGenMessage(m assistant.Message) genassistant.AssistantMessage {
	out := genassistant.AssistantMessage{
		Seq: m.Seq, Role: genassistant.AssistantMessageRole(m.Role), Content: m.Content, Question: m.Question, CreatedAt: m.CreatedAt,
	}
	if len(m.ToolCalls) > 0 {
		calls := make([]genassistant.AssistantToolCall, len(m.ToolCalls))
		for i, tc := range m.ToolCalls {
			calls[i] = genassistant.AssistantToolCall{Id: tc.ID, Name: tc.Name, Arguments: tc.Arguments}
		}
		out.ToolCalls = &calls
	}
	if m.ToolName != "" {
		out.ToolName = &m.ToolName
	}
	if m.Notice != "" {
		notice := genassistant.AssistantNotice(m.Notice)
		out.Notice = &notice
	}
	return out
}

func parseUUID(id string) openapi_types.UUID {
	var u openapi_types.UUID
	_ = u.UnmarshalText([]byte(id)) // ids come from the database's uuid column
	return u
}
