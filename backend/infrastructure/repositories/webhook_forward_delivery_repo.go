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

type webhookForwardDeliveryModel struct {
	ID             string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID       string `gorm:"not null;type:uuid"`
	RuleID         string `gorm:"column:rule_id;not null;type:uuid"`
	WebhookEventID string `gorm:"column:webhook_event_id;not null;type:uuid"`
	Provider       string `gorm:"not null"`
	Topic          string `gorm:"not null"`
	TargetURL      string `gorm:"column:target_url;not null"`
	Payload        []byte `gorm:"type:jsonb;not null"`
	Status         string `gorm:"not null"`
	Attempts       int    `gorm:"not null;default:0"`
	LastError      string `gorm:"column:last_error;not null"`
	NextAttemptAt  time.Time
	DeliveredAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (webhookForwardDeliveryModel) TableName() string { return "webhook_forward_deliveries" }

type WebhookForwardDeliveryRepo struct{ db *gorm.DB }

func NewWebhookForwardDeliveryRepo(db *gorm.DB) *WebhookForwardDeliveryRepo {
	return &WebhookForwardDeliveryRepo{db: db}
}

func (r *WebhookForwardDeliveryRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("webhookForwardDeliveryRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Create is a no-op (not an error) on a (webhook_event_id, rule_id)
// conflict -- see ForwardDeliveryRepository.Create's doc comment on why a
// redelivered envelope must never fan out twice.
func (r *WebhookForwardDeliveryRepo) Create(ctx context.Context, d *webhooks.ForwardDelivery) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	nextAttemptAt := d.NextAttemptAt
	if nextAttemptAt.IsZero() {
		nextAttemptAt = time.Now()
	}
	m := webhookForwardDeliveryModel{
		TenantID: d.TenantID, RuleID: d.RuleID, WebhookEventID: d.WebhookEventID,
		Provider: d.Provider, Topic: d.Topic, TargetURL: d.TargetURL, Payload: d.Payload,
		Status: string(webhooks.ForwardDeliveryStatusPending), NextAttemptAt: nextAttemptAt,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "webhook_event_id"}, {Name: "rule_id"}}, DoNothing: true,
	}).Create(&m).Error; err != nil {
		return fmt.Errorf("webhookForwardDeliveryRepo.Create: %w", err)
	}
	d.ID = m.ID
	return nil
}

func (r *WebhookForwardDeliveryRepo) ListDue(ctx context.Context, now time.Time, limit int) ([]webhooks.ForwardDelivery, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []webhookForwardDeliveryModel
	err = tx.Where("status = ? AND next_attempt_at <= ?", string(webhooks.ForwardDeliveryStatusPending), now).
		Order("next_attempt_at asc").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("webhookForwardDeliveryRepo.ListDue: %w", err)
	}
	out := make([]webhooks.ForwardDelivery, len(rows))
	for i, row := range rows {
		out[i] = toForwardDeliveryDomain(row)
	}
	return out, nil
}

func (r *WebhookForwardDeliveryRepo) MarkDelivered(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	res := tx.Model(&webhookForwardDeliveryModel{}).Where("id = ?", id).Updates(map[string]any{
		"status": string(webhooks.ForwardDeliveryStatusDelivered), "delivered_at": now, "updated_at": now,
	})
	if res.Error != nil {
		return fmt.Errorf("webhookForwardDeliveryRepo.MarkDelivered: %w", res.Error)
	}
	return nil
}

func (r *WebhookForwardDeliveryRepo) MarkFailed(ctx context.Context, id string, errMsg string, nextAttemptAt time.Time, maxAttempts int) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	var current webhookForwardDeliveryModel
	if err := tx.Select("attempts").Where("id = ?", id).First(&current).Error; err != nil {
		return fmt.Errorf("webhookForwardDeliveryRepo.MarkFailed: load: %w", err)
	}
	attempts := current.Attempts + 1
	status := string(webhooks.ForwardDeliveryStatusPending)
	if attempts >= maxAttempts {
		status = string(webhooks.ForwardDeliveryStatusFailed)
	}
	res := tx.Model(&webhookForwardDeliveryModel{}).Where("id = ?", id).Updates(map[string]any{
		"status": status, "attempts": attempts, "last_error": errMsg,
		"next_attempt_at": nextAttemptAt, "updated_at": time.Now(),
	})
	if res.Error != nil {
		return fmt.Errorf("webhookForwardDeliveryRepo.MarkFailed: %w", res.Error)
	}
	return nil
}

func (r *WebhookForwardDeliveryRepo) ListRecent(ctx context.Context, limit int) ([]webhooks.ForwardDelivery, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []webhookForwardDeliveryModel
	if err := tx.Order("created_at desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("webhookForwardDeliveryRepo.ListRecent: %w", err)
	}
	out := make([]webhooks.ForwardDelivery, len(rows))
	for i, row := range rows {
		out[i] = toForwardDeliveryDomain(row)
	}
	return out, nil
}

func toForwardDeliveryDomain(m webhookForwardDeliveryModel) webhooks.ForwardDelivery {
	return webhooks.ForwardDelivery{
		ID: m.ID, TenantID: m.TenantID, RuleID: m.RuleID, WebhookEventID: m.WebhookEventID,
		Provider: m.Provider, Topic: m.Topic, TargetURL: m.TargetURL, Payload: m.Payload,
		Status: webhooks.ForwardDeliveryStatus(m.Status), Attempts: m.Attempts, LastError: m.LastError,
		NextAttemptAt: m.NextAttemptAt, DeliveredAt: m.DeliveredAt, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
