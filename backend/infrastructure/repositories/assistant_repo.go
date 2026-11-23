package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type assistantRequestModel struct {
	ID            string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID      string `gorm:"not null;type:uuid"`
	UserID        string `gorm:"column:user_id;not null"`
	UserRole      string `gorm:"column:user_role;not null"`
	Title         string `gorm:"not null"`
	Status        string `gorm:"not null"`
	Attempts      int    `gorm:"not null"`
	NextAttemptAt time.Time
	WaitReason    string `gorm:"column:wait_reason;not null"`
	LastError     string `gorm:"column:last_error;not null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (assistantRequestModel) TableName() string { return "assistant_requests" }

func assistantTx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("assistant: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// AssistantRequestRepo implements assistant.RequestRepository.
type AssistantRequestRepo struct{ db *gorm.DB }

func NewAssistantRequestRepo(db *gorm.DB) *AssistantRequestRepo { return &AssistantRequestRepo{db: db} }

func (r *AssistantRequestRepo) Create(ctx context.Context, req *assistant.Request) error {
	tx, err := assistantTx(ctx)
	if err != nil {
		return err
	}
	m := toAssistantRequestModel(*req)
	m.ID = ""
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("assistantRequestRepo.Create: %w", err)
	}
	req.ID = m.ID
	return nil
}

func (r *AssistantRequestRepo) Get(ctx context.Context, id string) (*assistant.Request, error) {
	return r.get(ctx, id, false)
}

func (r *AssistantRequestRepo) GetForUpdate(ctx context.Context, id string) (*assistant.Request, error) {
	return r.get(ctx, id, true)
}

func (r *AssistantRequestRepo) get(ctx context.Context, id string, lock bool) (*assistant.Request, error) {
	tx, err := assistantTx(ctx)
	if err != nil {
		return nil, err
	}
	if lock {
		tx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var m assistantRequestModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, assistant.ErrRequestNotFound
		}
		return nil, fmt.Errorf("assistantRequestRepo.Get: %w", err)
	}
	out := toAssistantRequest(m)
	return &out, nil
}

// List pages newest first by (created_at, id), continuing after the
// cursor row.
func (r *AssistantRequestRepo) List(ctx context.Context, f assistant.ListFilter) ([]assistant.Request, error) {
	tx, err := assistantTx(ctx)
	if err != nil {
		return nil, err
	}
	q := tx.Model(&assistantRequestModel{})
	if f.Status != "" {
		q = q.Where("status = ?", string(f.Status))
	}
	if f.Cursor != "" {
		q = q.Where("(created_at, id) < (SELECT created_at, id FROM assistant_requests WHERE id = ?)", f.Cursor)
	}
	var rows []assistantRequestModel
	if err := q.Order("created_at DESC, id DESC").Limit(f.Limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("assistantRequestRepo.List: %w", err)
	}
	return toAssistantRequests(rows), nil
}

func (r *AssistantRequestRepo) Update(ctx context.Context, req *assistant.Request) error {
	tx, err := assistantTx(ctx)
	if err != nil {
		return err
	}
	m := toAssistantRequestModel(*req)
	res := tx.Model(&assistantRequestModel{}).Where("id = ?", req.ID).Updates(map[string]any{
		"title": m.Title, "status": m.Status, "attempts": m.Attempts, "next_attempt_at": m.NextAttemptAt,
		"wait_reason": m.WaitReason, "last_error": m.LastError, "updated_at": m.UpdatedAt,
	})
	if res.Error != nil {
		return fmt.Errorf("assistantRequestRepo.Update: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return assistant.ErrRequestNotFound
	}
	return nil
}

func (r *AssistantRequestRepo) LockDue(ctx context.Context, now time.Time, limit int) ([]assistant.Request, error) {
	tx, err := assistantTx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []assistantRequestModel
	err = tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("status = ? AND next_attempt_at <= ?", string(assistant.StatusQueued), now).
		Order("next_attempt_at, id").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("assistantRequestRepo.LockDue: %w", err)
	}
	return toAssistantRequests(rows), nil
}

func (r *AssistantRequestRepo) RequeueStale(ctx context.Context, cutoff, now time.Time) (int64, error) {
	tx, err := assistantTx(ctx)
	if err != nil {
		return 0, err
	}
	res := tx.Model(&assistantRequestModel{}).
		Where("status = ? AND updated_at < ?", string(assistant.StatusProcessing), cutoff).
		Updates(map[string]any{"status": string(assistant.StatusQueued), "next_attempt_at": now, "updated_at": now})
	if res.Error != nil {
		return 0, fmt.Errorf("assistantRequestRepo.RequeueStale: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func toAssistantRequestModel(r assistant.Request) assistantRequestModel {
	return assistantRequestModel{
		ID: r.ID, TenantID: r.TenantID, UserID: r.UserID, UserRole: r.UserRole, Title: r.Title,
		Status: string(r.Status), Attempts: r.Attempts, NextAttemptAt: r.NextAttemptAt,
		WaitReason: string(r.WaitReason), LastError: r.LastError, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func toAssistantRequest(m assistantRequestModel) assistant.Request {
	return assistant.Request{
		ID: m.ID, TenantID: m.TenantID, UserID: m.UserID, UserRole: m.UserRole, Title: m.Title,
		Status: assistant.Status(m.Status), Attempts: m.Attempts, NextAttemptAt: m.NextAttemptAt.UTC(),
		WaitReason: assistant.WaitReason(m.WaitReason), LastError: m.LastError,
		CreatedAt: m.CreatedAt.UTC(), UpdatedAt: m.UpdatedAt.UTC(),
	}
}

func toAssistantRequests(rows []assistantRequestModel) []assistant.Request {
	out := make([]assistant.Request, len(rows))
	for i, m := range rows {
		out[i] = toAssistantRequest(m)
	}
	return out
}

type assistantMessageModel struct {
	ID         string                           `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID   string                           `gorm:"not null;type:uuid"`
	RequestID  string                           `gorm:"column:request_id;not null;type:uuid"`
	Seq        int                              `gorm:"not null"`
	Role       string                           `gorm:"not null"`
	Content    string                           `gorm:"not null"`
	ToolCalls  jsonbValue[[]assistant.ToolCall] `gorm:"column:tool_calls;type:jsonb;not null"`
	ToolCallID string                           `gorm:"column:tool_call_id;not null"`
	ToolName   string                           `gorm:"column:tool_name;not null"`
	Question   bool                             `gorm:"not null"`
	Notice     string                           `gorm:"not null"`
	ProposalID *string                          `gorm:"column:proposal_id;type:uuid"`
	CreatedAt  time.Time
}

func (assistantMessageModel) TableName() string { return "assistant_messages" }

// AssistantMessageRepo implements assistant.MessageRepository.
type AssistantMessageRepo struct{ db *gorm.DB }

func NewAssistantMessageRepo(db *gorm.DB) *AssistantMessageRepo { return &AssistantMessageRepo{db: db} }

// Append numbers the message after the request's last one. Callers append
// to one request from one place at a time (the request row is locked, or
// it was just created); the (request_id, seq) key catches any other case.
func (r *AssistantMessageRepo) Append(ctx context.Context, msg *assistant.Message) error {
	tx, err := assistantTx(ctx)
	if err != nil {
		return err
	}
	var last int
	if err := tx.Model(&assistantMessageModel{}).Where("request_id = ?", msg.RequestID).
		Select("COALESCE(MAX(seq), 0)").Scan(&last).Error; err != nil {
		return fmt.Errorf("assistantMessageRepo.Append: next seq: %w", err)
	}
	calls := msg.ToolCalls
	if calls == nil {
		calls = []assistant.ToolCall{}
	}
	m := assistantMessageModel{
		TenantID: msg.TenantID, RequestID: msg.RequestID, Seq: last + 1, Role: string(msg.Role), Content: msg.Content,
		ToolCalls: jsonbValue[[]assistant.ToolCall]{V: calls}, ToolCallID: msg.ToolCallID, ToolName: msg.ToolName,
		Question: msg.Question, Notice: string(msg.Notice), CreatedAt: msg.CreatedAt,
	}
	if msg.ProposalID != "" {
		m.ProposalID = &msg.ProposalID
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("assistantMessageRepo.Append: %w", err)
	}
	msg.ID, msg.Seq, msg.CreatedAt = m.ID, m.Seq, m.CreatedAt
	return nil
}

func (r *AssistantMessageRepo) ListByRequest(ctx context.Context, requestID string) ([]assistant.Message, error) {
	tx, err := assistantTx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []assistantMessageModel
	if err := tx.Where("request_id = ?", requestID).Order("seq").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("assistantMessageRepo.ListByRequest: %w", err)
	}
	out := make([]assistant.Message, len(rows))
	for i, m := range rows {
		calls := m.ToolCalls.V
		if len(calls) == 0 {
			calls = nil
		}
		out[i] = assistant.Message{
			ID: m.ID, RequestID: m.RequestID, TenantID: m.TenantID, Seq: m.Seq, Role: assistant.Role(m.Role),
			Content: m.Content, ToolCalls: calls, ToolCallID: m.ToolCallID, ToolName: m.ToolName,
			Question: m.Question, Notice: assistant.Notice(m.Notice), CreatedAt: m.CreatedAt.UTC(),
		}
		if m.ProposalID != nil {
			out[i].ProposalID = *m.ProposalID
		}
	}
	return out, nil
}
