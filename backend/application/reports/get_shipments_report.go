package reports

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/domain/errs"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

var (
	ErrInvalidShipmentGranularity = errs.New("REPORT_INVALID_SHIPMENT_GRANULARITY", "granularity must be one of week, month, quarter, year")
	ErrInvalidShipmentBasis       = errs.New("REPORT_INVALID_SHIPMENT_BASIS", "basis must be one of label_created, carrier_scan")
)

// CarrierShipments is one carrier's shipments, with its display name.
type CarrierShipments struct {
	Carrier   int
	Name      string
	Shipments int64
}

// ShipmentsReport counts shipments per week/month/quarter/year: every
// period in the range (empty ones as zero), totals, and the range's split
// by carrier, channel and country.
type ShipmentsReport struct {
	Granularity    shippingDomain.VolumeGranularity
	Basis          shippingDomain.VolumeBasis
	Points         []shippingDomain.VolumePoint
	TotalShipments int64
	TotalOrders    int64
	ByCarrier      []CarrierShipments
	ByChannel      []shippingDomain.VolumeShare
	ByCountry      []shippingDomain.VolumeShare
}

type GetShipmentsReportUseCase struct {
	db           *gorm.DB
	volumes      shippingDomain.ShipmentVolumeRepository
	settingsRepo settingsDomain.Repository
}

func NewGetShipmentsReportUseCase(db *gorm.DB, volumes shippingDomain.ShipmentVolumeRepository, settingsRepo settingsDomain.Repository) *GetShipmentsReportUseCase {
	return &GetShipmentsReportUseCase{db: db, volumes: volumes, settingsRepo: settingsRepo}
}

// Execute counts shipments in [from, to). An empty granularity means
// week; an empty basis means label_created.
func (uc *GetShipmentsReportUseCase) Execute(ctx context.Context, from, to time.Time, granularity shippingDomain.VolumeGranularity, basis shippingDomain.VolumeBasis) (*ShipmentsReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getShipmentsReport: %w", ErrToBeforeFrom)
	}
	if granularity == "" {
		granularity = shippingDomain.VolumeGranularityWeek
	}
	if !granularity.Valid() {
		return nil, fmt.Errorf("getShipmentsReport: %w", ErrInvalidShipmentGranularity)
	}
	if basis == "" {
		basis = shippingDomain.VolumeBasisLabelCreated
	}
	if !basis.Valid() {
		return nil, fmt.Errorf("getShipmentsReport: %w", ErrInvalidShipmentBasis)
	}

	report := &ShipmentsReport{Granularity: granularity, Basis: basis}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		settings, err := uc.settingsRepo.Get(ctx)
		if err != nil {
			return err
		}
		volume, err := uc.volumes.ShipmentVolume(ctx, shippingDomain.VolumeQuery{
			From: from, To: to, Granularity: granularity, Basis: basis, Timezone: settings.ReportingTimezone(),
		})
		if err != nil {
			return err
		}
		report.Points = shippingDomain.FillVolumePeriods(volume.Points, from, to, granularity, settings.ReportingLocation())
		report.TotalShipments = volume.TotalShipments
		report.TotalOrders = volume.TotalOrders
		report.ByCarrier = carrierShipments(volume.ByCarrier)
		report.ByChannel = volume.ByChannel
		report.ByCountry = volume.ByCountry
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getShipmentsReport: %w", err)
	}
	return report, nil
}

func carrierShipments(shares []shippingDomain.VolumeCarrierShare) []CarrierShipments {
	out := make([]CarrierShipments, len(shares))
	for i, s := range shares {
		out[i] = CarrierShipments{Carrier: int(s.Carrier), Name: s.Carrier.String(), Shipments: s.Shipments}
	}
	return out
}
