package reports

import (
	"github.com/gin-gonic/gin"

	genreports "github.com/adirkuhn/commerceos/api/reports"
	appreports "github.com/adirkuhn/commerceos/application/reports"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) GetShipmentsReport(c *gin.Context, params genreports.GetShipmentsReportParams) {
	var granularity shippingDomain.VolumeGranularity
	if params.Granularity != nil {
		if !params.Granularity.Valid() {
			_ = mapError(c, appreports.ErrInvalidShipmentGranularity)
			return
		}
		granularity = shippingDomain.VolumeGranularity(*params.Granularity)
	}
	var basis shippingDomain.VolumeBasis
	if params.Basis != nil {
		if !params.Basis.Valid() {
			_ = mapError(c, appreports.ErrInvalidShipmentBasis)
			return
		}
		basis = shippingDomain.VolumeBasis(*params.Basis)
	}

	report, err := h.getShipmentsReport.Execute(c.Request.Context(), params.From, params.To, granularity, basis)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenShipmentsReport(report))
}

func toGenShipmentsReport(r *appreports.ShipmentsReport) genreports.ShipmentsReport {
	points := make([]genreports.ShipmentsPoint, len(r.Points))
	for i, p := range r.Points {
		points[i] = genreports.ShipmentsPoint{PeriodStart: p.PeriodStart, Shipments: p.Shipments, Orders: p.Orders}
	}
	carriers := make([]genreports.CarrierShipments, len(r.ByCarrier))
	for i, s := range r.ByCarrier {
		carriers[i] = genreports.CarrierShipments{Carrier: s.Carrier, Name: s.Name, Shipments: s.Shipments}
	}
	return genreports.ShipmentsReport{
		Granularity: genreports.ShipmentGranularity(r.Granularity), Basis: genreports.ShipmentBasis(r.Basis),
		Points: points, TotalShipments: r.TotalShipments, TotalOrders: r.TotalOrders,
		ByCarrier: carriers, ByChannel: toGenShipmentsShares(r.ByChannel), ByCountry: toGenShipmentsShares(r.ByCountry),
	}
}

func toGenShipmentsShares(shares []shippingDomain.VolumeShare) []genreports.ShipmentsShare {
	out := make([]genreports.ShipmentsShare, len(shares))
	for i, s := range shares {
		out[i] = genreports.ShipmentsShare{Key: s.Key, Shipments: s.Shipments}
	}
	return out
}
