package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type webhookEventModel struct {
	ID          string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID    string `gorm:"not null;type:uuid"`
	Provider    string `gorm:"not null"`
	EventID     string `gorm:"not null"`
	Topic       string `gorm:"not null"`
	Payload     []byte `gorm:"type:jsonb;not null"`
	ReceivedAt  time.Time
	ProcessedAt time.Time
}

func (webhookEventModel) TableName() string { return "webhook_events" }

type WebhookEventRepo struct{ db *gorm.DB }

func NewWebhookEventRepo(db *gorm.DB) *WebhookEventRepo { return &WebhookEventRepo{db: db} }

func (r *WebhookEventRepo) Save(ctx context.Context, e *webhooks.WebhookEvent) error {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return errors.New("webhookEventRepo: Save called outside a transaction")
	}
	txc := tx.WithContext(ctx)
	m := webhookEventModel{
		TenantID: e.TenantID, Provider: e.Provider, EventID: e.EventID, Topic: e.Topic,
		Payload: e.Payload, ReceivedAt: e.ReceivedAt,
	}
	res := txc.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "provider"}, {Name: "event_id"}}, DoNothing: true,
	}).Create(&m)
	if res.Error != nil {
		return fmt.Errorf("webhookEventRepo.Save: %w", res.Error)
	}
	// A conflict (redelivery of an event already persisted) means DoNothing
	// skipped the insert -- m.ID never got the DB-generated default, so it
	// must be looked up by the same unique key instead. Callers
	// (ProcessWebhookEventUseCase) depend on e.ID always being the real
	// row's id, even on this no-op path.
	if res.RowsAffected == 0 {
		if err := txc.Select("id").
			Where("tenant_id = ? AND provider = ? AND event_id = ?", e.TenantID, e.Provider, e.EventID).
			First(&m).Error; err != nil {
			return fmt.Errorf("webhookEventRepo.Save: lookup existing: %w", err)
		}
	}
	e.ID = m.ID
	return nil
}

// List returns this tenant's received webhooks, newest first, optionally
// filtered to one provider -- see WebhookEventRepository.List's doc
// comment.
func (r *WebhookEventRepo) List(ctx context.Context, provider string, limit int) ([]webhooks.WebhookEvent, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("webhookEventRepo: List called outside a transaction")
	}
	q := tx.WithContext(ctx).Order("received_at DESC").Limit(limit)
	if provider != "" {
		q = q.Where("provider = ?", provider)
	}
	var rows []webhookEventModel
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("webhookEventRepo.List: %w", err)
	}
	out := make([]webhooks.WebhookEvent, len(rows))
	for i, m := range rows {
		out[i] = webhooks.WebhookEvent{
			ID: m.ID, TenantID: m.TenantID, Provider: m.Provider, EventID: m.EventID,
			Topic: m.Topic, Payload: m.Payload, ReceivedAt: m.ReceivedAt, ProcessedAt: m.ProcessedAt,
		}
	}
	return out, nil
}

func (r *WebhookEventRepo) DistinctTopics(ctx context.Context) ([]webhooks.TopicRef, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("webhookEventRepo: DistinctTopics called outside a transaction")
	}
	var rows []webhookEventModel
	if err := tx.WithContext(ctx).Distinct("provider", "topic").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("webhookEventRepo.DistinctTopics: %w", err)
	}
	out := make([]webhooks.TopicRef, len(rows))
	for i, row := range rows {
		out[i] = webhooks.TopicRef{Provider: row.Provider, Topic: row.Topic}
	}
	return out, nil
}
