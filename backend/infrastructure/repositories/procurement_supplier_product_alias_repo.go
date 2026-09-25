package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── SupplierProductAliasRepo ─────────────────────────────────────────────────

type supplierProductAliasModel struct {
	ID                    string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID              string `gorm:"not null;type:uuid"`
	SupplierID            string `gorm:"column:supplier_id;not null;type:uuid"`
	RawKey                string `gorm:"column:raw_key;not null"`
	SKUID                 string `gorm:"column:sku_id;not null;type:uuid"`
	UnitsPerPurchasedItem int64  `gorm:"column:units_per_purchased_item;not null;default:1"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (supplierProductAliasModel) TableName() string { return "supplier_product_aliases" }

type SupplierProductAliasRepo struct{ db *gorm.DB }

func NewSupplierProductAliasRepo(db *gorm.DB) *SupplierProductAliasRepo {
	return &SupplierProductAliasRepo{db: db}
}

func (r *SupplierProductAliasRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("procurement: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Upsert writes or updates the (tenant, supplier, raw_key) alias -- relies
// on the supplier_product_aliases_unique constraint from migration 027.
func (r *SupplierProductAliasRepo) Upsert(ctx context.Context, alias *procurementDomain.SupplierProductAlias) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := supplierProductAliasModel{
		TenantID: alias.TenantID, SupplierID: alias.SupplierID, RawKey: alias.RawKey, SKUID: alias.SKUID,
		UnitsPerPurchasedItem: alias.UnitsPerPurchasedItem,
		CreatedAt:             alias.CreatedAt, UpdatedAt: alias.UpdatedAt,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "supplier_id"}, {Name: "raw_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"sku_id", "units_per_purchased_item", "updated_at"}),
	}).Create(&m).Error; err != nil {
		return fmt.Errorf("supplierProductAliasRepo.Upsert: %w", err)
	}
	alias.ID = m.ID
	return nil
}

func (r *SupplierProductAliasRepo) FindBySupplierAndRawKey(ctx context.Context, supplierID, rawKey string) (*procurementDomain.SupplierProductAlias, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m supplierProductAliasModel
	if err := tx.Where("supplier_id = ? AND raw_key = ?", supplierID, rawKey).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("supplierProductAliasRepo.FindBySupplierAndRawKey: %w", err)
	}
	return toAliasDomain(m), nil
}

func toAliasDomain(m supplierProductAliasModel) *procurementDomain.SupplierProductAlias {
	return &procurementDomain.SupplierProductAlias{
		ID: m.ID, TenantID: m.TenantID, SupplierID: m.SupplierID, RawKey: m.RawKey, SKUID: m.SKUID,
		UnitsPerPurchasedItem: m.UnitsPerPurchasedItem,
		CreatedAt:             m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
