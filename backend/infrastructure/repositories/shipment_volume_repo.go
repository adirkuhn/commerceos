package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ShipmentVolumeRepo counts shipments for the shipments report.
type ShipmentVolumeRepo struct{ db *gorm.DB }

func NewShipmentVolumeRepo(db *gorm.DB) *ShipmentVolumeRepo { return &ShipmentVolumeRepo{db: db} }

var _ shippingDomain.ShipmentVolumeRepository = (*ShipmentVolumeRepo)(nil)

func (r *ShipmentVolumeRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("shipmentVolumeRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// shipmentMomentSQL is the instant that places a shipment in a period,
// per basis. carrier_scan mirrors shipping.IsCarrierConfirmed: the first
// status event past "registered"; NULL (not counted) until there is one.
var shipmentMomentSQL = map[shippingDomain.VolumeBasis]string{
	shippingDomain.VolumeBasisLabelCreated: `sh.created_at`,
	shippingDomain.VolumeBasisCarrierScan: `(SELECT MIN(e.occurred_at) FROM shipment_status_events e
		WHERE e.shipment_id = sh.id AND e.status_main <> '' AND e.status_main <> '` + shippingDomain.StatusMainRegistered + `')`,
}

// volumePeriodSQL buckets the "at" column. Postgres weeks start on Monday;
// shifting by a day makes them start on Sunday (shipping.VolumePeriodStart).
// Only validated granularities reach here, from this fixed map.
var volumePeriodSQL = map[shippingDomain.VolumeGranularity]string{
	shippingDomain.VolumeGranularityWeek:    `date_trunc('week', at + interval '1 day') - interval '1 day'`,
	shippingDomain.VolumeGranularityMonth:   `date_trunc('month', at)`,
	shippingDomain.VolumeGranularityQuarter: `date_trunc('quarter', at)`,
	shippingDomain.VolumeGranularityYear:    `date_trunc('year', at)`,
}

// countedShipmentsSQL is the shipments the report counts in [?, ?): not
// cancelled (a replaced label counts once, as its replacement), with the
// order's source as the channel.
func countedShipmentsSQL(moment string) string {
	return `WITH counted AS (
		SELECT * FROM (
			SELECT sh.sales_order_id, sh.carrier, sh.recipient_cc, so.source, ` + moment + ` AS at
			FROM shipments sh
			JOIN sales_orders so ON so.id = sh.sales_order_id
			WHERE sh.cancelled_at IS NULL
		) s
		WHERE at >= ? AND at < ?
	)`
}

type volumePointRow struct {
	PeriodStart time.Time
	Shipments   int64
	Orders      int64
}

type volumeShareRow struct {
	Dimension string
	Carrier   int
	Key       string
	Shipments int64
	Orders    int64
}

func (r *ShipmentVolumeRepo) ShipmentVolume(ctx context.Context, q shippingDomain.VolumeQuery) (*shippingDomain.ShipmentVolume, error) {
	moment, ok := shipmentMomentSQL[q.Basis]
	if !ok {
		return nil, fmt.Errorf("shipmentVolumeRepo.ShipmentVolume: unknown basis %q", q.Basis)
	}
	period, ok := volumePeriodSQL[q.Granularity]
	if !ok {
		return nil, fmt.Errorf("shipmentVolumeRepo.ShipmentVolume: unknown granularity %q", q.Granularity)
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if err := useReportTimezone(tx, q.Timezone); err != nil {
		return nil, fmt.Errorf("shipmentVolumeRepo.ShipmentVolume: %w", err)
	}
	counted := countedShipmentsSQL(moment)

	var points []volumePointRow
	err = tx.Raw(counted+`
		SELECT `+period+` AS period_start, COUNT(*) AS shipments, COUNT(DISTINCT sales_order_id) AS orders
		FROM counted
		GROUP BY period_start
		ORDER BY period_start`, q.From, q.To).Scan(&points).Error
	if err != nil {
		return nil, fmt.Errorf("shipmentVolumeRepo.ShipmentVolume: points: %w", err)
	}

	// One pass for the whole range: the total and the split by carrier,
	// channel and country, biggest first.
	var shares []volumeShareRow
	err = tx.Raw(counted+`
		SELECT CASE
				WHEN GROUPING(carrier) = 0 THEN 'carrier'
				WHEN GROUPING(source) = 0 THEN 'channel'
				WHEN GROUPING(recipient_cc) = 0 THEN 'country'
				ELSE 'total' END AS dimension,
			COALESCE(carrier, 0) AS carrier,
			COALESCE(source, recipient_cc, '') AS key,
			COUNT(*) AS shipments, COUNT(DISTINCT sales_order_id) AS orders
		FROM counted
		GROUP BY GROUPING SETS ((carrier), (source), (recipient_cc), ())
		ORDER BY shipments DESC, key, carrier`, q.From, q.To).Scan(&shares).Error
	if err != nil {
		return nil, fmt.Errorf("shipmentVolumeRepo.ShipmentVolume: shares: %w", err)
	}

	v := &shippingDomain.ShipmentVolume{
		Points:    make([]shippingDomain.VolumePoint, len(points)),
		ByCarrier: []shippingDomain.VolumeCarrierShare{},
		ByChannel: []shippingDomain.VolumeShare{},
		ByCountry: []shippingDomain.VolumeShare{},
	}
	for i, p := range points {
		v.Points[i] = shippingDomain.VolumePoint{PeriodStart: p.PeriodStart, Shipments: p.Shipments, Orders: p.Orders}
	}
	for _, s := range shares {
		switch s.Dimension {
		case "total":
			v.TotalShipments, v.TotalOrders = s.Shipments, s.Orders
		case "carrier":
			v.ByCarrier = append(v.ByCarrier, shippingDomain.VolumeCarrierShare{Carrier: shippingDomain.Carrier(s.Carrier), Shipments: s.Shipments})
		case "channel":
			v.ByChannel = append(v.ByChannel, shippingDomain.VolumeShare{Key: s.Key, Shipments: s.Shipments})
		case "country":
			v.ByCountry = append(v.ByCountry, shippingDomain.VolumeShare{Key: s.Key, Shipments: s.Shipments})
		}
	}
	return v, nil
}
