package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

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
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		log.Printf("channelsRepo: corrupt stored JSON, treating as empty: %v", err)
	}
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
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		log.Printf("channelsRepo: corrupt stored JSON, treating as empty: %v", err)
	}
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
