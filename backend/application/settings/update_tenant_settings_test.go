package settings_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appsettings "github.com/adirkuhn/commerceos/application/settings"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestUpdateTenantSettings_upsertsAllFields(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo)

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
	require.Equal(t, "DD/MM/YYYY", result.DateFormat)
	require.Equal(t, "en", result.Language)
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
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD"})

	require.Error(t, err)
}

func TestUpdateTenantSettings_rejectsNegativeTargetMarkup(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", DefaultTargetMarkupBasisPoints: -1})

	require.ErrorIs(t, err, settingsDomain.ErrInvalidTargetMarkupBasisPoints)
}

func TestUpdateTenantSettings_rejectsBundleConfidenceThresholdBelowZero(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", BundleConfidenceThreshold: -0.01})

	require.ErrorIs(t, err, settingsDomain.ErrInvalidBundleConfidenceThreshold)
}

func TestUpdateTenantSettings_rejectsBundleConfidenceThresholdAboveOne(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", BundleConfidenceThreshold: 1.01})

	require.ErrorIs(t, err, settingsDomain.ErrInvalidBundleConfidenceThreshold)
}

func TestUpdateTenantSettings_allowsBundleConfidenceThresholdOfZero(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo)

	result, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", BundleConfidenceThreshold: 0})

	require.NoError(t, err)
	require.Equal(t, 0.0, result.BundleConfidenceThreshold)
}

func TestUpdateTenantSettings_allowsBundleConfidenceThresholdOfOne(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo)

	result, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", BundleConfidenceThreshold: 1})

	require.NoError(t, err)
	require.Equal(t, 1.0, result.BundleConfidenceThreshold)
}

func TestUpdateTenantSettings_rejectsNegativeRecommendationsMinSupport(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", RecommendationsMinSupport: -1})

	require.ErrorIs(t, err, settingsDomain.ErrInvalidRecommendationsMinSupport)
}

func TestUpdateTenantSettings_allowsRecommendationsMinSupportOfZero(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo)

	result, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", RecommendationsMinSupport: 0})

	require.NoError(t, err)
	require.Equal(t, 0, result.RecommendationsMinSupport)
}

func TestUpdateTenantSettings_rejectsNegativeRecommendationsLookbackDays(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", RecommendationsLookbackDays: -1})

	require.ErrorIs(t, err, settingsDomain.ErrInvalidRecommendationsLookbackDays)
}

func TestUpdateTenantSettings_allowsRecommendationsLookbackDaysOfZero(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo)

	result, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", RecommendationsLookbackDays: 0})

	require.NoError(t, err)
	require.Equal(t, 0, result.RecommendationsLookbackDays)
}

func TestUpdateTenantSettings_allowsMarkupOf100PercentAndAbove(t *testing.T) {
	// Unlike margin (capped under 100%), markup has no upper bound --
	// 10000 bps (100% markup, price = 2x cost) and above must be accepted.
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo)

	result, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", DefaultTargetMarkupBasisPoints: 30000})

	require.NoError(t, err)
	require.Equal(t, 30000, result.DefaultTargetMarkupBasisPoints)
}
