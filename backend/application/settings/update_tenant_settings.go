package settings

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/domain/locale"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateTenantSettingsInput struct {
	DefaultCurrency                string
	Timezone                       string
	DateFormat                     string
	Language                       string
	PricesIncludeTax               bool
	DefaultTaxCountry              string
	InventorySyncRulesEnabled      bool
	BatchExpiryThresholdDays       int
	DefaultTargetMarkupBasisPoints int
	BundleConfidenceThreshold      float64
	RecommendationsMinSupport      int
	RecommendationsLookbackDays    int
	// PickingMode empty means the default, scan -- see settings.ParsePickingMode.
	PickingMode string
}

type UpdateTenantSettingsUseCase struct {
	db   *gorm.DB
	repo settingsDomain.Repository
}

func NewUpdateTenantSettingsUseCase(db *gorm.DB, repo settingsDomain.Repository) *UpdateTenantSettingsUseCase {
	return &UpdateTenantSettingsUseCase{db: db, repo: repo}
}

func (uc *UpdateTenantSettingsUseCase) Execute(ctx context.Context, in UpdateTenantSettingsInput) (*settingsDomain.TenantSettings, error) {
	if in.DefaultTargetMarkupBasisPoints < 0 {
		return nil, fmt.Errorf("updateTenantSettings: %w", settingsDomain.ErrInvalidTargetMarkupBasisPoints)
	}
	if in.BundleConfidenceThreshold < 0 || in.BundleConfidenceThreshold > 1 {
		return nil, fmt.Errorf("updateTenantSettings: %w", settingsDomain.ErrInvalidBundleConfidenceThreshold)
	}
	if in.RecommendationsMinSupport < 0 {
		return nil, fmt.Errorf("updateTenantSettings: %w", settingsDomain.ErrInvalidRecommendationsMinSupport)
	}
	if in.RecommendationsLookbackDays < 0 {
		return nil, fmt.Errorf("updateTenantSettings: %w", settingsDomain.ErrInvalidRecommendationsLookbackDays)
	}

	if err := settingsDomain.ValidateTimezone(in.Timezone); err != nil {
		return nil, fmt.Errorf("updateTenantSettings: %w", err)
	}

	pickingMode, err := settingsDomain.ParsePickingMode(in.PickingMode)
	if err != nil {
		return nil, fmt.Errorf("updateTenantSettings: %w", err)
	}
	dateFormat, err := settingsDomain.ParseDateFormat(in.DateFormat)
	if err != nil {
		return nil, fmt.Errorf("updateTenantSettings: %w", err)
	}

	language, err := locale.Resolve(locale.Locale(in.Language))
	if err != nil {
		return nil, fmt.Errorf("updateTenantSettings: %w", err)
	}

	var result settingsDomain.TenantSettings
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		result = settingsDomain.TenantSettings{
			TenantID:                       tenant.IDFromContext(ctx),
			DefaultCurrency:                in.DefaultCurrency,
			Timezone:                       in.Timezone,
			DateFormat:                     dateFormat,
			Language:                       language,
			PricesIncludeTax:               in.PricesIncludeTax,
			DefaultTaxCountry:              in.DefaultTaxCountry,
			InventorySyncRulesEnabled:      in.InventorySyncRulesEnabled,
			BatchExpiryThresholdDays:       in.BatchExpiryThresholdDays,
			DefaultTargetMarkupBasisPoints: in.DefaultTargetMarkupBasisPoints,
			BundleConfidenceThreshold:      in.BundleConfidenceThreshold,
			RecommendationsMinSupport:      in.RecommendationsMinSupport,
			RecommendationsLookbackDays:    in.RecommendationsLookbackDays,
			PickingMode:                    pickingMode,
			UpdatedAt:                      time.Now(),
		}
		return uc.repo.Upsert(ctx, &result)
	})
	if err != nil {
		return nil, fmt.Errorf("updateTenantSettings: %w", err)
	}
	return &result, nil
}
