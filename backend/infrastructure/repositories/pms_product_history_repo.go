package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type productHistoryModel struct {
	ID         string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID   string `gorm:"not null;type:uuid"`
	EntityType string `gorm:"column:entity_type;not null"`
	EntityID   string `gorm:"column:entity_id;not null;type:uuid"`
	Field      string `gorm:"not null"`
	OldValue   string `gorm:"column:old_value;not null"`
	NewValue   string `gorm:"column:new_value;not null"`
	ChangedAt  time.Time
}

func (productHistoryModel) TableName() string { return "product_history" }

type ProductHistoryRepo struct{ db *gorm.DB }

func NewProductHistoryRepo(db *gorm.DB) *ProductHistoryRepo {
	return &ProductHistoryRepo{db: db}
}

func (r *ProductHistoryRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("pms: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ProductHistoryRepo) Create(ctx context.Context, h *pmsDomain.ProductHistory) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := productHistoryModel{
		TenantID: h.TenantID, EntityType: string(h.EntityType), EntityID: h.EntityID,
		Field: string(h.Field), OldValue: h.OldValue, NewValue: h.NewValue, ChangedAt: h.ChangedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("productHistoryRepo.Create: %w", err)
	}
	h.ID = m.ID
	return nil
}

func (r *ProductHistoryRepo) ListByEntity(ctx context.Context, entityType pmsDomain.ProductHistoryEntityType, entityID string) ([]pmsDomain.ProductHistory, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []productHistoryModel
	if err := tx.Where("entity_type = ? AND entity_id = ?", string(entityType), entityID).Order("changed_at desc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("productHistoryRepo.ListByEntity: %w", err)
	}
	history := make([]pmsDomain.ProductHistory, len(rows))
	for i, row := range rows {
		history[i] = pmsDomain.ProductHistory{
			ID: row.ID, TenantID: row.TenantID, EntityType: pmsDomain.ProductHistoryEntityType(row.EntityType),
			EntityID: row.EntityID, Field: pmsDomain.ProductHistoryField(row.Field),
			OldValue: row.OldValue, NewValue: row.NewValue, ChangedAt: row.ChangedAt,
		}
	}
	return history, nil
}
