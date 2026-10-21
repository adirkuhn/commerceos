package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type shippingPreferencesModel struct {
	TenantID            string  `gorm:"primaryKey;type:uuid"`
	DefaultConnectionID *string `gorm:"type:uuid"`
	UpdatedAt           time.Time
}

func (shippingPreferencesModel) TableName() string { return "shipping_preferences" }

type ShippingPreferencesRepo struct{ db *gorm.DB }

var _ shippingDomain.ShippingPreferencesRepository = (*ShippingPreferencesRepo)(nil)

func NewShippingPreferencesRepo(db *gorm.DB) *ShippingPreferencesRepo {
	return &ShippingPreferencesRepo{db: db}
}

func (r *ShippingPreferencesRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("shippingPreferencesRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ShippingPreferencesRepo) Get(ctx context.Context) (*shippingDomain.ShippingPreferences, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m shippingPreferencesModel
	err = tx.First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("shippingPreferencesRepo.Get: %w", err)
	}
	return &shippingDomain.ShippingPreferences{TenantID: m.TenantID, DefaultConnectionID: m.DefaultConnectionID, UpdatedAt: m.UpdatedAt}, nil
}

func (r *ShippingPreferencesRepo) Upsert(ctx context.Context, p *shippingDomain.ShippingPreferences) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := shippingPreferencesModel{TenantID: p.TenantID, DefaultConnectionID: p.DefaultConnectionID, UpdatedAt: p.UpdatedAt}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"default_connection_id", "updated_at"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("shippingPreferencesRepo.Upsert: %w", err)
	}
	return nil
}
