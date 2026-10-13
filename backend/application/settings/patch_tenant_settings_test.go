package settings_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsettings "github.com/adirkuhn/commerceos/application/settings"
	"github.com/adirkuhn/commerceos/domain/locale"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

type stubSettingsGetter struct {
	out *settingsDomain.TenantSettings
	err error
}

func (s stubSettingsGetter) Execute(context.Context) (*settingsDomain.TenantSettings, error) {
	return s.out, s.err
}

type recordingUpdater struct {
	got *appsettings.UpdateTenantSettingsInput
}

func (r *recordingUpdater) Execute(_ context.Context, in appsettings.UpdateTenantSettingsInput) (*settingsDomain.TenantSettings, error) {
	r.got = &in
	return &settingsDomain.TenantSettings{}, nil
}

func storedSettings() *settingsDomain.TenantSettings {
	return &settingsDomain.TenantSettings{
		DefaultCurrency: "EUR", Timezone: "Europe/Amsterdam", DateFormat: settingsDomain.DateFormatDayMonthYear, Language: locale.Portuguese,
		PricesIncludeTax: true, DefaultTaxCountry: "NL", InventorySyncRulesEnabled: true, BatchExpiryThresholdDays: 3,
		DefaultTargetMarkupBasisPoints: 4000, BundleConfidenceThreshold: 0.5, RecommendationsMinSupport: 2, RecommendationsLookbackDays: 90,
		PickingMode: settingsDomain.PickingModeList, ExpiryAlert: settingsDomain.ExpiryAlert{Enabled: true, Windows: []int{30, 7}},
		LowStockThreshold: 15,
	}
}

func asFullInput(s *settingsDomain.TenantSettings) appsettings.UpdateTenantSettingsInput {
	return appsettings.UpdateTenantSettingsInput{
		DefaultCurrency: s.DefaultCurrency, Timezone: s.Timezone, DateFormat: string(s.DateFormat), Language: string(s.Language),
		PricesIncludeTax: s.PricesIncludeTax, DefaultTaxCountry: s.DefaultTaxCountry, InventorySyncRulesEnabled: s.InventorySyncRulesEnabled,
		BatchExpiryThresholdDays: s.BatchExpiryThresholdDays, DefaultTargetMarkupBasisPoints: s.DefaultTargetMarkupBasisPoints,
		BundleConfidenceThreshold: s.BundleConfidenceThreshold, RecommendationsMinSupport: s.RecommendationsMinSupport,
		RecommendationsLookbackDays: s.RecommendationsLookbackDays, PickingMode: string(s.PickingMode),
		ExpiryAlertEnabled: s.ExpiryAlert.Enabled, ExpiryAlertWindows: s.ExpiryAlert.Windows,
		LowStockThreshold: s.LowStockThreshold,
	}
}

// The bug this fixes: saving one settings section sent every other field
// from a stale copy, switching e.g. the expiry report back off.
func TestPatchTenantSettings_omittedFieldsKeepTheirStoredValue(t *testing.T) {
	updater := &recordingUpdater{}
	uc := appsettings.NewPatchTenantSettingsUseCase(stubSettingsGetter{out: storedSettings()}, updater)

	syncOff, threshold := false, 5
	_, err := uc.Execute(context.Background(), appsettings.PatchTenantSettingsInput{InventorySyncRulesEnabled: &syncOff, BatchExpiryThresholdDays: &threshold})

	require.NoError(t, err)
	want := asFullInput(storedSettings())
	want.InventorySyncRulesEnabled, want.BatchExpiryThresholdDays = false, 5
	assert.Equal(t, want, *updater.got)
}

func TestPatchTenantSettings_everyFieldCanBeSet(t *testing.T) {
	updater := &recordingUpdater{}
	uc := appsettings.NewPatchTenantSettingsUseCase(stubSettingsGetter{out: &settingsDomain.TenantSettings{}}, updater)
	full := asFullInput(storedSettings())

	_, err := uc.Execute(context.Background(), appsettings.PatchTenantSettingsInput{
		DefaultCurrency: &full.DefaultCurrency, Timezone: &full.Timezone, DateFormat: &full.DateFormat, Language: &full.Language,
		PricesIncludeTax: &full.PricesIncludeTax, DefaultTaxCountry: &full.DefaultTaxCountry, InventorySyncRulesEnabled: &full.InventorySyncRulesEnabled,
		BatchExpiryThresholdDays: &full.BatchExpiryThresholdDays, DefaultTargetMarkupBasisPoints: &full.DefaultTargetMarkupBasisPoints,
		BundleConfidenceThreshold: &full.BundleConfidenceThreshold, RecommendationsMinSupport: &full.RecommendationsMinSupport,
		RecommendationsLookbackDays: &full.RecommendationsLookbackDays, PickingMode: &full.PickingMode,
		ExpiryAlertEnabled: &full.ExpiryAlertEnabled, ExpiryAlertWindows: &full.ExpiryAlertWindows,
		LowStockThreshold: &full.LowStockThreshold,
	})

	require.NoError(t, err)
	assert.Equal(t, full, *updater.got)
}

func TestPatchTenantSettings_readError(t *testing.T) {
	boom := errors.New("boom")
	updater := &recordingUpdater{}
	_, err := appsettings.NewPatchTenantSettingsUseCase(stubSettingsGetter{err: boom}, updater).Execute(context.Background(), appsettings.PatchTenantSettingsInput{})

	assert.ErrorIs(t, err, boom)
	assert.Nil(t, updater.got)
}
