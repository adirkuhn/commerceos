package settings_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appsettings "github.com/adirkuhn/commerceos/application/settings"
	"github.com/adirkuhn/commerceos/domain/locale"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestUpdateTenantSettings_upsertsAllFields(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx, appsettings.UpdateTenantSettingsInput{
		DefaultCurrency: "USD", Timezone: "Europe/Amsterdam", DateFormat: "DD/MM/YYYY", Language: "en",
		PricesIncludeTax: true, DefaultTaxCountry: "NL",
		InventorySyncRulesEnabled: true, BatchExpiryThresholdDays: 30,
		DefaultTargetMarkupBasisPoints: 3000,
		BundleConfidenceThreshold:      0.25,
		RecommendationsMinSupport:      3,
		RecommendationsLookbackDays:    90,
	})

	require.NoError(t, err)
	require.Equal(t, 3000, result.DefaultTargetMarkupBasisPoints)
	require.InDelta(t, 0.25, result.BundleConfidenceThreshold, 0.0001)
	require.Equal(t, 3, result.RecommendationsMinSupport)
	require.Equal(t, 90, result.RecommendationsLookbackDays)
	require.Equal(t, "USD", result.DefaultCurrency)
	require.Equal(t, "Europe/Amsterdam", result.Timezone)
	require.Equal(t, settingsDomain.DateFormatDayMonthYear, result.DateFormat)
	require.Equal(t, locale.English, result.Language)
	require.True(t, result.PricesIncludeTax)
	require.Equal(t, "NL", result.DefaultTaxCountry)
	require.True(t, result.InventorySyncRulesEnabled)
	require.Equal(t, 30, result.BatchExpiryThresholdDays)
	require.Equal(t, "tenant-1", result.TenantID)
	require.Len(t, repo.upserted, 1)
	require.Equal(t, "USD", repo.upserted[0].DefaultCurrency)
	require.True(t, repo.upserted[0].PricesIncludeTax)
	require.Equal(t, "NL", repo.upserted[0].DefaultTaxCountry)
	require.True(t, repo.upserted[0].InventorySyncRulesEnabled)
	require.Equal(t, 30, repo.upserted[0].BatchExpiryThresholdDays)
	require.InDelta(t, 0.25, repo.upserted[0].BundleConfidenceThreshold, 0.0001)
	require.Equal(t, 3, repo.upserted[0].RecommendationsMinSupport)
	require.Equal(t, 90, repo.upserted[0].RecommendationsLookbackDays)
}

func TestUpdateTenantSettings_propagatesRepoError(t *testing.T) {
	repo := &stubSettingsRepo{upsertErr: errBoom}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD"})

	require.Error(t, err)
}

func TestUpdateTenantSettings_rejectsNegativeTargetMarkup(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", DefaultTargetMarkupBasisPoints: -1})

	require.ErrorIs(t, err, settingsDomain.ErrInvalidTargetMarkupBasisPoints)
}

func TestUpdateTenantSettings_rejectsBundleConfidenceThresholdBelowZero(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", BundleConfidenceThreshold: -0.01})

	require.ErrorIs(t, err, settingsDomain.ErrInvalidBundleConfidenceThreshold)
}

func TestUpdateTenantSettings_rejectsBundleConfidenceThresholdAboveOne(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", BundleConfidenceThreshold: 1.01})

	require.ErrorIs(t, err, settingsDomain.ErrInvalidBundleConfidenceThreshold)
}

func TestUpdateTenantSettings_allowsBundleConfidenceThresholdOfZero(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", BundleConfidenceThreshold: 0})

	require.NoError(t, err)
	require.Equal(t, 0.0, result.BundleConfidenceThreshold)
}

func TestUpdateTenantSettings_allowsBundleConfidenceThresholdOfOne(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", BundleConfidenceThreshold: 1})

	require.NoError(t, err)
	require.Equal(t, 1.0, result.BundleConfidenceThreshold)
}

func TestUpdateTenantSettings_rejectsNegativeRecommendationsMinSupport(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", RecommendationsMinSupport: -1})

	require.ErrorIs(t, err, settingsDomain.ErrInvalidRecommendationsMinSupport)
}

func TestUpdateTenantSettings_allowsRecommendationsMinSupportOfZero(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", RecommendationsMinSupport: 0})

	require.NoError(t, err)
	require.Equal(t, 0, result.RecommendationsMinSupport)
}

func TestUpdateTenantSettings_rejectsNegativeRecommendationsLookbackDays(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", RecommendationsLookbackDays: -1})

	require.ErrorIs(t, err, settingsDomain.ErrInvalidRecommendationsLookbackDays)
}

func TestUpdateTenantSettings_allowsRecommendationsLookbackDaysOfZero(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", RecommendationsLookbackDays: 0})

	require.NoError(t, err)
	require.Equal(t, 0, result.RecommendationsLookbackDays)
}

func TestUpdateTenantSettings_allowsMarkupOf100PercentAndAbove(t *testing.T) {
	// Unlike margin (capped under 100%), markup has no upper bound --
	// 10000 bps (100% markup, price = 2x cost) and above must be accepted.
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", DefaultTargetMarkupBasisPoints: 30000})

	require.NoError(t, err)
	require.Equal(t, 30000, result.DefaultTargetMarkupBasisPoints)
}

func TestUpdateTenantSettings_setsPickingMode(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appsettings.UpdateTenantSettingsInput{PickingMode: "list"})

	require.NoError(t, err)
	require.Equal(t, settingsDomain.PickingModeList, result.PickingMode)
	require.Equal(t, settingsDomain.PickingModeList, repo.upserted[0].PickingMode)
}

func TestUpdateTenantSettings_emptyPickingModeMeansScan(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appsettings.UpdateTenantSettingsInput{})

	require.NoError(t, err)
	require.Equal(t, settingsDomain.PickingModeScan, result.PickingMode)
}

func TestUpdateTenantSettings_rejectsUnknownPickingMode(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appsettings.UpdateTenantSettingsInput{PickingMode: "barcode"})

	require.ErrorIs(t, err, settingsDomain.ErrInvalidPickingMode)
	require.Empty(t, repo.upserted)
}

func TestUpdateTenantSettings_rejectsUnknownTimezone(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appsettings.UpdateTenantSettingsInput{Timezone: "Mars/Olympus"})

	require.ErrorIs(t, err, settingsDomain.ErrInvalidTimezone)
	require.Empty(t, repo.upserted)
}

func TestUpdateTenantSettings_rejectsUnknownDateFormat(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appsettings.UpdateTenantSettingsInput{DateFormat: "D.M.YY"})

	require.ErrorIs(t, err, settingsDomain.ErrInvalidDateFormat)
	require.Empty(t, repo.upserted)
}

func TestUpdateTenantSettings_languageDefaultsToEnglish(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "EUR"})

	require.NoError(t, err)
	require.Equal(t, locale.English, result.Language)
}

func TestUpdateTenantSettings_keepsSupportedLanguage(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{Language: "pt-BR"})

	require.NoError(t, err)
	require.Equal(t, locale.Portuguese, result.Language)
}

func TestUpdateTenantSettings_rejectsUnsupportedLanguage(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{Language: "de"})

	require.ErrorIs(t, err, locale.ErrUnsupported)
	require.Empty(t, repo.upserted)
}

func TestUpdateTenantSettings_setsExpiryAlert(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appsettings.UpdateTenantSettingsInput{
		ExpiryAlertEnabled: true, ExpiryAlertWindows: []int{7, 90},
	})

	require.NoError(t, err)
	want := settingsDomain.ExpiryAlert{Enabled: true, Windows: []int{90, 7}}
	require.Equal(t, want, result.ExpiryAlert)
	require.Equal(t, want, repo.upserted[0].ExpiryAlert)
}

func TestUpdateTenantSettings_rejectsUnknownExpiryAlertWindow(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appsettings.UpdateTenantSettingsInput{
		ExpiryAlertEnabled: true, ExpiryAlertWindows: []int{60},
	})

	require.ErrorIs(t, err, settingsDomain.ErrInvalidExpiryAlertWindow)
	require.Empty(t, repo.upserted)
}

func TestUpdateTenantSettings_rejectsEnabledExpiryAlertWithoutWindows(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appsettings.UpdateTenantSettingsInput{
		ExpiryAlertEnabled: true,
	})

	require.ErrorIs(t, err, settingsDomain.ErrExpiryAlertWindowsRequired)
}
