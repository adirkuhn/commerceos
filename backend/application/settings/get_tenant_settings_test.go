package settings_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appsettings "github.com/adirkuhn/commerceos/application/settings"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestGetTenantSettings_returnsExistingRow(t *testing.T) {
	repo := &stubSettingsRepo{getResult: &settingsDomain.TenantSettings{DefaultCurrency: "USD", Timezone: "UTC"}}
	uc := appsettings.NewGetTenantSettingsUseCase(newMockGormDB(t), repo)

	ctx := tenant.WithTenantID(context.Background(), "tenant-1")
	result, err := uc.Execute(ctx)

	require.NoError(t, err)
	require.Equal(t, "USD", result.DefaultCurrency)
	require.Equal(t, "UTC", result.Timezone)
	require.Equal(t, "tenant-1", result.TenantID)
}

func TestGetTenantSettings_returnsZeroValueWhenNoneSavedYet(t *testing.T) {
	repo := &stubSettingsRepo{}
	uc := appsettings.NewGetTenantSettingsUseCase(newMockGormDB(t), repo)

	result, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"))

	require.NoError(t, err)
	require.Equal(t, "", result.DefaultCurrency)
	require.Equal(t, "tenant-1", result.TenantID)
}

func TestGetTenantSettings_propagatesRepoError(t *testing.T) {
	repo := &stubSettingsRepo{getErr: errBoom}
	uc := appsettings.NewGetTenantSettingsUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}
