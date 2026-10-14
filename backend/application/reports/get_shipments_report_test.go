package reports_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

type stubShipmentVolumeRepo struct {
	volume *shippingDomain.ShipmentVolume
	err    error
	got    shippingDomain.VolumeQuery
	calls  int
}

func (s *stubShipmentVolumeRepo) ShipmentVolume(_ context.Context, q shippingDomain.VolumeQuery) (*shippingDomain.ShipmentVolume, error) {
	s.calls++
	s.got = q
	if s.err != nil {
		return nil, s.err
	}
	return s.volume, nil
}

var (
	shipFrom = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	shipTo   = time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
)

func TestGetShipmentsReport_countsPerPeriodWithEmptyPeriodsAndTotals(t *testing.T) {
	feb := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	repo := &stubShipmentVolumeRepo{volume: &shippingDomain.ShipmentVolume{
		Points:         []shippingDomain.VolumePoint{{PeriodStart: feb, Shipments: 5, Orders: 4}},
		TotalShipments: 5, TotalOrders: 4,
		ByCarrier: []shippingDomain.VolumeCarrierShare{{Carrier: shippingDomain.CarrierPostNL, Shipments: 5}},
		ByChannel: []shippingDomain.VolumeShare{{Key: "bol", Shipments: 5}},
		ByCountry: []shippingDomain.VolumeShare{{Key: "NL", Shipments: 5}},
	}}
	uc := appreports.NewGetShipmentsReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{})

	r, err := uc.Execute(context.Background(), shipFrom, shipTo, shippingDomain.VolumeGranularityMonth, shippingDomain.VolumeBasisLabelCreated)

	require.NoError(t, err)
	require.Len(t, r.Points, 3)
	assert.Equal(t, int64(0), r.Points[0].Shipments)
	assert.Equal(t, int64(5), r.Points[1].Shipments)
	assert.Equal(t, int64(4), r.Points[1].Orders)
	assert.Equal(t, int64(5), r.TotalShipments)
	assert.Equal(t, int64(4), r.TotalOrders)
	assert.Equal(t, shippingDomain.VolumeGranularityMonth, r.Granularity)
	assert.Equal(t, shippingDomain.VolumeBasisLabelCreated, r.Basis)
	require.Len(t, r.ByCarrier, 1)
	assert.Equal(t, "PostNL", r.ByCarrier[0].Name)
	assert.Equal(t, 1, r.ByCarrier[0].Carrier)
	assert.Equal(t, []shippingDomain.VolumeShare{{Key: "bol", Shipments: 5}}, r.ByChannel)
	assert.Equal(t, []shippingDomain.VolumeShare{{Key: "NL", Shipments: 5}}, r.ByCountry)
	assert.Equal(t, shippingDomain.VolumeQuery{
		From: shipFrom, To: shipTo, Granularity: shippingDomain.VolumeGranularityMonth,
		Basis: shippingDomain.VolumeBasisLabelCreated, Timezone: "UTC",
	}, repo.got)
}

// Periods are the tenant's local calendar.
func TestGetShipmentsReport_bucketsInTheTenantTimezone(t *testing.T) {
	repo := &stubShipmentVolumeRepo{volume: &shippingDomain.ShipmentVolume{}}
	settings := &stubSettingsRepo{settings: &settingsDomain.TenantSettings{Timezone: "Europe/Amsterdam"}}
	uc := appreports.NewGetShipmentsReportUseCase(newMockGormDB(t), repo, settings)

	r, err := uc.Execute(context.Background(), shipFrom, shipTo, shippingDomain.VolumeGranularityQuarter, shippingDomain.VolumeBasisCarrierScan)

	require.NoError(t, err)
	assert.Equal(t, "Europe/Amsterdam", repo.got.Timezone)
	assert.Equal(t, shippingDomain.VolumeBasisCarrierScan, repo.got.Basis)
	require.NotEmpty(t, r.Points)
	assert.Equal(t, "Europe/Amsterdam", r.Points[0].PeriodStart.Location().String())
}

func TestGetShipmentsReport_defaultsToWeeksByLabelDate(t *testing.T) {
	repo := &stubShipmentVolumeRepo{volume: &shippingDomain.ShipmentVolume{}}
	uc := appreports.NewGetShipmentsReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{})

	r, err := uc.Execute(context.Background(), shipFrom, shipTo, "", "")

	require.NoError(t, err)
	assert.Equal(t, shippingDomain.VolumeGranularityWeek, repo.got.Granularity)
	assert.Equal(t, shippingDomain.VolumeBasisLabelCreated, repo.got.Basis)
	assert.Equal(t, shippingDomain.VolumeGranularityWeek, r.Granularity)
}

func TestGetShipmentsReport_rejectsBadInput(t *testing.T) {
	cases := map[string]struct {
		from, to    time.Time
		granularity shippingDomain.VolumeGranularity
		basis       shippingDomain.VolumeBasis
		want        error
	}{
		"to before from":  {shipTo, shipFrom, "month", "label_created", appreports.ErrToBeforeFrom},
		"day granularity": {shipFrom, shipTo, "day", "label_created", appreports.ErrInvalidShipmentGranularity},
		"unknown basis":   {shipFrom, shipTo, "month", "delivered", appreports.ErrInvalidShipmentBasis},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &stubShipmentVolumeRepo{}
			uc := appreports.NewGetShipmentsReportUseCase(newMockGormDB(t), repo, &stubSettingsRepo{})

			_, err := uc.Execute(context.Background(), tc.from, tc.to, tc.granularity, tc.basis)

			require.ErrorIs(t, err, tc.want)
			assert.Zero(t, repo.calls)
		})
	}
}

func TestGetShipmentsReport_propagatesRepoError(t *testing.T) {
	boom := errors.New("boom")
	uc := appreports.NewGetShipmentsReportUseCase(newMockGormDB(t), &stubShipmentVolumeRepo{err: boom}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), shipFrom, shipTo, shippingDomain.VolumeGranularityWeek, shippingDomain.VolumeBasisLabelCreated)

	require.ErrorIs(t, err, boom)
}
