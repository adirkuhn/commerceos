package shipping_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
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
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) {
			return &shippingDomain.ShippingConnectionSettings{DefaultCarrier: shippingDomain.CarrierPostNL, DefaultLabelFormat: shippingDomain.LabelFormatZPL}, nil
		},
		UpsertFn: func(_ context.Context, s *shippingDomain.ShippingConnectionSettings) error {
			upserted = s
			return nil
		},
	}
	outbox := &eventrecordtest.Outbox{}
	uc := appshipping.NewUpdateShippingSettingsUseCase(newMockGormDB(t, 1), settingsRepo, outbox)

	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appshipping.UpdateShippingSettingsInput{
		ConnectionID: "conn-1", DefaultCarrier: shippingDomain.CarrierDPD, DefaultLabelFormat: shippingDomain.LabelFormatZPL,
	})

	require.NoError(t, err)
	require.NotNil(t, upserted)
	assert.Equal(t, "tenant-1", upserted.TenantID)
	assert.Equal(t, shippingDomain.CarrierDPD, upserted.DefaultCarrier)
	assert.Equal(t, shippingDomain.LabelFormatZPL, upserted.DefaultLabelFormat)

	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "shipping.connection_settings.updated", e.Subject)
	assert.Equal(t, "conn-1", e.Payload["entity_id"])
	assert.Equal(t, map[string][2]any{"default_carrier": {float64(shippingDomain.CarrierPostNL), float64(shippingDomain.CarrierDPD)}}, e.Changes())
}

func TestUpsertCarrierOverride_persistsAndStampsTenant(t *testing.T) {
	var upserted *shippingDomain.ShippingCarrierOverride
	overrideRepo := &fakeShippingCarrierOverrideRepo{
		findByConnectionAndCarrierFn: noCarrierOverride,
		upsertFn: func(_ context.Context, o *shippingDomain.ShippingCarrierOverride) error {
			upserted = o
			return nil
		},
	}
	outbox := &eventrecordtest.Outbox{}
	uc := appshipping.NewUpsertCarrierOverrideUseCase(newMockGormDB(t, 1), overrideRepo, outbox)

	sig := true
	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appshipping.UpsertCarrierOverrideInput{
		ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, SignatureRequired: &sig,
	})

	require.NoError(t, err)
	require.NotNil(t, upserted)
	assert.Equal(t, "tenant-1", upserted.TenantID)
	require.NotNil(t, upserted.SignatureRequired)
	assert.True(t, *upserted.SignatureRequired)

	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "shipping.carrier_override.updated", e.Subject)
	assert.Equal(t, map[string]any{"carrier": float64(shippingDomain.CarrierPostNL)}, e.Data())
	assert.Equal(t, map[string][2]any{"signature_required": {nil, true}}, e.Changes())
}

func noCarrierOverride(_ context.Context, _ string, _ shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
	return nil, nil
}

func TestShippingSettings_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	settingsRepo := &mocks.ShippingConnectionSettingsRepo{
		FindByConnectionIDFn: func(_ context.Context, _ string) (*shippingDomain.ShippingConnectionSettings, error) { return nil, nil },
		UpsertFn:             func(_ context.Context, _ *shippingDomain.ShippingConnectionSettings) error { return nil },
	}
	_, err := appshipping.NewUpdateShippingSettingsUseCase(newMockGormDB(t, 1), settingsRepo, &eventrecordtest.Outbox{Err: boom}).
		Execute(context.Background(), appshipping.UpdateShippingSettingsInput{ConnectionID: "conn-1"})
	assert.ErrorIs(t, err, boom)

	overrideRepo := &fakeShippingCarrierOverrideRepo{
		findByConnectionAndCarrierFn: noCarrierOverride,
		upsertFn:                     func(_ context.Context, _ *shippingDomain.ShippingCarrierOverride) error { return nil },
	}
	_, err = appshipping.NewUpsertCarrierOverrideUseCase(newMockGormDB(t, 1), overrideRepo, &eventrecordtest.Outbox{Err: boom}).
		Execute(context.Background(), appshipping.UpsertCarrierOverrideInput{ConnectionID: "conn-1"})
	assert.ErrorIs(t, err, boom)
}

func TestUpsertCarrierOverride_persistsTrackingURLTemplate(t *testing.T) {
	var upserted *shippingDomain.ShippingCarrierOverride
	overrideRepo := &fakeShippingCarrierOverrideRepo{
		findByConnectionAndCarrierFn: noCarrierOverride,
		upsertFn: func(_ context.Context, o *shippingDomain.ShippingCarrierOverride) error {
			upserted = o
			return nil
		},
	}
	uc := appshipping.NewUpsertCarrierOverrideUseCase(newMockGormDB(t, 1), overrideRepo, &eventrecordtest.Outbox{})

	tmpl := "https://postnl.nl/tracktrace/?B={barcode}&P={postcode}&D=NL&T=C&L=EN"
	_, err := uc.Execute(tenant.WithTenantID(context.Background(), "tenant-1"), appshipping.UpsertCarrierOverrideInput{
		ConnectionID: "conn-1", Carrier: shippingDomain.CarrierPostNL, TrackingURLTemplate: &tmpl,
	})

	require.NoError(t, err)
	require.NotNil(t, upserted.TrackingURLTemplate)
	assert.Equal(t, tmpl, *upserted.TrackingURLTemplate)
}
