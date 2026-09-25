package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── ExternalCategoryLinkRepo ─────────────────────────────────────────────

type externalCategoryLinkModel struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string `gorm:"not null;type:uuid"`
	ConnectionID string `gorm:"not null;type:uuid"`
	Provider     string `gorm:"not null"`
	ExternalID   string `gorm:"not null"`
	CategoryID   string `gorm:"not null;type:uuid"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (externalCategoryLinkModel) TableName() string { return "external_category_links" }

type ExternalCategoryLinkRepo struct{ db *gorm.DB }

func NewExternalCategoryLinkRepo(db *gorm.DB) *ExternalCategoryLinkRepo {
	return &ExternalCategoryLinkRepo{db: db}
}

func (r *ExternalCategoryLinkRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("channels: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Upsert inserts or updates by the (connection_id, external_id) unique
// constraint (see migration 090). A repeat auto-match or manual repoint of
// the same Woo category overwrites the prior link.
func (r *ExternalCategoryLinkRepo) Upsert(ctx context.Context, l *channelsDomain.ExternalCategoryLink) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := externalCategoryLinkModel{
		TenantID: l.TenantID, ConnectionID: l.ConnectionID, Provider: l.Provider,
		ExternalID: l.ExternalID, CategoryID: l.CategoryID, CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "connection_id"}, {Name: "external_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"category_id", "updated_at"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("externalCategoryLinkRepo.Upsert: %w", err)
	}
	l.ID = m.ID
	return nil
}

func (r *ExternalCategoryLinkRepo) FindByConnectionAndExternalID(ctx context.Context, connectionID, externalID string) (*channelsDomain.ExternalCategoryLink, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m externalCategoryLinkModel
	if err := tx.Where("connection_id = ? AND external_id = ?", connectionID, externalID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, channelsDomain.ErrExternalCategoryLinkNotFound
		}
		return nil, fmt.Errorf("externalCategoryLinkRepo.FindByConnectionAndExternalID: %w", err)
	}
	return toExternalCategoryLinkDomain(m), nil
}

func (r *ExternalCategoryLinkRepo) FindByConnectionAndCategory(ctx context.Context, connectionID, categoryID string) (*channelsDomain.ExternalCategoryLink, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m externalCategoryLinkModel
	if err := tx.Where("connection_id = ? AND category_id = ?", connectionID, categoryID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, channelsDomain.ErrExternalCategoryLinkNotFound
		}
		return nil, fmt.Errorf("externalCategoryLinkRepo.FindByConnectionAndCategory: %w", err)
	}
	return toExternalCategoryLinkDomain(m), nil
}

func (r *ExternalCategoryLinkRepo) ListByConnection(ctx context.Context, connectionID string) ([]channelsDomain.ExternalCategoryLink, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []externalCategoryLinkModel
	if err := tx.Where("connection_id = ?", connectionID).Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("externalCategoryLinkRepo.ListByConnection: %w", err)
	}
	links := make([]channelsDomain.ExternalCategoryLink, len(rows))
	for i, row := range rows {
		links[i] = *toExternalCategoryLinkDomain(row)
	}
	return links, nil
}

func (r *ExternalCategoryLinkRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&externalCategoryLinkModel{}).Error; err != nil {
		return fmt.Errorf("externalCategoryLinkRepo.Delete: %w", err)
	}
	return nil
}

func toExternalCategoryLinkDomain(m externalCategoryLinkModel) *channelsDomain.ExternalCategoryLink {
	return &channelsDomain.ExternalCategoryLink{
		ID: m.ID, TenantID: m.TenantID, ConnectionID: m.ConnectionID, Provider: m.Provider,
		ExternalID: m.ExternalID, CategoryID: m.CategoryID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
