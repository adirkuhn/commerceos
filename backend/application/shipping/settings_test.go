package shipping_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestGetShippingSettings_returnsSettingsAndOverridesTogether(t *testing.T) {
	settings := &shippingDomain.ShippingConnectionSettings{ConnectionID: "conn-1", DefaultCarrier: shippingDomain.CarrierPostNL}
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) {
			return settings, nil
		},
	}
	overrides := []shippingDomain.ShippingCarrierOverride{{Carrier: shippingDomain.CarrierDPD}}
	overrideRepo := &fakeShippingCarrierOverrideRepo{
		listByConnectionFn: func(_ context.Context, _ string) ([]shippingDomain.ShippingCarrierOverride, error) {
			return overrides, nil
		},
	}
	uc := appshipping.NewGetShippingSettingsUseCase(newMockGormDB(t, 1), settingsRepo, overrideRepo)

	result, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "conn-1")

	require.NoError(t, err)
	assert.Equal(t, shippingDomain.CarrierPostNL, result.Settings.DefaultCarrier)
	require.Len(t, result.CarrierOverrides, 1)
	assert.Equal(t, shippingDomain.CarrierDPD, result.CarrierOverrides[0].Carrier)
}

func TestUpdateShippingSettings_persistsAndStampsTenant(t *testing.T) {
	var upserted *shippingDomain.ShippingConnectionSettings
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		UpsertFn: func(_ context.Context, s *shippingDomain.ShippingConnectionSettings) error {
			upserted = s
			return nil
		},
	}
	uc := appshipping.NewUpdateShippingSettingsUseCase(newMockGormDB(t, 1), settingsRepo)

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appshipping.UpdateShippingSettingsInput{
		ConnectionID: "conn-1", DefaultCarrier: shippingDomain.CarrierDPD, DefaultLabelFormat: shippingDomain.LabelFormatZPL,
	})

	require.NoError(t, err)
	require.NotNil(t, upserted)
	assert.Equal(t, "tenant-1", upserted.TenantID)
	assert.Equal(t, shippingDomain.CarrierDPD, upserted.DefaultCarrier)
	assert.Equal(t, shippingDomain.LabelFormatZPL, upserted.DefaultLabelFormat)
}

func TestUpsertCarrierOverride_persistsAndStampsTenant(t *testing.T) {
	var upserted *shippingDomain.ShippingCarrierOverride
	overrideRepo := &fakeShippingCarrierOverrideRepo{
		upsertFn: func(_ context.Context, o *shippingDomain.ShippingCarrierOverride) error {
			upserted = o
			return nil
		},
	}
	uc := appshipping.NewUpsertCarrierOverrideUseCase(newMockGormDB(t, 1), overrideRepo)

	sig := true
	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appshipping.UpsertCarrierOverrideInput{
		ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, SignatureRequired: &sig,
	})

	require.NoError(t, err)
	require.NotNil(t, upserted)
	assert.Equal(t, "tenant-1", upserted.TenantID)
	require.NotNil(t, upserted.SignatureRequired)
	assert.True(t, *upserted.SignatureRequired)
}
