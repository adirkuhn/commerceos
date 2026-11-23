// Package assistant holds the admin AI assistant's use cases: queueing a
// request, listing/reading/replying/cancelling it, and the worker side
// that runs the model over the queue (process.go, process_due.go).
package assistant

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/domain/errs"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

const (
	DefaultListLimit = 25
	MaxListLimit     = 100
)

var (
	ErrNoIdentity          = errs.New("ASSISTANT_NO_IDENTITY", "the assistant needs a signed-in user of a tenant")
	ErrInvalidStatusFilter = errs.New("ASSISTANT_INVALID_STATUS_FILTER", "unknown assistant request status")
)

// CreateRequestUseCase queues what the user wrote; the worker picks it up.
type CreateRequestUseCase struct {
	db       *gorm.DB
	requests assistant.RequestRepository
	messages assistant.MessageRepository
	outbox   eventrecord.Enqueuer
	now      func() time.Time
}

func NewCreateRequestUseCase(db *gorm.DB, requests assistant.RequestRepository, messages assistant.MessageRepository, outbox eventrecord.Enqueuer, now func() time.Time) *CreateRequestUseCase {
	return &CreateRequestUseCase{db: db, requests: requests, messages: messages, outbox: outbox, now: now}
}

func (uc *CreateRequestUseCase) Execute(ctx context.Context, text string) (*assistant.Request, error) {
	tenantID, userID := tenant.IDFromContext(ctx), tenant.UserIDFromContext(ctx)
	if tenantID == "" || userID == "" {
		return nil, fmt.Errorf("createAssistantRequest: %w", ErrNoIdentity)
	}
	at := uc.now()
	r, err := assistant.NewRequest(tenantID, userID, string(tenant.RoleFromContext(ctx)), text, at)
	if err != nil {
		return nil, fmt.Errorf("createAssistantRequest: %w", err)
	}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.requests.Create(ctx, r); err != nil {
			return err
		}
		if err := uc.messages.Append(ctx, &assistant.Message{
			RequestID: r.ID, TenantID: tenantID, Role: assistant.RoleUser, Content: text, CreatedAt: at,
		}); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.AssistantRequestCreated, r.ID)
	})
	if err != nil {
		return nil, fmt.Errorf("createAssistantRequest: %w", err)
	}
	return r, nil
}

// ListRequestsUseCase pages through the tenant's requests, newest first.
type ListRequestsUseCase struct {
	db       *gorm.DB
	requests assistant.RequestRepository
}

type RequestPage struct {
	Requests   []assistant.Request
	NextCursor string
}

func NewListRequestsUseCase(db *gorm.DB, requests assistant.RequestRepository) *ListRequestsUseCase {
	return &ListRequestsUseCase{db: db, requests: requests}
}

func (uc *ListRequestsUseCase) Execute(ctx context.Context, f assistant.ListFilter) (*RequestPage, error) {
	if f.Status != "" && !f.Status.Valid() {
		return nil, fmt.Errorf("listAssistantRequests: %w", ErrInvalidStatusFilter)
	}
	limit := f.Limit
	if limit <= 0 {
		limit = DefaultListLimit
	}
	limit = min(limit, MaxListLimit)
	f.Limit = limit + 1

	var rows []assistant.Request
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		rows, err = uc.requests.List(ctx, f)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listAssistantRequests: %w", err)
	}
	page := &RequestPage{Requests: rows}
	if len(rows) > limit {
		page.Requests = rows[:limit]
		page.NextCursor = rows[limit-1].ID
	}
	return page, nil
}

// GetRequestUseCase returns a request with its conversation. Raw tool
// results stay server-side: they're the model's working data, can be
// large, and the UI only shows which tool ran.
type GetRequestUseCase struct {
	db       *gorm.DB
	requests assistant.RequestRepository
	messages assistant.MessageRepository
}

type RequestDetail struct {
	Request  assistant.Request
	Messages []assistant.Message
}

func NewGetRequestUseCase(db *gorm.DB, requests assistant.RequestRepository, messages assistant.MessageRepository) *GetRequestUseCase {
	return &GetRequestUseCase{db: db, requests: requests, messages: messages}
}

func (uc *GetRequestUseCase) Execute(ctx context.Context, id string) (*RequestDetail, error) {
	var d RequestDetail
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		r, err := uc.requests.Get(ctx, id)
		if err != nil {
			return err
		}
		msgs, err := uc.messages.ListByRequest(ctx, id)
		if err != nil {
			return err
		}
		d.Request = *r
		d.Messages = msgs
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getAssistantRequest: %w", err)
	}
	for i := range d.Messages {
		if d.Messages[i].Role == assistant.RoleTool {
			d.Messages[i].Content = ""
		}
	}
	return &d, nil
}

// ReplyToRequestUseCase adds the user's answer or follow-up and queues the
// request again.
type ReplyToRequestUseCase struct {
	db       *gorm.DB
	requests assistant.RequestRepository
	messages assistant.MessageRepository
	outbox   eventrecord.Enqueuer
	now      func() time.Time
}

func NewReplyToRequestUseCase(db *gorm.DB, requests assistant.RequestRepository, messages assistant.MessageRepository, outbox eventrecord.Enqueuer, now func() time.Time) *ReplyToRequestUseCase {
	return &ReplyToRequestUseCase{db: db, requests: requests, messages: messages, outbox: outbox, now: now}
}

func (uc *ReplyToRequestUseCase) Execute(ctx context.Context, id, text string) (*assistant.Request, error) {
	if err := assistant.ValidateText(text); err != nil {
		return nil, fmt.Errorf("replyToAssistantRequest: %w", err)
	}
	var out *assistant.Request
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		r, err := uc.requests.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}
		at := uc.now()
		if err := r.Reply(at); err != nil {
			return err
		}
		if err := uc.requests.Update(ctx, r); err != nil {
			return err
		}
		if err := uc.messages.Append(ctx, &assistant.Message{
			RequestID: r.ID, TenantID: r.TenantID, Role: assistant.RoleUser, Content: text, CreatedAt: at,
		}); err != nil {
			return err
		}
		out = r
		return eventrecord.Record(ctx, uc.outbox, events.AssistantRequestReplied, r.ID)
	})
	if err != nil {
		return nil, fmt.Errorf("replyToAssistantRequest: %w", err)
	}
	return out, nil
}

// CancelRequestUseCase stops a request that hasn't finished. A worker
// mid-attempt notices on its next write and drops its result.
type CancelRequestUseCase struct {
	db       *gorm.DB
	requests assistant.RequestRepository
	outbox   eventrecord.Enqueuer
	now      func() time.Time
}

func NewCancelRequestUseCase(db *gorm.DB, requests assistant.RequestRepository, outbox eventrecord.Enqueuer, now func() time.Time) *CancelRequestUseCase {
	return &CancelRequestUseCase{db: db, requests: requests, outbox: outbox, now: now}
}

func (uc *CancelRequestUseCase) Execute(ctx context.Context, id string) (*assistant.Request, error) {
	var out *assistant.Request
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		r, err := uc.requests.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}
		previous := r.Status
		if err := r.Cancel(uc.now()); err != nil {
			return err
		}
		if err := uc.requests.Update(ctx, r); err != nil {
			return err
		}
		out = r
		return eventrecord.Record(ctx, uc.outbox, events.AssistantRequestCancelled, r.ID,
			eventrecord.Data{"previous_status": string(previous)})
	})
	if err != nil {
		return nil, fmt.Errorf("cancelAssistantRequest: %w", err)
	}
	return out, nil
}
