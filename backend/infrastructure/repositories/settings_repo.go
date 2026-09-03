package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type tenantSettingsModel struct {
	TenantID                       string `gorm:"primaryKey;type:uuid"`
	DefaultCurrency                string `gorm:"not null;default:''"`
	Timezone                       string `gorm:"not null;default:''"`
	DateFormat                     string `gorm:"column:date_format;not null;default:''"`
	Language                       string `gorm:"not null;default:''"`
	PricesIncludeTax               bool   `gorm:"not null;default:true"`
	DefaultTaxCountry              string `gorm:"not null;default:''"`
	InventorySyncRulesEnabled      bool   `gorm:"not null;default:false"`
	BatchExpiryThresholdDays       int    `gorm:"not null;default:0"`
	DefaultTargetMarkupBasisPoints int    `gorm:"column:default_target_markup_basis_points;not null;default:0"`
	// StorefrontKeyHash is deliberately not part of settingsDomain.
	// TenantSettings -- see domain/settings.Repository's doc comment on
	// GetStorefrontKeyHash. Nullable: most tenants never generate one.
	StorefrontKeyHash *string
	UpdatedAt         time.Time
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
	return toTenantSettingsDomain(m), nil
}

func (r *TenantSettingsRepo) Upsert(ctx context.Context, s *settingsDomain.TenantSettings) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := tenantSettingsModel{
		TenantID: s.TenantID, DefaultCurrency: s.DefaultCurrency, Timezone: s.Timezone,
		DateFormat: s.DateFormat, Language: s.Language, PricesIncludeTax: s.PricesIncludeTax,
		DefaultTaxCountry: s.DefaultTaxCountry, InventorySyncRulesEnabled: s.InventorySyncRulesEnabled,
		BatchExpiryThresholdDays:       s.BatchExpiryThresholdDays,
		DefaultTargetMarkupBasisPoints: s.DefaultTargetMarkupBasisPoints, UpdatedAt: s.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"default_currency", "timezone", "date_format", "language",
			"prices_include_tax", "default_tax_country",
			"inventory_sync_rules_enabled", "batch_expiry_threshold_days",
			"default_target_markup_basis_points", "updated_at",
		}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("tenantSettingsRepo.Upsert: %w", err)
	}
	return nil
}

// GetStorefrontKeyHash returns "" when no key has been generated yet for
// the current tenant.
func (r *TenantSettingsRepo) GetStorefrontKeyHash(ctx context.Context) (string, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return "", err
	}
	var m tenantSettingsModel
	if err := tx.Limit(1).Find(&m).Error; err != nil {
		return "", fmt.Errorf("tenantSettingsRepo.GetStorefrontKeyHash: %w", err)
	}
	return stringFromPtr(m.StorefrontKeyHash), nil
}

// SetStorefrontKeyHash upserts just the one column -- unlike Upsert, which
// replaces every localization setting, this must never touch currency/
// timezone/etc for a tenant that already has a settings row.
func (r *TenantSettingsRepo) SetStorefrontKeyHash(ctx context.Context, hash string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := tenantSettingsModel{TenantID: tenant.IDFromContext(ctx), StorefrontKeyHash: stringPtrOrNil(hash), UpdatedAt: time.Now()}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"storefront_key_hash", "updated_at"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("tenantSettingsRepo.SetStorefrontKeyHash: %w", err)
	}
	return nil
}

// FindTenantIDByStorefrontKeyHash looks up a hash across every tenant --
// the caller (see application/integrations.ResolveStorefrontKeyUseCase) has
// no tenant to scope by yet, that's exactly what this resolves.
func (r *TenantSettingsRepo) FindTenantIDByStorefrontKeyHash(ctx context.Context, hash string) (string, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return "", err
	}
	var m tenantSettingsModel
	if err := tx.Where("storefront_key_hash = ?", hash).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", settingsDomain.ErrStorefrontKeyNotFound
		}
		return "", fmt.Errorf("tenantSettingsRepo.FindTenantIDByStorefrontKeyHash: %w", err)
	}
	return m.TenantID, nil
}

// stringPtrOrNil/stringFromPtr convert between the domain's "" -> unset
// convention and the model's nullable TEXT column.
func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func stringFromPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toTenantSettingsDomain(m tenantSettingsModel) *settingsDomain.TenantSettings {
	return &settingsDomain.TenantSettings{
		TenantID: m.TenantID, DefaultCurrency: m.DefaultCurrency, Timezone: m.Timezone,
		DateFormat: m.DateFormat, Language: m.Language, PricesIncludeTax: m.PricesIncludeTax,
		DefaultTaxCountry: m.DefaultTaxCountry, InventorySyncRulesEnabled: m.InventorySyncRulesEnabled,
		BatchExpiryThresholdDays:       m.BatchExpiryThresholdDays,
		DefaultTargetMarkupBasisPoints: m.DefaultTargetMarkupBasisPoints, UpdatedAt: m.UpdatedAt,
	}
}
