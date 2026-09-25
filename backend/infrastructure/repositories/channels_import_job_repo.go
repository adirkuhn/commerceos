package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── Import job repo ──────────────────────────────────────────────────────────

type importJobModel struct {
	ID                 string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID           string `gorm:"not null;type:uuid"`
	ConnectionID       string `gorm:"not null;type:uuid"`
	DefaultWarehouseID string `gorm:"not null;type:uuid"`
	DefaultCurrency    string `gorm:"not null"`
	Status             string `gorm:"not null"`
	Imported           int    `gorm:"not null"`
	AlreadyLinked      int    `gorm:"not null"`
	SkippedType        int    `gorm:"not null"`
	// Errors is hand-marshaled JSON (via errorsToJSON/errorsFromJSON), not
	// GORM's `serializer:json` tag — that serializer only fires for
	// struct-based operations (Create, Find). MarkCompleted updates via a
	// map[string]any (needed so legitimate zero values like
	// already_linked:0 are actually written — GORM's struct-based Updates
	// silently skips zero-valued fields), and a serializer tag never
	// applies to a plain map key, so an empty/nil []string there wrote SQL
	// NULL and violated this column's NOT NULL constraint.
	Errors        string `gorm:"type:jsonb;not null;default:'[]'"`
	ErrorMessage  string `gorm:"not null"`
	CreatedAt     time.Time
	StartedAt     *time.Time
	FinishedAt    *time.Time
	RetryCount    int  `gorm:"column:retry_count;not null;default:0"`
	IsFirstImport bool `gorm:"column:is_first_import;not null;default:false"`
}

func errorsToJSON(errs []string) string {
	if errs == nil {
		errs = []string{}
	}
	b, err := json.Marshal(errs)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func errorsFromJSON(raw string) []string {
	var out []string
	if raw == "" {
		return out
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		log.Printf("channelsRepo: corrupt stored JSON, treating as empty: %v", err)
	}
	return out
}

func (importJobModel) TableName() string { return "import_jobs" }

type ImportJobRepo struct{ db *gorm.DB }

func NewImportJobRepo(db *gorm.DB) *ImportJobRepo { return &ImportJobRepo{db: db} }

func (r *ImportJobRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("channels: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ImportJobRepo) Create(ctx context.Context, j *channelsDomain.ImportJob) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := importJobModel{
		TenantID:           j.TenantID,
		ConnectionID:       j.ConnectionID,
		DefaultWarehouseID: j.DefaultWarehouseID,
		DefaultCurrency:    j.DefaultCurrency,
		Status:             string(j.Status),
		Errors:             errorsToJSON(nil),
		CreatedAt:          j.CreatedAt,
		IsFirstImport:      j.IsFirstImport,
	}
	if err := tx.Create(&m).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return channelsDomain.ErrImportJobAlreadyActive
		}
		return fmt.Errorf("importJobRepo.Create: %w", err)
	}
	j.ID = m.ID
	return nil
}

func (r *ImportJobRepo) HasActiveForConnection(ctx context.Context, connectionID string) (bool, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return false, err
	}
	var count int64
	if err := tx.Model(&importJobModel{}).
		Where("connection_id = ? AND status IN ?", connectionID, []string{
			string(channelsDomain.ImportJobPending), string(channelsDomain.ImportJobRunning),
		}).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("importJobRepo.HasActiveForConnection: %w", err)
	}
	return count > 0, nil
}

func (r *ImportJobRepo) ListPending(ctx context.Context) ([]channelsDomain.ImportJob, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []importJobModel
	if err := tx.Where("status = ?", string(channelsDomain.ImportJobPending)).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("importJobRepo.ListPending: %w", err)
	}
	out := make([]channelsDomain.ImportJob, len(rows))
	for i, row := range rows {
		out[i] = *toImportJobDomain(row)
	}
	return out, nil
}

func (r *ImportJobRepo) MarkRunning(ctx context.Context, id string, startedAt time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&importJobModel{}).Where("id = ?", id).
		Updates(map[string]any{"status": string(channelsDomain.ImportJobRunning), "started_at": startedAt}).Error; err != nil {
		return fmt.Errorf("importJobRepo.MarkRunning: %w", err)
	}
	return nil
}

func (r *ImportJobRepo) MarkCompleted(ctx context.Context, id string, imported, alreadyLinked, skippedType int, errs []string, finishedAt time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&importJobModel{}).Where("id = ?", id).
		Updates(map[string]any{
			"status": string(channelsDomain.ImportJobCompleted), "imported": imported,
			"already_linked": alreadyLinked, "skipped_type": skippedType, "errors": errorsToJSON(errs), "finished_at": finishedAt,
		}).Error; err != nil {
		return fmt.Errorf("importJobRepo.MarkCompleted: %w", err)
	}
	return nil
}

func (r *ImportJobRepo) MarkFailed(ctx context.Context, id string, errMsg string, finishedAt time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&importJobModel{}).Where("id = ?", id).
		Updates(map[string]any{"status": string(channelsDomain.ImportJobFailed), "error_message": errMsg, "finished_at": finishedAt}).Error; err != nil {
		return fmt.Errorf("importJobRepo.MarkFailed: %w", err)
	}
	return nil
}

func (r *ImportJobRepo) RecoverOrphanedRunning(ctx context.Context, maxRetries int, errMsg string, finishedAt time.Time) (int64, int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return 0, 0, err
	}
	requeued := tx.Model(&importJobModel{}).
		Where("status = ? AND retry_count < ?", string(channelsDomain.ImportJobRunning), maxRetries).
		Updates(map[string]any{
			"status": string(channelsDomain.ImportJobPending), "retry_count": gorm.Expr("retry_count + 1"), "started_at": nil,
		})
	if requeued.Error != nil {
		return 0, 0, fmt.Errorf("importJobRepo.RecoverOrphanedRunning: requeue: %w", requeued.Error)
	}
	// Runs second: rows the requeue above just moved to pending no longer
	// match status = running, so only genuinely exhausted rows are failed.
	failed := tx.Model(&importJobModel{}).Where("status = ?", string(channelsDomain.ImportJobRunning)).
		Updates(map[string]any{"status": string(channelsDomain.ImportJobFailed), "error_message": errMsg, "finished_at": finishedAt})
	if failed.Error != nil {
		return requeued.RowsAffected, 0, fmt.Errorf("importJobRepo.RecoverOrphanedRunning: fail: %w", failed.Error)
	}
	return requeued.RowsAffected, failed.RowsAffected, nil
}

func toImportJobDomain(m importJobModel) *channelsDomain.ImportJob {
	return &channelsDomain.ImportJob{
		ID: m.ID, TenantID: m.TenantID, ConnectionID: m.ConnectionID,
		DefaultWarehouseID: m.DefaultWarehouseID, DefaultCurrency: m.DefaultCurrency,
		Status:   channelsDomain.ImportJobStatus(m.Status),
		Imported: m.Imported, AlreadyLinked: m.AlreadyLinked, SkippedType: m.SkippedType,
		Errors: errorsFromJSON(m.Errors), ErrorMessage: m.ErrorMessage,
		CreatedAt: m.CreatedAt, StartedAt: m.StartedAt, FinishedAt: m.FinishedAt,
		RetryCount: m.RetryCount, IsFirstImport: m.IsFirstImport,
	}
}
