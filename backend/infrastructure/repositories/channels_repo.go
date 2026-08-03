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
