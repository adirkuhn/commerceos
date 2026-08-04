package repositories

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type processedEventModel struct {
	EventID     string `gorm:"primaryKey;column:event_id"`
	Consumer    string `gorm:"primaryKey"`
	ProcessedAt time.Time
}

func (processedEventModel) TableName() string { return "processed_events" }

// ProcessedEventRepo is not tenant-scoped (event-processing bookkeeping,
// not business data — same reasoning as the outbox table), so it talks to
// the plain *gorm.DB directly rather than going through tenant.WithTenant.
type ProcessedEventRepo struct {
	db *gorm.DB
}

func NewProcessedEventRepo(db *gorm.DB) *ProcessedEventRepo {
	return &ProcessedEventRepo{db: db}
}

func (r *ProcessedEventRepo) IsProcessed(ctx context.Context, eventID, consumer string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&processedEventModel{}).
		Where("event_id = ? AND consumer = ?", eventID, consumer).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("processedEventRepo.IsProcessed: %w", err)
	}
	return count > 0, nil
}

// MarkProcessed is itself idempotent (ON CONFLICT DO NOTHING) — two
// concurrent deliveries of the same event both succeeding here is fine,
// the row just ends up written once either way.
func (r *ProcessedEventRepo) MarkProcessed(ctx context.Context, eventID, consumer string) error {
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&processedEventModel{
		EventID: eventID, Consumer: consumer, ProcessedAt: time.Now(),
	}).Error
	if err != nil {
		return fmt.Errorf("processedEventRepo.MarkProcessed: %w", err)
	}
	return nil
}
