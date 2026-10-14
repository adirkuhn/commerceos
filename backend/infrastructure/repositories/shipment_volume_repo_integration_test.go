//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type shipmentVolumeFixture struct {
	t            *testing.T
	db           *gorm.DB
	ctx          context.Context
	warehouseID  string
	connectionID string
	repo         *repositories.ShipmentVolumeRepo
}

func newShipmentVolumeFixture(t *testing.T) shipmentVolumeFixture {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	return shipmentVolumeFixture{
		t: t, db: db, ctx: ctx, warehouseID: warehouseID,
		connectionID: seedMyParcelConnection(t, db, ctx), repo: repositories.NewShipmentVolumeRepo(db),
	}
}

func (f shipmentVolumeFixture) order(extID string, source omsDomain.SalesOrderSource) string {
	f.t.Helper()
	var id string
	require.NoError(f.t, tenant.WithTenant(f.ctx, f.db, func(ctx context.Context) error {
		now := time.Now()
		o := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: source, ExternalOrderID: extID, WarehouseID: f.warehouseID,
			CurrencyCode: "EUR", Status: omsDomain.SalesOrderStatusProcessing, TotalMinorUnits: 1000,
			OccurredAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := repositories.NewSalesOrderRepo(f.db).Create(ctx, &o); err != nil {
			return err
		}
		id = o.ID
		return nil
	}))
	return id
}

type seededShipment struct {
	orderID   string
	carrier   shippingDomain.Carrier
	cc        string
	createdAt time.Time
	cancelled bool
	// scans are status events (status_main, at), in any order.
	scans []statusScan
}

type statusScan struct {
	main string
	at   time.Time
}

func (f shipmentVolumeFixture) shipment(s seededShipment) {
	f.t.Helper()
	require.NoError(f.t, tenant.WithTenant(f.ctx, f.db, func(ctx context.Context) error {
		sh := shippingDomain.Shipment{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: s.orderID, ConnectionID: f.connectionID,
			Carrier: s.carrier, LabelFormat: shippingDomain.LabelFormatPDF, StatusMain: "registered",
			RecipientPerson: "P", RecipientStreet: "S", RecipientCity: "C", RecipientCC: s.cc,
		}
		if err := repositories.NewShipmentRepo(f.db).Create(ctx, &sh); err != nil {
			return err
		}
		tx, _ := txctx.FromContext(ctx)
		if err := tx.Exec(`UPDATE shipments SET created_at = ? WHERE id = ?`, s.createdAt, sh.ID).Error; err != nil {
			return err
		}
		if s.cancelled {
			if err := tx.Exec(`UPDATE shipments SET cancelled_at = now() WHERE id = ?`, sh.ID).Error; err != nil {
				return err
			}
		}
		events := repositories.NewShipmentStatusEventRepo(f.db)
		for _, sc := range s.scans {
			if err := events.Append(ctx, &shippingDomain.ShipmentStatusEvent{
				TenantID: tenant.IDFromContext(ctx), ShipmentID: sh.ID, Status: 3, StatusMain: sc.main,
				Source: shippingDomain.StatusEventSourceWebhook, OccurredAt: sc.at,
			}); err != nil {
				return err
			}
		}
		return nil
	}))
}

func (f shipmentVolumeFixture) volume(q shippingDomain.VolumeQuery) *shippingDomain.ShipmentVolume {
	f.t.Helper()
	var v *shippingDomain.ShipmentVolume
	require.NoError(f.t, tenant.WithTenant(f.ctx, f.db, func(ctx context.Context) error {
		var err error
		v, err = f.repo.ShipmentVolume(ctx, q)
		return err
	}))
	return v
}

func scan(main string, at time.Time) statusScan { return statusScan{main, at} }

// Label date basis: Sunday weeks in the tenant's zone, cancelled labels
// left out, orders counted once however many labels they have, and the
// carrier/channel/country split over the whole range.
func TestShipmentVolumeRepo_Integration_LabelCreatedSundayWeeks(t *testing.T) {
	const tz = "Europe/Amsterdam"
	ams, err := time.LoadLocation(tz)
	require.NoError(t, err)
	f := newShipmentVolumeFixture(t)
	bol := f.order("vol-bol", omsDomain.SalesOrderSourceBol)
	woo := f.order("vol-woo", omsDomain.SalesOrderSourceWooCommerce)
	woo2 := f.order("vol-woo-2", omsDomain.SalesOrderSourceWooCommerce)

	// Saturday 2026-10-17 23:30 UTC is Sunday 01:30 in Amsterdam: week of 18 Oct.
	f.shipment(seededShipment{orderID: bol, carrier: shippingDomain.CarrierPostNL, cc: "NL", createdAt: time.Date(2026, 10, 17, 23, 30, 0, 0, time.UTC)})
	// Two labels for one order in the week of 11 Oct (Wednesday + Saturday).
	f.shipment(seededShipment{orderID: woo, carrier: shippingDomain.CarrierPostNL, cc: "NL", createdAt: time.Date(2026, 10, 14, 10, 0, 0, 0, ams)})
	f.shipment(seededShipment{orderID: woo, carrier: shippingDomain.CarrierDHLForYou, cc: "BE", createdAt: time.Date(2026, 10, 17, 10, 0, 0, 0, ams)})
	// Cancelled label: not counted.
	f.shipment(seededShipment{orderID: woo2, carrier: shippingDomain.CarrierDPD, cc: "DE", createdAt: time.Date(2026, 10, 15, 10, 0, 0, 0, ams), cancelled: true})
	// Outside the range.
	f.shipment(seededShipment{orderID: woo2, carrier: shippingDomain.CarrierPostNL, cc: "NL", createdAt: time.Date(2026, 9, 1, 10, 0, 0, 0, ams)})

	v := f.volume(shippingDomain.VolumeQuery{
		From: time.Date(2026, 10, 11, 0, 0, 0, 0, ams), To: time.Date(2026, 10, 25, 0, 0, 0, 0, ams),
		Granularity: shippingDomain.VolumeGranularityWeek, Basis: shippingDomain.VolumeBasisLabelCreated, Timezone: tz,
	})

	require.Len(t, v.Points, 2)
	assert.True(t, time.Date(2026, 10, 11, 0, 0, 0, 0, ams).Equal(v.Points[0].PeriodStart), v.Points[0].PeriodStart)
	assert.Equal(t, int64(2), v.Points[0].Shipments)
	assert.Equal(t, int64(1), v.Points[0].Orders)
	assert.True(t, time.Date(2026, 10, 18, 0, 0, 0, 0, ams).Equal(v.Points[1].PeriodStart), v.Points[1].PeriodStart)
	assert.Equal(t, int64(1), v.Points[1].Shipments)
	assert.Equal(t, int64(3), v.TotalShipments)
	assert.Equal(t, int64(2), v.TotalOrders)
	assert.Equal(t, []shippingDomain.VolumeCarrierShare{
		{Carrier: shippingDomain.CarrierPostNL, Shipments: 2}, {Carrier: shippingDomain.CarrierDHLForYou, Shipments: 1},
	}, v.ByCarrier)
	assert.Equal(t, []shippingDomain.VolumeShare{{Key: "woocommerce", Shipments: 2}, {Key: "bol", Shipments: 1}}, v.ByChannel)
	assert.Equal(t, []shippingDomain.VolumeShare{{Key: "NL", Shipments: 2}, {Key: "BE", Shipments: 1}}, v.ByCountry)
}

// Carrier scan basis: a shipment is placed by its first status past
// "registered"; one the carrier hasn't scanned is not counted.
func TestShipmentVolumeRepo_Integration_CarrierScanUsesFirstScan(t *testing.T) {
	const tz = "Europe/Amsterdam"
	ams, err := time.LoadLocation(tz)
	require.NoError(t, err)
	f := newShipmentVolumeFixture(t)
	o1 := f.order("scan-1", omsDomain.SalesOrderSourceBol)
	o2 := f.order("scan-2", omsDomain.SalesOrderSourceBol)

	// Label in March, handed to the carrier in April (first scan), delivered later.
	f.shipment(seededShipment{orderID: o1, carrier: shippingDomain.CarrierPostNL, cc: "NL", createdAt: time.Date(2026, 3, 31, 18, 0, 0, 0, ams), scans: []statusScan{
		scan("delivered", time.Date(2026, 4, 2, 12, 0, 0, 0, ams)),
		scan("registered", time.Date(2026, 3, 31, 18, 0, 0, 0, ams)),
		scan("handed_to_carrier", time.Date(2026, 4, 1, 9, 0, 0, 0, ams)),
	}})
	// Only registered: not scanned yet.
	f.shipment(seededShipment{orderID: o2, carrier: shippingDomain.CarrierPostNL, cc: "NL", createdAt: time.Date(2026, 4, 3, 18, 0, 0, 0, ams), scans: []statusScan{scan("registered", time.Date(2026, 4, 3, 18, 0, 0, 0, ams))}})

	q := shippingDomain.VolumeQuery{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, ams), To: time.Date(2027, 1, 1, 0, 0, 0, 0, ams),
		Granularity: shippingDomain.VolumeGranularityQuarter, Basis: shippingDomain.VolumeBasisCarrierScan, Timezone: tz,
	}
	v := f.volume(q)

	require.Len(t, v.Points, 1)
	assert.True(t, time.Date(2026, 4, 1, 0, 0, 0, 0, ams).Equal(v.Points[0].PeriodStart), v.Points[0].PeriodStart)
	assert.Equal(t, int64(1), v.TotalShipments)

	q.Basis = shippingDomain.VolumeBasisLabelCreated
	v = f.volume(q)
	require.Len(t, v.Points, 2)
	assert.True(t, time.Date(2026, 1, 1, 0, 0, 0, 0, ams).Equal(v.Points[0].PeriodStart), v.Points[0].PeriodStart)
	assert.Equal(t, int64(2), v.TotalShipments)
}

// Month and year buckets come straight from date_trunc in the tenant's zone.
func TestShipmentVolumeRepo_Integration_MonthAndYear(t *testing.T) {
	const tz = "Europe/Amsterdam"
	ams, err := time.LoadLocation(tz)
	require.NoError(t, err)
	f := newShipmentVolumeFixture(t)
	o := f.order("my-1", omsDomain.SalesOrderSourceManual)
	// 22:30 UTC on 31 Dec 2025 is already 2026 in Amsterdam.
	f.shipment(seededShipment{orderID: o, carrier: shippingDomain.CarrierPostNL, cc: "NL", createdAt: time.Date(2025, 12, 31, 23, 30, 0, 0, time.UTC)})

	q := shippingDomain.VolumeQuery{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, ams), To: time.Date(2027, 1, 1, 0, 0, 0, 0, ams),
		Granularity: shippingDomain.VolumeGranularityYear, Basis: shippingDomain.VolumeBasisLabelCreated, Timezone: tz,
	}
	v := f.volume(q)
	require.Len(t, v.Points, 1)
	assert.True(t, time.Date(2026, 1, 1, 0, 0, 0, 0, ams).Equal(v.Points[0].PeriodStart), v.Points[0].PeriodStart)

	q.Granularity = shippingDomain.VolumeGranularityMonth
	v = f.volume(q)
	require.Len(t, v.Points, 1)
	assert.True(t, time.Date(2026, 1, 1, 0, 0, 0, 0, ams).Equal(v.Points[0].PeriodStart), v.Points[0].PeriodStart)
}

// Empty range: zero totals, empty (not nil) lists.
func TestShipmentVolumeRepo_Integration_Empty(t *testing.T) {
	f := newShipmentVolumeFixture(t)

	v := f.volume(shippingDomain.VolumeQuery{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		Granularity: shippingDomain.VolumeGranularityWeek, Basis: shippingDomain.VolumeBasisLabelCreated, Timezone: "UTC",
	})

	assert.Zero(t, v.TotalShipments)
	assert.Zero(t, v.TotalOrders)
	assert.NotNil(t, v.Points)
	assert.NotNil(t, v.ByCarrier)
	assert.NotNil(t, v.ByChannel)
	assert.NotNil(t, v.ByCountry)
}
