package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type productChannelListingModel struct {
	ID            string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID      string `gorm:"not null;type:uuid"`
	ProductID     string `gorm:"not null;type:uuid"`
	ConnectionID  string `gorm:"not null;type:uuid"`
	Provider      string `gorm:"not null"`
	ExternalID    string `gorm:"not null"`
	SyncStatus    string `gorm:"not null"`
	LastSyncedAt  *time.Time
	LastSyncError string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (productChannelListingModel) TableName() string { return "product_channel_listings" }

type ProductChannelListingRepo struct{ db *gorm.DB }

func NewProductChannelListingRepo(db *gorm.DB) *ProductChannelListingRepo {
	return &ProductChannelListingRepo{db: db}
}

func (r *ProductChannelListingRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("channels: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ProductChannelListingRepo) Create(ctx context.Context, l *channelsDomain.ProductChannelListing) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := productChannelListingModel{
		TenantID:      l.TenantID,
		ProductID:     l.ProductID,
		ConnectionID:  l.ConnectionID,
		Provider:      l.Provider,
		ExternalID:    l.ExternalID,
		SyncStatus:    string(l.SyncStatus),
		LastSyncedAt:  l.LastSyncedAt,
		LastSyncError: l.LastSyncError,
		CreatedAt:     l.CreatedAt,
		UpdatedAt:     l.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("productChannelListingRepo.Create: %w", err)
	}
	l.ID = m.ID
	return nil
}

func (r *ProductChannelListingRepo) FindByExternalID(ctx context.Context, connectionID, externalID string) (*channelsDomain.ProductChannelListing, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m productChannelListingModel
	if err := tx.Where("connection_id = ? AND external_id = ?", connectionID, externalID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, channelsDomain.ErrListingNotFound
		}
		return nil, fmt.Errorf("productChannelListingRepo.FindByExternalID: %w", err)
	}
	return toListingDomain(m), nil
}

func (r *ProductChannelListingRepo) FindByProductAndConnection(ctx context.Context, productID, connectionID string) (*channelsDomain.ProductChannelListing, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m productChannelListingModel
	if err := tx.Where("product_id = ? AND connection_id = ?", productID, connectionID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, channelsDomain.ErrListingNotFound
		}
		return nil, fmt.Errorf("productChannelListingRepo.FindByProductAndConnection: %w", err)
	}
	return toListingDomain(m), nil
}

func (r *ProductChannelListingRepo) ListByConnection(ctx context.Context, connectionID string) ([]channelsDomain.ProductChannelListing, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []productChannelListingModel
	if err := tx.Where("connection_id = ?", connectionID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("productChannelListingRepo.ListByConnection: %w", err)
	}
	out := make([]channelsDomain.ProductChannelListing, len(rows))
	for i, row := range rows {
		out[i] = *toListingDomain(row)
	}
	return out, nil
}

func (r *ProductChannelListingRepo) ListByProduct(ctx context.Context, productID string) ([]channelsDomain.ProductChannelListing, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []productChannelListingModel
	if err := tx.Where("product_id = ?", productID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("productChannelListingRepo.ListByProduct: %w", err)
	}
	out := make([]channelsDomain.ProductChannelListing, len(rows))
	for i, row := range rows {
		out[i] = *toListingDomain(row)
	}
	return out, nil
}

func (r *ProductChannelListingRepo) Update(ctx context.Context, l *channelsDomain.ProductChannelListing) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&productChannelListingModel{}).Where("id = ?", l.ID).
		Updates(map[string]any{
			"sync_status":     string(l.SyncStatus),
			"last_synced_at":  l.LastSyncedAt,
			"last_sync_error": l.LastSyncError,
			"updated_at":      l.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("productChannelListingRepo.Update: %w", err)
	}
	return nil
}

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

func (r *SyncScheduleRepo) RecordRunResult(ctx context.Context, id string, ranAt time.Time, status, errMsg string, nextRunAt time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&syncScheduleModel{}).Where("id = ?", id).
		Updates(map[string]any{
			"last_run_at":     ranAt,
			"last_run_status": status,
			"last_run_error":  errMsg,
			"next_run_at":     nextRunAt,
			"updated_at":      ranAt,
		}).Error; err != nil {
		return fmt.Errorf("syncScheduleRepo.RecordRunResult: %w", err)
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
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

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
	Errors       string `gorm:"type:jsonb;not null;default:'[]'"`
	ErrorMessage string `gorm:"not null"`
	CreatedAt    time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
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
	_ = json.Unmarshal([]byte(raw), &out)
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

func (r *ImportJobRepo) MarkOrphanedRunningAsFailed(ctx context.Context, errMsg string, finishedAt time.Time) (int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return 0, err
	}
	result := tx.Model(&importJobModel{}).Where("status = ?", string(channelsDomain.ImportJobRunning)).
		Updates(map[string]any{"status": string(channelsDomain.ImportJobFailed), "error_message": errMsg, "finished_at": finishedAt})
	if result.Error != nil {
		return 0, fmt.Errorf("importJobRepo.MarkOrphanedRunningAsFailed: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func toImportJobDomain(m importJobModel) *channelsDomain.ImportJob {
	return &channelsDomain.ImportJob{
		ID: m.ID, TenantID: m.TenantID, ConnectionID: m.ConnectionID,
		DefaultWarehouseID: m.DefaultWarehouseID, DefaultCurrency: m.DefaultCurrency,
		Status:   channelsDomain.ImportJobStatus(m.Status),
		Imported: m.Imported, AlreadyLinked: m.AlreadyLinked, SkippedType: m.SkippedType,
		Errors: errorsFromJSON(m.Errors), ErrorMessage: m.ErrorMessage,
		CreatedAt: m.CreatedAt, StartedAt: m.StartedAt, FinishedAt: m.FinishedAt,
	}
}

func toListingDomain(m productChannelListingModel) *channelsDomain.ProductChannelListing {
	return &channelsDomain.ProductChannelListing{
		ID:            m.ID,
		TenantID:      m.TenantID,
		ProductID:     m.ProductID,
		ConnectionID:  m.ConnectionID,
		Provider:      m.Provider,
		ExternalID:    m.ExternalID,
		SyncStatus:    channelsDomain.SyncStatus(m.SyncStatus),
		LastSyncedAt:  m.LastSyncedAt,
		LastSyncError: m.LastSyncError,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}
