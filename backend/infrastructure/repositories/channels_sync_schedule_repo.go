package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── Sync schedule repo ───────────────────────────────────────────────────────

type syncScheduleModel struct {
	ID                 string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID           string `gorm:"not null;type:uuid"`
	ConnectionID       string `gorm:"not null;type:uuid"`
	EntityType         string `gorm:"not null"`
	Direction          string `gorm:"not null"`
	Enabled            bool   `gorm:"not null"`
	IntervalMinutes    int    `gorm:"not null"`
	DefaultWarehouseID *string
	DefaultCurrency    *string
	LastRunAt          *time.Time
	LastRunStatus      string `gorm:"not null"`
	LastRunError       string `gorm:"not null"`
	NextRunAt          *time.Time
	FailingSince       *time.Time
	FailureAlerted     bool `gorm:"not null"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (syncScheduleModel) TableName() string { return "sync_schedules" }

type SyncScheduleRepo struct{ db *gorm.DB }

func NewSyncScheduleRepo(db *gorm.DB) *SyncScheduleRepo { return &SyncScheduleRepo{db: db} }

func (r *SyncScheduleRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("channels: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Upsert finds the existing row for (ConnectionID, EntityType, Direction)
// and updates it, or creates a new one — matches the unique constraint on
// sync_schedules, one config row per combination rather than an
// append-only list.
func (r *SyncScheduleRepo) Upsert(ctx context.Context, s *channelsDomain.SyncSchedule) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	var existing syncScheduleModel
	err = tx.Where("connection_id = ? AND entity_type = ? AND direction = ?",
		s.ConnectionID, string(s.EntityType), string(s.Direction)).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		m := syncScheduleModel{
			TenantID:           s.TenantID,
			ConnectionID:       s.ConnectionID,
			EntityType:         string(s.EntityType),
			Direction:          string(s.Direction),
			Enabled:            s.Enabled,
			IntervalMinutes:    s.IntervalMinutes,
			DefaultWarehouseID: s.DefaultWarehouseID,
			DefaultCurrency:    s.DefaultCurrency,
			NextRunAt:          s.NextRunAt,
			CreatedAt:          s.CreatedAt,
			UpdatedAt:          s.UpdatedAt,
		}
		if err := tx.Create(&m).Error; err != nil {
			return fmt.Errorf("syncScheduleRepo.Upsert: %w", err)
		}
		s.ID = m.ID
		return nil
	case err != nil:
		return fmt.Errorf("syncScheduleRepo.Upsert: %w", err)
	default:
		if err := tx.Model(&syncScheduleModel{}).Where("id = ?", existing.ID).
			Updates(map[string]any{
				"enabled":              s.Enabled,
				"interval_minutes":     s.IntervalMinutes,
				"default_warehouse_id": s.DefaultWarehouseID,
				"default_currency":     s.DefaultCurrency,
				"next_run_at":          s.NextRunAt,
				"updated_at":           s.UpdatedAt,
			}).Error; err != nil {
			return fmt.Errorf("syncScheduleRepo.Upsert: %w", err)
		}
		s.ID = existing.ID
		return nil
	}
}

func (r *SyncScheduleRepo) ListByConnection(ctx context.Context, connectionID string) ([]channelsDomain.SyncSchedule, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []syncScheduleModel
	if err := tx.Where("connection_id = ?", connectionID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("syncScheduleRepo.ListByConnection: %w", err)
	}
	out := make([]channelsDomain.SyncSchedule, len(rows))
	for i, row := range rows {
		out[i] = *toSyncScheduleDomain(row)
	}
	return out, nil
}

func (r *SyncScheduleRepo) ListAll(ctx context.Context) ([]channelsDomain.SyncSchedule, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []syncScheduleModel
	if err := tx.Order("connection_id, entity_type, direction").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("syncScheduleRepo.ListAll: %w", err)
	}
	out := make([]channelsDomain.SyncSchedule, len(rows))
	for i, row := range rows {
		out[i] = *toSyncScheduleDomain(row)
	}
	return out, nil
}

func (r *SyncScheduleRepo) TriggerNow(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	res := tx.Model(&syncScheduleModel{}).Where("id = ?", id).Updates(map[string]any{"next_run_at": now, "updated_at": now})
	if res.Error != nil {
		return fmt.Errorf("syncScheduleRepo.TriggerNow: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return channelsDomain.ErrSyncScheduleNotFound
	}
	return nil
}

func (r *SyncScheduleRepo) ListDue(ctx context.Context, now time.Time) ([]channelsDomain.SyncSchedule, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []syncScheduleModel
	if err := tx.Where("enabled = true AND next_run_at IS NOT NULL AND next_run_at <= ?", now).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("syncScheduleRepo.ListDue: %w", err)
	}
	out := make([]channelsDomain.SyncSchedule, len(rows))
	for i, row := range rows {
		out[i] = *toSyncScheduleDomain(row)
	}
	return out, nil
}

func (r *SyncScheduleRepo) RecordRunResult(ctx context.Context, id string, ranAt time.Time, status, errMsg string, nextRunAt time.Time, cursorAt time.Time, advanceLastRunAt bool) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	fields := map[string]any{
		"last_run_status": status,
		"last_run_error":  errMsg,
		"next_run_at":     nextRunAt,
		"updated_at":      ranAt,
	}
	if advanceLastRunAt {
		fields["last_run_at"] = cursorAt
	}
	if err := tx.Model(&syncScheduleModel{}).Where("id = ?", id).Updates(fields).Error; err != nil {
		return fmt.Errorf("syncScheduleRepo.RecordRunResult: %w", err)
	}
	return nil
}

func (r *SyncScheduleRepo) RecordHealth(ctx context.Context, id string, failingSince *time.Time, failureAlerted bool) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&syncScheduleModel{}).Where("id = ?", id).Updates(map[string]any{
		"failing_since":   failingSince,
		"failure_alerted": failureAlerted,
	}).Error; err != nil {
		return fmt.Errorf("syncScheduleRepo.RecordHealth: %w", err)
	}
	return nil
}

func toSyncScheduleDomain(m syncScheduleModel) *channelsDomain.SyncSchedule {
	return &channelsDomain.SyncSchedule{
		ID:                 m.ID,
		TenantID:           m.TenantID,
		ConnectionID:       m.ConnectionID,
		EntityType:         channelsDomain.SyncEntityType(m.EntityType),
		Direction:          channelsDomain.SyncDirection(m.Direction),
		Enabled:            m.Enabled,
		IntervalMinutes:    m.IntervalMinutes,
		DefaultWarehouseID: m.DefaultWarehouseID,
		DefaultCurrency:    m.DefaultCurrency,
		LastRunAt:          m.LastRunAt,
		LastRunStatus:      m.LastRunStatus,
		LastRunError:       m.LastRunError,
		NextRunAt:          m.NextRunAt,
		FailingSince:       m.FailingSince,
		FailureAlerted:     m.FailureAlerted,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}
