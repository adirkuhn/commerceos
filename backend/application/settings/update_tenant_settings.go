package settings

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
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
	// ExpiryAlertEnabled / ExpiryAlertWindows -- see settings.NewExpiryAlert.
	ExpiryAlertEnabled bool
	ExpiryAlertWindows []int
}

type UpdateTenantSettingsUseCase struct {
	db   *gorm.DB
	repo   settingsDomain.Repository
	outbox eventrecord.Enqueuer
}

func NewUpdateTenantSettingsUseCase(db *gorm.DB, repo settingsDomain.Repository, outbox eventrecord.Enqueuer) *UpdateTenantSettingsUseCase {
	return &UpdateTenantSettingsUseCase{db: db, repo: repo, outbox: outbox}
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

	expiryAlert, err := settingsDomain.NewExpiryAlert(in.ExpiryAlertEnabled, in.ExpiryAlertWindows)
	if err != nil {
		return nil, fmt.Errorf("updateTenantSettings: %w", err)
	}

	var result settingsDomain.TenantSettings
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		before, err := uc.repo.Get(ctx)
		if err != nil {
			return err
		}
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
			ExpiryAlert:                    expiryAlert,
			UpdatedAt:                      time.Now(),
		}
		if err := uc.repo.Upsert(ctx, &result); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.TenantSettingsUpdated, result.TenantID, tenantSettingsChanges(*before, result))
	})
	if err != nil {
		return nil, fmt.Errorf("updateTenantSettings: %w", err)
	}
	return &result, nil
}

func tenantSettingsChanges(before, after settingsDomain.TenantSettings) eventrecord.Changes {
	return eventrecord.NewChanges().
		Set("default_currency", before.DefaultCurrency, after.DefaultCurrency).
		Set("timezone", before.Timezone, after.Timezone).
		Set("date_format", string(before.DateFormat), string(after.DateFormat)).
		Set("language", string(before.Language), string(after.Language)).
		Set("prices_include_tax", before.PricesIncludeTax, after.PricesIncludeTax).
		Set("default_tax_country", before.DefaultTaxCountry, after.DefaultTaxCountry).
		Set("inventory_sync_rules_enabled", before.InventorySyncRulesEnabled, after.InventorySyncRulesEnabled).
		Set("batch_expiry_threshold_days", before.BatchExpiryThresholdDays, after.BatchExpiryThresholdDays).
		Set("default_target_markup_basis_points", before.DefaultTargetMarkupBasisPoints, after.DefaultTargetMarkupBasisPoints).
		Set("bundle_confidence_threshold", before.BundleConfidenceThreshold, after.BundleConfidenceThreshold).
		Set("recommendations_min_support", before.RecommendationsMinSupport, after.RecommendationsMinSupport).
		Set("recommendations_lookback_days", before.RecommendationsLookbackDays, after.RecommendationsLookbackDays).
		Set("picking_mode", string(before.PickingMode), string(after.PickingMode)).
		Set("expiry_alert_enabled", before.ExpiryAlert.Enabled, after.ExpiryAlert.Enabled).
		Set("expiry_alert_windows", nonNilInts(before.ExpiryAlert.Windows), nonNilInts(after.ExpiryAlert.Windows))
}

// nonNilInts makes a never-saved tenant's nil window list compare equal to an
// explicitly empty one, so saving "no windows" over "unset" is not a change.
func nonNilInts(v []int) []int {
	if v == nil {
		return []int{}
	}
	return v
}
