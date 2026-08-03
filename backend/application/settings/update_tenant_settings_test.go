package settings_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appsettings "github.com/adirkuhn/commerceos/application/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestUpdateTenantSettings_upsertsAllFields(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx, appsettings.UpdateTenantSettingsInput{
		DefaultCurrency: "USD", Timezone: "Europe/Amsterdam", DateFormat: "DD/MM/YYYY", Language: "en",
	})

	require.NoError(t, err)
	require.Equal(t, "USD", result.DefaultCurrency)
	require.Equal(t, "Europe/Amsterdam", result.Timezone)
	require.Equal(t, "DD/MM/YYYY", result.DateFormat)
	require.Equal(t, "en", result.Language)
	require.Equal(t, "tenant-1", result.TenantID)
	require.Len(t, repo.upserted, 1)
	require.Equal(t, "USD", repo.upserted[0].DefaultCurrency)
}

func TestUpdateTenantSettings_propagatesRepoError(t *testing.T) {
	repo := &stubSettingsRepo{upsertErr: errBoom}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appsettings.UpdateTenantSettingsInput{DefaultCurrency: "USD"})

	require.Error(t, err)
}
