package repositories

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type outboxModel struct {
	ID          uint       `gorm:"primaryKey;autoIncrement"`
	EventID     string     `gorm:"uniqueIndex;not null"`
	Subject     string     `gorm:"not null"`
	Payload     []byte     `gorm:"type:jsonb;not null"`
	Status      string     `gorm:"default:'pending';index"`
	OccurredAt  time.Time  `gorm:"not null"`
	PublishedAt *time.Time
	CreatedAt   time.Time
}

func (outboxModel) TableName() string { return "outbox_messages" }

type OutboxRepository struct {
	db *gorm.DB
}

func NewOutboxRepository(db *gorm.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

// Enqueue must be called inside tenant.WithTenant — tx is extracted from ctx.
func (r *OutboxRepository) Enqueue(ctx context.Context, subject, eventID string, payload []byte, occurredAt time.Time) error {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return errors.New("outbox: Enqueue called outside a transaction")
	}
	return tx.WithContext(ctx).Create(&outboxModel{
		EventID:    eventID,
		Subject:    subject,
		Payload:    payload,
		Status:     string(outbox.StatusPending),
		OccurredAt: occurredAt,
	}).Error
}

func (r *OutboxRepository) FetchPending(ctx context.Context, limit int) ([]outbox.Message, error) {
	var rows []outboxModel
	err := r.db.WithContext(ctx).
		Where("status = ?", outbox.StatusPending).
		Order("id asc").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	msgs := make([]outbox.Message, len(rows))
	for i, row := range rows {
		msgs[i] = outbox.Message{
			ID:          row.ID,
			EventID:     row.EventID,
			Subject:     row.Subject,
			Payload:     row.Payload,
			Status:      outbox.Status(row.Status),
			OccurredAt:  row.OccurredAt,
			PublishedAt: row.PublishedAt,
			CreatedAt:   row.CreatedAt,
		}
	}
	return msgs, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&outboxModel{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": string(outbox.StatusPublished), "published_at": now}).Error
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&outboxModel{}).
		Where("id = ?", id).
		Update("status", string(outbox.StatusFailed)).Error
}
