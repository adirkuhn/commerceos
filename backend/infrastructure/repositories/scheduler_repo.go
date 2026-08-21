package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	schedulerDomain "github.com/adirkuhn/commerceos/domain/scheduler"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type scheduledJobModel struct {
	ID              int64  `gorm:"primaryKey"`
	JobKey          string `gorm:"column:job_key;not null"`
	Enabled         bool   `gorm:"not null"`
	IntervalMinutes int    `gorm:"column:interval_minutes;not null"`
	LastRunAt       *time.Time
	LastRunStatus   string `gorm:"not null"`
	LastRunError    string `gorm:"not null"`
	NextRunAt       time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (scheduledJobModel) TableName() string { return "scheduled_jobs" }

type ScheduledJobRepo struct{ db *gorm.DB }

func NewScheduledJobRepo(db *gorm.DB) *ScheduledJobRepo { return &ScheduledJobRepo{db: db} }

func (r *ScheduledJobRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("scheduledJobRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ScheduledJobRepo) ListAll(ctx context.Context) ([]schedulerDomain.Job, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []scheduledJobModel
	if err := tx.Order("job_key").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("scheduledJobRepo.ListAll: %w", err)
	}
	out := make([]schedulerDomain.Job, len(rows))
	for i, row := range rows {
		out[i] = toJobDomain(row)
	}
	return out, nil
}

func (r *ScheduledJobRepo) ListDue(ctx context.Context, now time.Time) ([]schedulerDomain.Job, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []scheduledJobModel
	if err := tx.Where("enabled = true AND next_run_at <= ?", now).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("scheduledJobRepo.ListDue: %w", err)
	}
	out := make([]schedulerDomain.Job, len(rows))
	for i, row := range rows {
		out[i] = toJobDomain(row)
	}
	return out, nil
}

func (r *ScheduledJobRepo) FindByKey(ctx context.Context, jobKey string) (*schedulerDomain.Job, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var row scheduledJobModel
	err = tx.Where("job_key = ?", jobKey).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, schedulerDomain.ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scheduledJobRepo.FindByKey: %w", err)
	}
	job := toJobDomain(row)
	return &job, nil
}

func (r *ScheduledJobRepo) RecordRunResult(ctx context.Context, jobKey string, ranAt time.Time, status, errMsg string, nextRunAt time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Model(&scheduledJobModel{}).Where("job_key = ?", jobKey).Updates(map[string]any{
		"last_run_at": ranAt, "last_run_status": status, "last_run_error": errMsg,
		"next_run_at": nextRunAt, "updated_at": ranAt,
	})
	if res.Error != nil {
		return fmt.Errorf("scheduledJobRepo.RecordRunResult: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return schedulerDomain.ErrJobNotFound
	}
	return nil
}

func (r *ScheduledJobRepo) UpdateConfig(ctx context.Context, jobKey string, enabled bool, intervalMinutes int) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Model(&scheduledJobModel{}).Where("job_key = ?", jobKey).
		Updates(map[string]any{"enabled": enabled, "interval_minutes": intervalMinutes, "updated_at": time.Now()})
	if res.Error != nil {
		return fmt.Errorf("scheduledJobRepo.UpdateConfig: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return schedulerDomain.ErrJobNotFound
	}
	return nil
}

func (r *ScheduledJobRepo) TriggerNow(ctx context.Context, jobKey string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Model(&scheduledJobModel{}).Where("job_key = ?", jobKey).
		Updates(map[string]any{"next_run_at": time.Now(), "updated_at": time.Now()})
	if res.Error != nil {
		return fmt.Errorf("scheduledJobRepo.TriggerNow: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return schedulerDomain.ErrJobNotFound
	}
	return nil
}

func toJobDomain(m scheduledJobModel) schedulerDomain.Job {
	return schedulerDomain.Job{
		ID: m.ID, JobKey: m.JobKey, Enabled: m.Enabled, IntervalMinutes: m.IntervalMinutes,
		LastRunAt: m.LastRunAt, LastRunStatus: m.LastRunStatus, LastRunError: m.LastRunError,
		NextRunAt: m.NextRunAt, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
