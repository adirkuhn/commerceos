package settings

import (
	"context"
	"fmt"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

// PatchTenantSettingsInput is a partial settings update: nil keeps the
// stored value.
type PatchTenantSettingsInput struct {
	DefaultCurrency                *string
	Timezone                       *string
	DateFormat                     *string
	Language                       *string
	PricesIncludeTax               *bool
	DefaultTaxCountry              *string
	InventorySyncRulesEnabled      *bool
	BatchExpiryThresholdDays       *int
	DefaultTargetMarkupBasisPoints *int
	BundleConfidenceThreshold      *float64
	RecommendationsMinSupport      *int
	RecommendationsLookbackDays    *int
	PickingMode                    *string
	ExpiryAlertEnabled             *bool
	ExpiryAlertWindows             *[]int
}

type tenantSettingsReader interface {
	Execute(ctx context.Context) (*settingsDomain.TenantSettings, error)
}

type tenantSettingsUpdater interface {
	Execute(ctx context.Context, in UpdateTenantSettingsInput) (*settingsDomain.TenantSettings, error)
}

// PatchTenantSettingsUseCase applies a partial update: fields left out keep
// what is stored, so one settings section saving its own fields can't
// overwrite another section's change with a stale copy. Validation, saving
// and the event stay UpdateTenantSettingsUseCase's.
type PatchTenantSettingsUseCase struct {
	current tenantSettingsReader
	update  tenantSettingsUpdater
}

func NewPatchTenantSettingsUseCase(current tenantSettingsReader, update tenantSettingsUpdater) *PatchTenantSettingsUseCase {
	return &PatchTenantSettingsUseCase{current: current, update: update}
}

func (uc *PatchTenantSettingsUseCase) Execute(ctx context.Context, in PatchTenantSettingsInput) (*settingsDomain.TenantSettings, error) {
	s, err := uc.current.Execute(ctx)
	if err != nil {
		return nil, fmt.Errorf("patchTenantSettings: %w", err)
	}
	full := UpdateTenantSettingsInput{
		DefaultCurrency:                or(in.DefaultCurrency, s.DefaultCurrency),
		Timezone:                       or(in.Timezone, s.Timezone),
		DateFormat:                     or(in.DateFormat, string(s.DateFormat)),
		Language:                       or(in.Language, string(s.Language)),
		PricesIncludeTax:               or(in.PricesIncludeTax, s.PricesIncludeTax),
		DefaultTaxCountry:              or(in.DefaultTaxCountry, s.DefaultTaxCountry),
		InventorySyncRulesEnabled:      or(in.InventorySyncRulesEnabled, s.InventorySyncRulesEnabled),
		BatchExpiryThresholdDays:       or(in.BatchExpiryThresholdDays, s.BatchExpiryThresholdDays),
		DefaultTargetMarkupBasisPoints: or(in.DefaultTargetMarkupBasisPoints, s.DefaultTargetMarkupBasisPoints),
		BundleConfidenceThreshold:      or(in.BundleConfidenceThreshold, s.BundleConfidenceThreshold),
		RecommendationsMinSupport:      or(in.RecommendationsMinSupport, s.RecommendationsMinSupport),
		RecommendationsLookbackDays:    or(in.RecommendationsLookbackDays, s.RecommendationsLookbackDays),
		PickingMode:                    or(in.PickingMode, string(s.PickingMode)),
		ExpiryAlertEnabled:             or(in.ExpiryAlertEnabled, s.ExpiryAlert.Enabled),
		ExpiryAlertWindows:             or(in.ExpiryAlertWindows, s.ExpiryAlert.Windows),
	}
	return uc.update.Execute(ctx, full)
}

func or[T any](v *T, fallback T) T {
	if v != nil {
		return *v
	}
	return fallback
}
