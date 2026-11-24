package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type appearanceModel struct {
	TenantID    string    `gorm:"column:tenant_id;primaryKey;type:uuid"`
	DefaultLook string    `gorm:"column:default_look"`
	Corners     string    `gorm:"column:corners"`
	TextSize    string    `gorm:"column:text_size"`
	Density     string    `gorm:"column:density"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

// The retired font/accent_color/sidebar_color columns are left to their
// defaults (migration 254) until a later migration drops them.
func (appearanceModel) TableName() string { return "tenant_appearance" }

// AppearanceRepo implements settings.AppearanceRepository.
type AppearanceRepo struct{ db *gorm.DB }

func NewAppearanceRepo(db *gorm.DB) *AppearanceRepo { return &AppearanceRepo{db: db} }

func (r *AppearanceRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("appearanceRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Get returns the tenant's saved appearance, or nil if none is saved yet --
// RLS already scopes this to at most one row.
func (r *AppearanceRepo) Get(ctx context.Context) (*settingsDomain.Appearance, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []appearanceModel
	if err := tx.Limit(1).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("appearanceRepo.Get: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	m := rows[0]
	return &settingsDomain.Appearance{
		TenantID: m.TenantID, DefaultLook: settingsDomain.AppearanceLook(m.DefaultLook),
		Corners: settingsDomain.AppearanceCorners(m.Corners), TextSize: settingsDomain.AppearanceTextSize(m.TextSize),
		Density: settingsDomain.AppearanceDensity(m.Density), UpdatedAt: m.UpdatedAt,
	}, nil
}

func (r *AppearanceRepo) Upsert(ctx context.Context, a *settingsDomain.Appearance) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := appearanceModel{
		TenantID: a.TenantID, DefaultLook: string(a.DefaultLook),
		Corners: string(a.Corners), TextSize: string(a.TextSize), Density: string(a.Density), UpdatedAt: a.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"default_look", "corners", "text_size", "density", "updated_at"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("appearanceRepo.Upsert: %w", err)
	}
	return nil
}
