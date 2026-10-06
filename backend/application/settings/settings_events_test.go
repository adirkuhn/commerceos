package settings_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appsettings "github.com/adirkuhn/commerceos/application/settings"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func validSettingsInput() appsettings.UpdateTenantSettingsInput {
	return appsettings.UpdateTenantSettingsInput{
		DefaultCurrency: "EUR", Timezone: "Europe/Amsterdam", DateFormat: "DD/MM/YYYY", Language: "en",
		DefaultTaxCountry: "NL", BatchExpiryThresholdDays: 30,
	}
}

func TestUpdateTenantSettings_RecordsChangedFields(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	before := &settingsDomain.TenantSettings{
		TenantID: "tenant-1", DefaultCurrency: "EUR", Timezone: "Europe/Amsterdam", DateFormat: "DD/MM/YYYY",
		Language: "en", DefaultTaxCountry: "NL", BatchExpiryThresholdDays: 14, PickingMode: settingsDomain.PickingModeScan,
	}
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), &stubSettingsRepo{getResult: before}, outbox)

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), validSettingsInput())

	require.NoError(t, err)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "settings.tenant_settings.updated", e.Subject)
	assert.Equal(t, "tenant-1", e.Payload["entity_id"])
	assert.Equal(t, map[string][2]any{"batch_expiry_threshold_days": {float64(14), float64(30)}}, e.Changes())
}

func TestUpdateTenantSettings_FailsWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	uc := appsettings.NewUpdateTenantSettingsUseCase(newMockGormDB(t), &stubSettingsRepo{}, &eventrecordtest.Outbox{Err: boom})

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), validSettingsInput())

	assert.ErrorIs(t, err, boom)
}
