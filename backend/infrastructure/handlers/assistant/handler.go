// Package assistant wires the admin AI assistant API (see api/assistant's
// doc comment) to its use cases. Thin: parse, call, map.
package assistant

import (
	"context"
	"encoding/json"
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
	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/domain/errs"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/infrastructure/assistanttools"
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

type confirmUC interface {
	Execute(ctx context.Context, requestID, proposalID string, choices appassistant.ConfirmChoices) (*appassistant.RequestDetail, error)
}

type rejectUC interface {
	Execute(ctx context.Context, requestID, proposalID string) (*appassistant.RequestDetail, error)
}

type retryUC interface {
	Execute(ctx context.Context, id string) (*assistant.Request, error)
}

// UseCases are the handler's ports, one per operation.
type UseCases struct {
	Retry   retryUC
	Create  createUC
	List    listUC
	Get     getUC
	Reply   replyUC
	Cancel  cancelUC
	Confirm confirmUC
	Reject  rejectUC
}

type Handler struct{ uc UseCases }

func NewHandler(uc UseCases) *Handler { return &Handler{uc: uc} }

var _ genassistant.ServerInterface = (*Handler)(nil)

// Register wires the use cases on the real repositories and proposal
// actions, and mounts them.
func Register(rg gin.IRouter, db *gorm.DB, outboxRepo outbox.Repository) error {
	requests, messages := repositories.NewAssistantRequestRepo(db), repositories.NewAssistantMessageRepo(db)
	proposals := repositories.NewAssistantProposalRepo(db)
	decide := appassistant.DecideDeps{
		Requests: requests, Messages: messages, Proposals: proposals,
		Actions: assistanttools.ProposalActions(db, outboxRepo), Outbox: outboxRepo, Now: time.Now,
	}
	return Mount(rg, NewHandler(UseCases{
		Create:  appassistant.NewCreateRequestUseCase(db, requests, messages, outboxRepo, time.Now),
		List:    appassistant.NewListRequestsUseCase(db, requests),
		Get:     appassistant.NewGetRequestUseCase(db, requests, messages, proposals),
		Reply:   appassistant.NewReplyToRequestUseCase(db, requests, messages, proposals, outboxRepo, time.Now),
		Cancel:  appassistant.NewCancelRequestUseCase(db, requests, proposals, outboxRepo, time.Now),
		Confirm: appassistant.NewConfirmProposalUseCase(db, decide),
		Reject:  appassistant.NewRejectProposalUseCase(db, decide),
		Retry:   appassistant.NewRetryRequestUseCase(db, requests, outboxRepo, time.Now),
	}))
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
	page, err := h.uc.List.Execute(c.Request.Context(), f)
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
	r, err := h.uc.Create.Execute(c.Request.Context(), req.Text)
	if mapError(c, err) != nil {
		return
	}
	c.JSON(http.StatusCreated, genassistant.AssistantRequestResponse{Data: toGenRequest(*r)})
}

func (h *Handler) GetAssistantRequest(c *gin.Context, id genassistant.Id) {
	d, err := h.uc.Get.Execute(c.Request.Context(), id.String())
	if mapError(c, err) != nil {
		return
	}
	writeDetail(c, d)
}

func (h *Handler) ConfirmAssistantProposal(c *gin.Context, id genassistant.Id, proposalID genassistant.ProposalId) {
	var req genassistant.ConfirmProposalRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
			return
		}
	}
	var choices appassistant.ConfirmChoices
	if req.WarehouseId != nil {
		choices.WarehouseID = req.WarehouseId.String()
	}
	d, err := h.uc.Confirm.Execute(c.Request.Context(), id.String(), proposalID.String(), choices)
	if mapError(c, err) != nil {
		return
	}
	writeDetail(c, d)
}

func (h *Handler) RejectAssistantProposal(c *gin.Context, id genassistant.Id, proposalID genassistant.ProposalId) {
	d, err := h.uc.Reject.Execute(c.Request.Context(), id.String(), proposalID.String())
	if mapError(c, err) != nil {
		return
	}
	writeDetail(c, d)
}

func writeDetail(c *gin.Context, d *appassistant.RequestDetail) {
	out := genassistant.AssistantRequestDetail{
		Request:   toGenRequest(d.Request),
		Messages:  make([]genassistant.AssistantMessage, len(d.Messages)),
		Proposals: make([]genassistant.AssistantProposal, 0, len(d.Proposals)),
	}
	for i, m := range d.Messages {
		out.Messages[i] = toGenMessage(m)
	}
	for _, p := range d.Proposals {
		gp, err := toGenProposal(p)
		if err != nil {
			log.Printf("assistant: proposal %s preview: %v", p.ID, err)
			middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
			return
		}
		out.Proposals = append(out.Proposals, gp)
	}
	c.JSON(http.StatusOK, genassistant.AssistantRequestDetailResponse{Data: out})
}

func (h *Handler) ReplyToAssistantRequest(c *gin.Context, id genassistant.Id) {
	var req genassistant.AssistantTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	r, err := h.uc.Reply.Execute(c.Request.Context(), id.String(), req.Text)
	if mapError(c, err) != nil {
		return
	}
	c.JSON(http.StatusOK, genassistant.AssistantRequestResponse{Data: toGenRequest(*r)})
}

func (h *Handler) RetryAssistantRequest(c *gin.Context, id genassistant.Id) {
	r, err := h.uc.Retry.Execute(c.Request.Context(), id.String())
	if mapError(c, err) != nil {
		return
	}
	c.JSON(http.StatusOK, genassistant.AssistantRequestResponse{Data: toGenRequest(*r)})
}

func (h *Handler) CancelAssistantRequest(c *gin.Context, id genassistant.Id) {
	r, err := h.uc.Cancel.Execute(c.Request.Context(), id.String())
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
	case errors.Is(err, assistant.ErrRequestNotFound), errors.Is(err, assistant.ErrProposalNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, assistant.ErrInvalidTransition), errors.Is(err, assistant.ErrNotCancellable),
		errors.Is(err, assistant.ErrProposalNotPending):
		middleware.FailErr(c, http.StatusConflict, "CONFLICT", err)
	case isCoded(err):
		// Validation from this module, or the reason a confirmed action
		// was refused (a quote's or an order's own coded domain error).
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	default:
		log.Printf("assistant: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}

func isCoded(err error) bool {
	_, _, ok := errs.CodeOf(err)
	return ok
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
	if m.ProposalID != "" {
		id := parseUUID(m.ProposalID)
		out.ProposalId = &id
	}
	return out
}

// toGenProposal decodes the action-owned preview into the contract's typed
// preview for its kind.
func toGenProposal(p assistant.Proposal) (genassistant.AssistantProposal, error) {
	out := genassistant.AssistantProposal{
		Id: parseUUID(p.ID), Kind: genassistant.AssistantProposalKind(p.Kind), Status: genassistant.AssistantProposalStatus(p.Status),
		CreatedAt: p.CreatedAt, DecidedAt: p.DecidedAt,
		LastError: optional(p.LastError), ResultId: optional(p.ResultID), ResultRef: optional(p.ResultRef), DecidedBy: optional(p.DecidedBy),
	}
	switch p.Kind {
	case assistant.KindCreateQuote:
		var v genassistant.QuoteProposalPreview
		if err := json.Unmarshal(p.Preview, &v); err != nil {
			return out, err
		}
		out.Quote = &v
	case assistant.KindConvertQuote:
		var v genassistant.ConversionProposalPreview
		if err := json.Unmarshal(p.Preview, &v); err != nil {
			return out, err
		}
		out.Conversion = &v
	}
	return out, nil
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseUUID(id string) openapi_types.UUID {
	var u openapi_types.UUID
	_ = u.UnmarshalText([]byte(id)) // ids come from the database's uuid column
	return u
}
