package shipping

import (
	"context"
	"time"
)

// VolumeGranularity is the period size the shipments report counts by.
type VolumeGranularity string

const (
	VolumeGranularityWeek    VolumeGranularity = "week"
	VolumeGranularityMonth   VolumeGranularity = "month"
	VolumeGranularityQuarter VolumeGranularity = "quarter"
	VolumeGranularityYear    VolumeGranularity = "year"
)

func (g VolumeGranularity) Valid() bool {
	switch g {
	case VolumeGranularityWeek, VolumeGranularityMonth, VolumeGranularityQuarter, VolumeGranularityYear:
		return true
	}
	return false
}

// VolumeBasis is which moment of a shipment places it in a period.
type VolumeBasis string

const (
	// VolumeBasisLabelCreated: the moment the label was made -- every
	// shipment has one.
	VolumeBasisLabelCreated VolumeBasis = "label_created"
	// VolumeBasisCarrierScan: the first carrier status past "registered"
	// (see IsCarrierConfirmed) -- when the parcel really left. A shipment
	// the carrier hasn't scanned yet is not counted.
	VolumeBasisCarrierScan VolumeBasis = "carrier_scan"
)

func (b VolumeBasis) Valid() bool {
	return b == VolumeBasisLabelCreated || b == VolumeBasisCarrierScan
}

// VolumeQuery selects the shipments the report counts: not cancelled
// (a replaced label counts once, as its replacement), placed by Basis in
// [From, To), bucketed in Timezone (an IANA zone).
type VolumeQuery struct {
	From        time.Time
	To          time.Time
	Granularity VolumeGranularity
	Basis       VolumeBasis
	Timezone    string
}

// VolumePoint is one period: its shipments and the distinct orders they
// belong to (an order can have more than one label).
type VolumePoint struct {
	PeriodStart time.Time
	Shipments   int64
	Orders      int64
}

// VolumeShare is the whole range's shipments for one carrier, channel or
// country.
type VolumeShare struct {
	Key       string
	Shipments int64
}

// VolumeCarrierShare is VolumeShare keyed by carrier.
type VolumeCarrierShare struct {
	Carrier   Carrier
	Shipments int64
}

// ShipmentVolume is what the repository counts for one VolumeQuery.
// Points hold only periods with shipments; FillVolumePeriods adds the
// empty ones.
type ShipmentVolume struct {
	Points         []VolumePoint
	TotalShipments int64
	TotalOrders    int64
	ByCarrier      []VolumeCarrierShare
	// ByChannel is keyed by the sales order's source (woocommerce, bol, ...).
	ByChannel []VolumeShare
	// ByCountry is keyed by the recipient's ISO country code.
	ByCountry []VolumeShare
}

// ShipmentVolumeRepository counts shipments for the shipments report.
type ShipmentVolumeRepository interface {
	ShipmentVolume(ctx context.Context, q VolumeQuery) (*ShipmentVolume, error)
}

// VolumePeriodStart is the start of the period holding at, on loc's
// calendar. Weeks start on Sunday, like the dashboard's sales card.
func VolumePeriodStart(at time.Time, g VolumeGranularity, loc *time.Location) time.Time {
	local := at.In(loc)
	y, m, d := local.Date()
	switch g {
	case VolumeGranularityWeek:
		return time.Date(y, m, d-int(local.Weekday()), 0, 0, 0, 0, loc)
	case VolumeGranularityQuarter:
		return time.Date(y, m-(m-1)%3, 1, 0, 0, 0, 0, loc)
	case VolumeGranularityYear:
		return time.Date(y, time.January, 1, 0, 0, 0, 0, loc)
	default:
		return time.Date(y, m, 1, 0, 0, 0, 0, loc)
	}
}

func nextVolumePeriod(start time.Time, g VolumeGranularity) time.Time {
	switch g {
	case VolumeGranularityWeek:
		return start.AddDate(0, 0, 7)
	case VolumeGranularityQuarter:
		return start.AddDate(0, 3, 0)
	case VolumeGranularityYear:
		return start.AddDate(1, 0, 0)
	default:
		return start.AddDate(0, 1, 0)
	}
}

// FillVolumePeriods returns one point for every period touched by
// [from, to), in order, taking counts from points (matched by instant)
// and zero for periods without shipments. Period starts are in loc.
func FillVolumePeriods(points []VolumePoint, from, to time.Time, g VolumeGranularity, loc *time.Location) []VolumePoint {
	if !to.After(from) {
		return []VolumePoint{}
	}
	byStart := make(map[int64]VolumePoint, len(points))
	for _, p := range points {
		byStart[p.PeriodStart.Unix()] = p
	}
	filled := []VolumePoint{}
	for start := VolumePeriodStart(from, g, loc); start.Before(to); start = nextVolumePeriod(start, g) {
		p := byStart[start.Unix()]
		filled = append(filled, VolumePoint{PeriodStart: start, Shipments: p.Shipments, Orders: p.Orders})
	}
	return filled
}
