package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type productChannelListingModel struct {
	ID            string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID      string  `gorm:"not null;type:uuid"`
	ProductID     string  `gorm:"not null;type:uuid"`
	ConnectionID  string  `gorm:"not null;type:uuid"`
	Provider      string  `gorm:"not null"`
	ExternalID    string  `gorm:"not null"`
	VariantID     *string `gorm:"column:variant_id;type:uuid"`
	IsVariation   bool    `gorm:"column:is_variation;not null;default:false"`
	SyncStatus    string  `gorm:"not null"`
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
		VariantID:     l.VariantID,
		IsVariation:   l.IsVariation,
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

// FindByProductAndConnection prefers the product-level row (variant_id
// IS NULL) when one exists -- NULLS FIRST puts it first regardless of
// which row happened to be created first -- else returns an arbitrary
// variant-level row. See the interface doc comment for why "arbitrary" is
// fine here.
func (r *ProductChannelListingRepo) FindByProductAndConnection(ctx context.Context, productID, connectionID string) (*channelsDomain.ProductChannelListing, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m productChannelListingModel
	if err := tx.Where("product_id = ? AND connection_id = ?", productID, connectionID).
		Order("variant_id ASC NULLS FIRST").First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, channelsDomain.ErrListingNotFound
		}
		return nil, fmt.Errorf("productChannelListingRepo.FindByProductAndConnection: %w", err)
	}
	return toListingDomain(m), nil
}

func (r *ProductChannelListingRepo) FindByProductVariantAndConnection(ctx context.Context, productID string, variantID *string, connectionID string) (*channelsDomain.ProductChannelListing, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	q := tx.Where("product_id = ? AND connection_id = ?", productID, connectionID)
	if variantID == nil {
		q = q.Where("variant_id IS NULL")
	} else {
		q = q.Where("variant_id = ?", *variantID)
	}
	var m productChannelListingModel
	if err := q.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, channelsDomain.ErrListingNotFound
		}
		return nil, fmt.Errorf("productChannelListingRepo.FindByProductVariantAndConnection: %w", err)
	}
	return toListingDomain(m), nil
}

func (r *ProductChannelListingRepo) ListByProductAndConnection(ctx context.Context, productID, connectionID string) ([]channelsDomain.ProductChannelListing, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []productChannelListingModel
	if err := tx.Where("product_id = ? AND connection_id = ?", productID, connectionID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("productChannelListingRepo.ListByProductAndConnection: %w", err)
	}
	out := make([]channelsDomain.ProductChannelListing, len(rows))
	for i, row := range rows {
		out[i] = *toListingDomain(row)
	}
	return out, nil
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

func (r *ProductChannelListingRepo) ListAll(ctx context.Context) ([]channelsDomain.ProductChannelListing, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []productChannelListingModel
	if err := tx.Order("tenant_id, product_id").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("productChannelListingRepo.ListAll: %w", err)
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

func (r *ProductChannelListingRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&productChannelListingModel{}).Error; err != nil {
		return fmt.Errorf("productChannelListingRepo.Delete: %w", err)
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

// ── sync log ─────────────────────────────────────────────────────────────────

type channelSyncLogModel struct {
	ID           string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string  `gorm:"not null;type:uuid"`
	ListingID    *string `gorm:"type:uuid"`
	ProductID    string  `gorm:"not null;type:uuid"`
	ConnectionID string  `gorm:"not null;type:uuid"`
	Provider     string  `gorm:"not null"`
	ExternalID   string  `gorm:"not null"`
	Direction    string  `gorm:"not null"`
	Status       string  `gorm:"not null"`
	ErrorMessage string  `gorm:"not null;default:''"`
	// Fields/Changes are hand-marshaled JSON strings (not GORM's
	// serializer:json tag) stored in jsonb columns — matches
	// importJobModel.Errors' reasoning: this app never Update()s a sync
	// log row (it's insert-only), but keeping the convention consistent
	// avoids the tag silently doing nothing if that ever changes.
	Fields      string `gorm:"type:jsonb;not null;default:'{}'"`
	Changes     string `gorm:"type:jsonb;not null;default:'{}'"`
	SyncedAt    time.Time
	DismissedAt *time.Time `gorm:"column:dismissed_at"`
}

func (channelSyncLogModel) TableName() string { return "channel_sync_logs" }

type SyncLogRepo struct{ db *gorm.DB }

func NewSyncLogRepo(db *gorm.DB) *SyncLogRepo { return &SyncLogRepo{db: db} }

func (r *SyncLogRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("channels: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *SyncLogRepo) Create(ctx context.Context, e *channelsDomain.SyncLogEntry) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := channelSyncLogModel{
		TenantID: e.TenantID, ListingID: e.ListingID, ProductID: e.ProductID, ConnectionID: e.ConnectionID,
		Provider: e.Provider, ExternalID: e.ExternalID,
		Direction: string(e.Direction), Status: string(e.Status), ErrorMessage: e.ErrorMessage,
		Fields: stringMapToJSON(e.Fields), Changes: changesToJSON(e.Changes),
		SyncedAt: e.SyncedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("syncLogRepo.Create: %w", err)
	}
	e.ID = m.ID
	return nil
}

func (r *SyncLogRepo) FindByID(ctx context.Context, id string) (*channelsDomain.SyncLogEntry, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var row channelSyncLogModel
	err = tx.Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, channelsDomain.ErrSyncLogNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("syncLogRepo.FindByID: %w", err)
	}
	return toSyncLogDomain(row), nil
}

// ListNeedingAttention finds each listing's single most-recent entry (via
// DISTINCT ON, Postgres-specific), then filters that set to error+
// undismissed -- see the interface doc for why this is a live query over
// the append-only log rather than separate tracked state. RLS applies
// transparently to the raw SQL here the same as it does to GORM's own
// query builder, since both run on the same tx (SET LOCAL app.current_tenant
// already applied by tenant.WithTenant before this is ever called).
func (r *SyncLogRepo) ListNeedingAttention(ctx context.Context, page, perPage int) ([]channelsDomain.SyncLogEntry, int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, 0, err
	}

	const latestPerListing = `
		SELECT DISTINCT ON (listing_id) *
		FROM channel_sync_logs
		WHERE listing_id IS NOT NULL
		ORDER BY listing_id, synced_at DESC`

	var total int64
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM (%s) latest WHERE latest.status = 'error' AND latest.dismissed_at IS NULL`, latestPerListing)
	if err := tx.Raw(countSQL).Scan(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("syncLogRepo.ListNeedingAttention: count: %w", err)
	}

	var rows []channelSyncLogModel
	offset := (page - 1) * perPage
	listSQL := fmt.Sprintf(`SELECT * FROM (%s) latest WHERE latest.status = 'error' AND latest.dismissed_at IS NULL ORDER BY latest.synced_at DESC LIMIT ? OFFSET ?`, latestPerListing)
	if err := tx.Raw(listSQL, perPage, offset).Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("syncLogRepo.ListNeedingAttention: %w", err)
	}
	out := make([]channelsDomain.SyncLogEntry, len(rows))
	for i, row := range rows {
		out[i] = *toSyncLogDomain(row)
	}
	return out, total, nil
}

func (r *SyncLogRepo) Dismiss(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	res := tx.Model(&channelSyncLogModel{}).Where("id = ?", id).Update("dismissed_at", now)
	if res.Error != nil {
		return fmt.Errorf("syncLogRepo.Dismiss: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return channelsDomain.ErrSyncLogNotFound
	}
	return nil
}

func (r *SyncLogRepo) ListByListing(ctx context.Context, listingID string, limit int) ([]channelsDomain.SyncLogEntry, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []channelSyncLogModel
	if err := tx.Where("listing_id = ?", listingID).Order("synced_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("syncLogRepo.ListByListing: %w", err)
	}
	out := make([]channelsDomain.SyncLogEntry, len(rows))
	for i, row := range rows {
		out[i] = *toSyncLogDomain(row)
	}
	return out, nil
}

func (r *SyncLogRepo) ListAll(ctx context.Context, page, perPage int) ([]channelsDomain.SyncLogEntry, int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	if err := tx.Model(&channelSyncLogModel{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("syncLogRepo.ListAll: count: %w", err)
	}
	var rows []channelSyncLogModel
	offset := (page - 1) * perPage
	if err := tx.Order("synced_at DESC").Offset(offset).Limit(perPage).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("syncLogRepo.ListAll: %w", err)
	}
	out := make([]channelsDomain.SyncLogEntry, len(rows))
	for i, row := range rows {
		out[i] = *toSyncLogDomain(row)
	}
	return out, total, nil
}

func stringMapToJSON(m map[string]string) string {
	if m == nil {
		m = map[string]string{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func stringMapFromJSON(raw string) map[string]string {
	out := map[string]string{}
	if raw == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func changesToJSON(m map[string]channelsDomain.FieldChange) string {
	if m == nil {
		m = map[string]channelsDomain.FieldChange{}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func changesFromJSON(raw string) map[string]channelsDomain.FieldChange {
	out := map[string]channelsDomain.FieldChange{}
	if raw == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func toSyncLogDomain(m channelSyncLogModel) *channelsDomain.SyncLogEntry {
	return &channelsDomain.SyncLogEntry{
		ID: m.ID, TenantID: m.TenantID, ListingID: m.ListingID, ProductID: m.ProductID, ConnectionID: m.ConnectionID,
		Provider: m.Provider, ExternalID: m.ExternalID,
		Direction: channelsDomain.SyncDirectionFlow(m.Direction), Status: channelsDomain.SyncLogStatus(m.Status),
		ErrorMessage: m.ErrorMessage,
		Fields:       stringMapFromJSON(m.Fields), Changes: changesFromJSON(m.Changes),
		SyncedAt: m.SyncedAt, DismissedAt: m.DismissedAt,
	}
}

// ── Sales channel repo ───────────────────────────────────────────────────────

type salesChannelModel struct {
	ID                   string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID             string `gorm:"not null;type:uuid"`
	Name                 string `gorm:"not null"`
	Type                 string `gorm:"not null"`
	ConnectionID         *string
	EnrollmentMode       string `gorm:"column:enrollment_mode;not null"`
	EnrollmentCategoryID *string
	EnrollmentTag        string `gorm:"not null;default:''"`
	Active               bool   `gorm:"not null;default:true"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (salesChannelModel) TableName() string { return "sales_channels" }

type SalesChannelRepo struct{ db *gorm.DB }

func NewSalesChannelRepo(db *gorm.DB) *SalesChannelRepo { return &SalesChannelRepo{db: db} }

func (r *SalesChannelRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("channels: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *SalesChannelRepo) Create(ctx context.Context, c *channelsDomain.SalesChannel) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := salesChannelModel{
		TenantID: c.TenantID, Name: c.Name, Type: string(c.Type), ConnectionID: c.ConnectionID,
		EnrollmentMode: string(c.EnrollmentMode), EnrollmentCategoryID: c.EnrollmentCategoryID, EnrollmentTag: c.EnrollmentTag,
		Active: c.Active, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("salesChannelRepo.Create: %w", err)
	}
	c.ID = m.ID
	return nil
}

func (r *SalesChannelRepo) FindByID(ctx context.Context, id string) (*channelsDomain.SalesChannel, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m salesChannelModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, channelsDomain.ErrSalesChannelNotFound
		}
		return nil, fmt.Errorf("salesChannelRepo.FindByID: %w", err)
	}
	return toSalesChannelDomain(m), nil
}

func (r *SalesChannelRepo) ListByTenant(ctx context.Context) ([]channelsDomain.SalesChannel, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []salesChannelModel
	if err := tx.Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("salesChannelRepo.ListByTenant: %w", err)
	}
	out := make([]channelsDomain.SalesChannel, len(rows))
	for i, row := range rows {
		out[i] = *toSalesChannelDomain(row)
	}
	return out, nil
}

func (r *SalesChannelRepo) Update(ctx context.Context, c *channelsDomain.SalesChannel) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&salesChannelModel{}).Where("id = ?", c.ID).
		Updates(map[string]any{
			"name": c.Name, "connection_id": c.ConnectionID,
			"enrollment_mode": string(c.EnrollmentMode), "enrollment_category_id": c.EnrollmentCategoryID, "enrollment_tag": c.EnrollmentTag,
			"active": c.Active, "updated_at": c.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("salesChannelRepo.Update: %w", err)
	}
	return nil
}

func (r *SalesChannelRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&salesChannelModel{}).Error; err != nil {
		return fmt.Errorf("salesChannelRepo.Delete: %w", err)
	}
	return nil
}

func toSalesChannelDomain(m salesChannelModel) *channelsDomain.SalesChannel {
	return &channelsDomain.SalesChannel{
		ID: m.ID, TenantID: m.TenantID, Name: m.Name, Type: channelsDomain.ChannelType(m.Type), ConnectionID: m.ConnectionID,
		EnrollmentMode: channelsDomain.EnrollmentMode(m.EnrollmentMode), EnrollmentCategoryID: m.EnrollmentCategoryID, EnrollmentTag: m.EnrollmentTag,
		Active: m.Active, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

// ── Sales channel product override repo ─────────────────────────────────────

type salesChannelProductOverrideModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string `gorm:"not null;type:uuid"`
	ChannelID string `gorm:"not null;type:uuid"`
	ProductID string `gorm:"not null;type:uuid"`
	Included  bool   `gorm:"not null"`
	CreatedAt time.Time
}

func (salesChannelProductOverrideModel) TableName() string { return "sales_channel_product_overrides" }

type SalesChannelProductOverrideRepo struct{ db *gorm.DB }

func NewSalesChannelProductOverrideRepo(db *gorm.DB) *SalesChannelProductOverrideRepo {
	return &SalesChannelProductOverrideRepo{db: db}
}

func (r *SalesChannelProductOverrideRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("channels: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Upsert matches the unique constraint on (channel_id, product_id) -- a
// repeat add/remove action overwrites the prior override.
func (r *SalesChannelProductOverrideRepo) Upsert(ctx context.Context, o *channelsDomain.SalesChannelProductOverride) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "channel_id"}, {Name: "product_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"included"}),
	}).Create(&salesChannelProductOverrideModel{
		TenantID: o.TenantID, ChannelID: o.ChannelID, ProductID: o.ProductID, Included: o.Included, CreatedAt: o.CreatedAt,
	}).Error
	if err != nil {
		return fmt.Errorf("salesChannelProductOverrideRepo.Upsert: %w", err)
	}
	return nil
}

func (r *SalesChannelProductOverrideRepo) FindByChannelAndProduct(ctx context.Context, channelID, productID string) (*channelsDomain.SalesChannelProductOverride, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m salesChannelProductOverrideModel
	if err := tx.Where("channel_id = ? AND product_id = ?", channelID, productID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("salesChannelProductOverrideRepo.FindByChannelAndProduct: %w", err)
	}
	return toSalesChannelProductOverrideDomain(m), nil
}

func (r *SalesChannelProductOverrideRepo) ListByChannel(ctx context.Context, channelID string) ([]channelsDomain.SalesChannelProductOverride, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []salesChannelProductOverrideModel
	if err := tx.Where("channel_id = ?", channelID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("salesChannelProductOverrideRepo.ListByChannel: %w", err)
	}
	out := make([]channelsDomain.SalesChannelProductOverride, len(rows))
	for i, row := range rows {
		out[i] = *toSalesChannelProductOverrideDomain(row)
	}
	return out, nil
}

func (r *SalesChannelProductOverrideRepo) Delete(ctx context.Context, channelID, productID string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("channel_id = ? AND product_id = ?", channelID, productID).Delete(&salesChannelProductOverrideModel{}).Error; err != nil {
		return fmt.Errorf("salesChannelProductOverrideRepo.Delete: %w", err)
	}
	return nil
}

func toSalesChannelProductOverrideDomain(m salesChannelProductOverrideModel) *channelsDomain.SalesChannelProductOverride {
	return &channelsDomain.SalesChannelProductOverride{
		ID: m.ID, TenantID: m.TenantID, ChannelID: m.ChannelID, ProductID: m.ProductID, Included: m.Included, CreatedAt: m.CreatedAt,
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
		VariantID:     m.VariantID,
		IsVariation:   m.IsVariation,
		SyncStatus:    channelsDomain.SyncStatus(m.SyncStatus),
		LastSyncedAt:  m.LastSyncedAt,
		LastSyncError: m.LastSyncError,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}
