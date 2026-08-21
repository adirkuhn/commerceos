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
	})

	require.NoError(t, err)
	require.Equal(t, 3000, result.DefaultTargetMarkupBasisPoints)
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

func TestUpdateTenantSettings_allowsMarkupOf100PercentAndAbove(t *testing.T) {
	// Unlike margin (capped under 100%), markup has no upper bound --
	// 10000 bps (100% markup, price = 2x cost) and above must be accepted.
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo)

	result, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD", DefaultTargetMarkupBasisPoints: 30000})

	require.NoError(t, err)
	require.Equal(t, 30000, result.DefaultTargetMarkupBasisPoints)
}
