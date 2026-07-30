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
