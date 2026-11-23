package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

type assistantProposalModel struct {
	ID        string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string          `gorm:"not null;type:uuid"`
	RequestID string          `gorm:"column:request_id;not null;type:uuid"`
	Kind      string          `gorm:"not null"`
	Status    string          `gorm:"not null"`
	Payload   json.RawMessage `gorm:"type:jsonb;not null"`
	Preview   json.RawMessage `gorm:"type:jsonb;not null"`
	LastError string          `gorm:"column:last_error;not null"`
	ResultID  string          `gorm:"column:result_id;not null"`
	ResultRef string          `gorm:"column:result_ref;not null"`
	DecidedBy string          `gorm:"column:decided_by;not null"`
	DecidedAt *time.Time      `gorm:"column:decided_at"`
	CreatedAt time.Time
}

func (assistantProposalModel) TableName() string { return "assistant_proposals" }

// AssistantProposalRepo implements assistant.ProposalRepository.
type AssistantProposalRepo struct{ db *gorm.DB }

func NewAssistantProposalRepo(db *gorm.DB) *AssistantProposalRepo { return &AssistantProposalRepo{db: db} }

func (r *AssistantProposalRepo) Create(ctx context.Context, p *assistant.Proposal) error {
	tx, err := assistantTx(ctx)
	if err != nil {
		return err
	}
	m := toAssistantProposalModel(*p)
	m.ID = ""
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("assistantProposalRepo.Create: %w", err)
	}
	p.ID = m.ID
	return nil
}

func (r *AssistantProposalRepo) GetForUpdate(ctx context.Context, id string) (*assistant.Proposal, error) {
	tx, err := assistantTx(ctx)
	if err != nil {
		return nil, err
	}
	var m assistantProposalModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, assistant.ErrProposalNotFound
		}
		return nil, fmt.Errorf("assistantProposalRepo.GetForUpdate: %w", err)
	}
	out := toAssistantProposal(m)
	return &out, nil
}

func (r *AssistantProposalRepo) ListByRequest(ctx context.Context, requestID string) ([]assistant.Proposal, error) {
	tx, err := assistantTx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []assistantProposalModel
	if err := tx.Where("request_id = ?", requestID).Order("created_at, id").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("assistantProposalRepo.ListByRequest: %w", err)
	}
	out := make([]assistant.Proposal, len(rows))
	for i, m := range rows {
		out[i] = toAssistantProposal(m)
	}
	return out, nil
}

func (r *AssistantProposalRepo) Update(ctx context.Context, p *assistant.Proposal) error {
	tx, err := assistantTx(ctx)
	if err != nil {
		return err
	}
	res := tx.Model(&assistantProposalModel{}).Where("id = ?", p.ID).Updates(map[string]any{
		"status": string(p.Status), "last_error": p.LastError, "result_id": p.ResultID, "result_ref": p.ResultRef,
		"decided_by": p.DecidedBy, "decided_at": p.DecidedAt,
	})
	if res.Error != nil {
		return fmt.Errorf("assistantProposalRepo.Update: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return assistant.ErrProposalNotFound
	}
	return nil
}

func toAssistantProposalModel(p assistant.Proposal) assistantProposalModel {
	return assistantProposalModel{
		ID: p.ID, TenantID: p.TenantID, RequestID: p.RequestID, Kind: string(p.Kind), Status: string(p.Status),
		Payload: p.Payload, Preview: p.Preview, LastError: p.LastError, ResultID: p.ResultID, ResultRef: p.ResultRef,
		DecidedBy: p.DecidedBy, DecidedAt: p.DecidedAt, CreatedAt: p.CreatedAt,
	}
}

func toAssistantProposal(m assistantProposalModel) assistant.Proposal {
	p := assistant.Proposal{
		ID: m.ID, TenantID: m.TenantID, RequestID: m.RequestID, Kind: assistant.ProposalKind(m.Kind),
		Status: assistant.ProposalStatus(m.Status), Payload: m.Payload, Preview: m.Preview, LastError: m.LastError,
		ResultID: m.ResultID, ResultRef: m.ResultRef, DecidedBy: m.DecidedBy, CreatedAt: m.CreatedAt.UTC(),
	}
	if m.DecidedAt != nil {
		at := m.DecidedAt.UTC()
		p.DecidedAt = &at
	}
	return p
}
