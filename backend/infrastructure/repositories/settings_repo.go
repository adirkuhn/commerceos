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

type tenantSettingsModel struct {
	TenantID                       string  `gorm:"primaryKey;type:uuid"`
	DefaultCurrency                string  `gorm:"not null;default:''"`
	Timezone                       string  `gorm:"not null;default:''"`
	DateFormat                     string  `gorm:"column:date_format;not null;default:''"`
	Language                       string  `gorm:"not null;default:''"`
	PricesIncludeTax               bool    `gorm:"not null;default:true"`
	DefaultTaxCountry              string  `gorm:"not null;default:''"`
	InventorySyncRulesEnabled      bool    `gorm:"not null;default:false"`
	BatchExpiryThresholdDays       int     `gorm:"not null;default:0"`
	DefaultTargetMarkupBasisPoints int     `gorm:"column:default_target_markup_basis_points;not null;default:0"`
	BundleConfidenceThreshold      float64 `gorm:"column:bundle_confidence_threshold;not null;default:0.30"`
	RecommendationsMinSupport      int     `gorm:"column:recommendations_min_support;not null;default:2"`
	RecommendationsLookbackDays    int     `gorm:"column:recommendations_lookback_days;not null;default:180"`
	PickingMode                    string  `gorm:"column:picking_mode;not null;default:scan"`
	UpdatedAt                      time.Time
}

func (tenantSettingsModel) TableName() string { return "tenant_settings" }

type TenantSettingsRepo struct{ db *gorm.DB }

func NewTenantSettingsRepo(db *gorm.DB) *TenantSettingsRepo { return &TenantSettingsRepo{db: db} }

func (r *TenantSettingsRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("settings: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Get returns the tenant's settings row, or a zero-value TenantSettings if
// none has been saved yet — RLS already scopes this to at most one row.
func (r *TenantSettingsRepo) Get(ctx context.Context) (*settingsDomain.TenantSettings, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m tenantSettingsModel
	if err := tx.Limit(1).Find(&m).Error; err != nil {
		return nil, fmt.Errorf("tenantSettingsRepo.Get: %w", err)
	}
	return toTenantSettingsDomain(m)
}

func (r *TenantSettingsRepo) Upsert(ctx context.Context, s *settingsDomain.TenantSettings) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	pickingMode, err := settingsDomain.ParsePickingMode(string(s.PickingMode))
	if err != nil {
		return fmt.Errorf("tenantSettingsRepo.Upsert: %w", err)
	}
	m := tenantSettingsModel{
		TenantID: s.TenantID, DefaultCurrency: s.DefaultCurrency, Timezone: s.Timezone,
		DateFormat: s.DateFormat, Language: s.Language, PricesIncludeTax: s.PricesIncludeTax,
		DefaultTaxCountry: s.DefaultTaxCountry, InventorySyncRulesEnabled: s.InventorySyncRulesEnabled,
		BatchExpiryThresholdDays:       s.BatchExpiryThresholdDays,
		DefaultTargetMarkupBasisPoints: s.DefaultTargetMarkupBasisPoints,
		BundleConfidenceThreshold:      s.BundleConfidenceThreshold,
		RecommendationsMinSupport:      s.RecommendationsMinSupport,
		RecommendationsLookbackDays:    s.RecommendationsLookbackDays, PickingMode: string(pickingMode), UpdatedAt: s.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"default_currency", "timezone", "date_format", "language",
			"prices_include_tax", "default_tax_country",
			"inventory_sync_rules_enabled", "batch_expiry_threshold_days",
			"default_target_markup_basis_points", "bundle_confidence_threshold",
			"recommendations_min_support", "recommendations_lookback_days", "picking_mode", "updated_at",
		}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("tenantSettingsRepo.Upsert: %w", err)
	}
	return nil
}

// toTenantSettingsDomain maps a row -- including the zero row a never-saved
// tenant reads as, whose empty picking_mode parses to the default.
func toTenantSettingsDomain(m tenantSettingsModel) (*settingsDomain.TenantSettings, error) {
	pickingMode, err := settingsDomain.ParsePickingMode(m.PickingMode)
	if err != nil {
		return nil, fmt.Errorf("tenantSettingsRepo: picking_mode %q: %w", m.PickingMode, err)
	}
	return &settingsDomain.TenantSettings{
		TenantID: m.TenantID, DefaultCurrency: m.DefaultCurrency, Timezone: m.Timezone,
		DateFormat: m.DateFormat, Language: m.Language, PricesIncludeTax: m.PricesIncludeTax,
		DefaultTaxCountry: m.DefaultTaxCountry, InventorySyncRulesEnabled: m.InventorySyncRulesEnabled,
		BatchExpiryThresholdDays:       m.BatchExpiryThresholdDays,
		DefaultTargetMarkupBasisPoints: m.DefaultTargetMarkupBasisPoints,
		BundleConfidenceThreshold:      m.BundleConfidenceThreshold,
		RecommendationsMinSupport:      m.RecommendationsMinSupport,
		RecommendationsLookbackDays:    m.RecommendationsLookbackDays, PickingMode: pickingMode, UpdatedAt: m.UpdatedAt,
	}, nil
}
