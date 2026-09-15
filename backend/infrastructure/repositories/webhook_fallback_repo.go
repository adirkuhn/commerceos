package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type webhookFallbackEventModel struct {
	ID         string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID   string `gorm:"not null;type:uuid"`
	Provider   string `gorm:"not null"`
	EventID    string `gorm:"not null"`
	Payload    []byte `gorm:"type:jsonb;not null"`
	Headers    []byte `gorm:"type:jsonb;not null"`
	Status     string `gorm:"not null"`
	ReceivedAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (webhookFallbackEventModel) TableName() string { return "webhook_fallback_events" }

type WebhookFallbackRepo struct{ db *gorm.DB }

func NewWebhookFallbackRepo(db *gorm.DB) *WebhookFallbackRepo { return &WebhookFallbackRepo{db: db} }

func (r *WebhookFallbackRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("webhookFallbackRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Save is an upsert on (tenant_id, provider, event_id): ReceiveWebhookUseCase
// can legitimately be re-invoked for a redelivered webhook whose earlier
// delivery already landed here (e.g. the vendor retried after a first
// attempt that also failed to publish) -- ON CONFLICT DO NOTHING keeps that
// idempotent instead of erroring on the unique constraint from migration 093.
func (r *WebhookFallbackRepo) Save(ctx context.Context, e *webhooks.FallbackEvent) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	headers, err := json.Marshal(e.Headers)
	if err != nil {
		return fmt.Errorf("webhookFallbackRepo.Save: marshal headers: %w", err)
	}
	m := webhookFallbackEventModel{
		TenantID: e.TenantID, Provider: e.Provider, EventID: e.EventID,
		Payload: e.Payload, Headers: headers, Status: string(e.Status), ReceivedAt: e.ReceivedAt,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "provider"}, {Name: "event_id"}}, DoNothing: true,
	}).Create(&m).Error; err != nil {
		return fmt.Errorf("webhookFallbackRepo.Save: %w", err)
	}
	e.ID = m.ID
	return nil
}

func (r *WebhookFallbackRepo) Get(ctx context.Context, id string) (*webhooks.FallbackEvent, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m webhookFallbackEventModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, webhooks.ErrFallbackEventNotFound
		}
		return nil, fmt.Errorf("webhookFallbackRepo.Get: %w", err)
	}
	return toFallbackEventDomain(m)
}

func (r *WebhookFallbackRepo) List(ctx context.Context, status webhooks.FallbackStatus, limit, offset int) ([]webhooks.FallbackEvent, int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, 0, err
	}
	q := tx.Model(&webhookFallbackEventModel{})
	if status != "" {
		q = q.Where("status = ?", string(status))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("webhookFallbackRepo.List: count: %w", err)
	}
	var rows []webhookFallbackEventModel
	if err := q.Order("received_at desc").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("webhookFallbackRepo.List: %w", err)
	}
	out := make([]webhooks.FallbackEvent, len(rows))
	for i, row := range rows {
		e, err := toFallbackEventDomain(row)
		if err != nil {
			return nil, 0, err
		}
		out[i] = *e
	}
	return out, total, nil
}

func (r *WebhookFallbackRepo) UpdateStatusForAll(ctx context.Context, fromStatus, toStatus webhooks.FallbackStatus) (int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return 0, err
	}
	res := tx.Model(&webhookFallbackEventModel{}).Where("status = ?", string(fromStatus)).
		Updates(map[string]any{"status": string(toStatus), "updated_at": time.Now().UTC()})
	if res.Error != nil {
		return 0, fmt.Errorf("webhookFallbackRepo.UpdateStatusForAll: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *WebhookFallbackRepo) UpdateStatus(ctx context.Context, id string, status webhooks.FallbackStatus) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Model(&webhookFallbackEventModel{}).Where("id = ?", id).
		Updates(map[string]any{"status": string(status), "updated_at": time.Now().UTC()})
	if res.Error != nil {
		return fmt.Errorf("webhookFallbackRepo.UpdateStatus: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return webhooks.ErrFallbackEventNotFound
	}
	return nil
}

func toFallbackEventDomain(m webhookFallbackEventModel) (*webhooks.FallbackEvent, error) {
	var headers map[string]string
	if len(m.Headers) > 0 {
		if err := json.Unmarshal(m.Headers, &headers); err != nil {
			return nil, fmt.Errorf("webhookFallbackRepo: unmarshal headers: %w", err)
		}
	}
	return &webhooks.FallbackEvent{
		ID: m.ID, TenantID: m.TenantID, Provider: m.Provider, EventID: m.EventID,
		Payload: m.Payload, Headers: headers, Status: webhooks.FallbackStatus(m.Status),
		ReceivedAt: m.ReceivedAt, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}, nil
}
