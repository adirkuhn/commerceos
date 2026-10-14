package reports_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	reportshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/reports"
)

type stubGetShipmentsReport struct {
	out            *appreports.ShipmentsReport
	err            error
	gotGranularity shippingDomain.VolumeGranularity
	gotBasis       shippingDomain.VolumeBasis
	called         bool
}

func (s *stubGetShipmentsReport) Execute(_ context.Context, _, _ time.Time, granularity shippingDomain.VolumeGranularity, basis shippingDomain.VolumeBasis) (*appreports.ShipmentsReport, error) {
	s.called = true
	s.gotGranularity, s.gotBasis = granularity, basis
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.ShipmentsReport{Granularity: granularity, Basis: basis}, nil
}

func newShipmentsTestHandler(get *stubGetShipmentsReport) *reportshandler.Handler {
	return reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{}, &stubGetCustomerRFMReport{}, &stubGetFrequentlyBoughtTogetherReport{}, &stubGetStockoutRiskReport{}, &stubGetProductRecommendations{}, &stubGetProfitabilityTrendReport{}, &stubGetRevenueByChannelReport{}, get)
}

const shipmentsRange = "from=2026-01-01T00:00:00Z&to=2026-04-01T00:00:00Z"

func TestGetShipmentsReportHandler_success(t *testing.T) {
	get := &stubGetShipmentsReport{out: &appreports.ShipmentsReport{
		Granularity: shippingDomain.VolumeGranularityQuarter, Basis: shippingDomain.VolumeBasisCarrierScan,
		Points:         []shippingDomain.VolumePoint{{PeriodStart: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Shipments: 7, Orders: 6}},
		TotalShipments: 7, TotalOrders: 6,
		ByCarrier: []appreports.CarrierShipments{{Carrier: 1, Name: "PostNL", Shipments: 7}},
		ByChannel: []shippingDomain.VolumeShare{{Key: "bol", Shipments: 7}},
		ByCountry: []shippingDomain.VolumeShare{{Key: "NL", Shipments: 7}},
	}}

	w := doRequest(newTestRouter(newShipmentsTestHandler(get)), http.MethodGet, "/api/v1/reports/shipments?"+shipmentsRange+"&granularity=quarter&basis=carrier_scan")

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"data":{
		"granularity":"quarter","basis":"carrier_scan",
		"points":[{"period_start":"2026-01-01T00:00:00Z","shipments":7,"orders":6}],
		"total_shipments":7,"total_orders":6,
		"by_carrier":[{"carrier":1,"name":"PostNL","shipments":7}],
		"by_channel":[{"key":"bol","shipments":7}],
		"by_country":[{"key":"NL","shipments":7}]
	},"error":null}`, w.Body.String())
	assert.Equal(t, shippingDomain.VolumeGranularityQuarter, get.gotGranularity)
	assert.Equal(t, shippingDomain.VolumeBasisCarrierScan, get.gotBasis)
}

// Omitted granularity/basis reach the use case empty; it picks the defaults.
func TestGetShipmentsReportHandler_optionalParams(t *testing.T) {
	get := &stubGetShipmentsReport{}

	w := doRequest(newTestRouter(newShipmentsTestHandler(get)), http.MethodGet, "/api/v1/reports/shipments?"+shipmentsRange)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, get.gotGranularity)
	assert.Empty(t, get.gotBasis)
	assert.Contains(t, w.Body.String(), `"points":[]`)
}

func TestGetShipmentsReportHandler_validationErrors(t *testing.T) {
	for code, err := range map[string]error{
		"REPORT_END_BEFORE_START":             appreports.ErrToBeforeFrom,
		"REPORT_INVALID_SHIPMENT_GRANULARITY": appreports.ErrInvalidShipmentGranularity,
		"REPORT_INVALID_SHIPMENT_BASIS":       appreports.ErrInvalidShipmentBasis,
	} {
		w := doRequest(newTestRouter(newShipmentsTestHandler(&stubGetShipmentsReport{err: err})), http.MethodGet, "/api/v1/reports/shipments?"+shipmentsRange)

		require.Equal(t, http.StatusUnprocessableEntity, w.Code, code)
		assert.Contains(t, w.Body.String(), `"code":"`+code+`"`)
	}
}

// A value outside the contract's enum is rejected before the use case.
func TestGetShipmentsReportHandler_rejectsUnknownEnumValues(t *testing.T) {
	for q, code := range map[string]string{
		"&granularity=day": "REPORT_INVALID_SHIPMENT_GRANULARITY",
		"&basis=delivered": "REPORT_INVALID_SHIPMENT_BASIS",
	} {
		get := &stubGetShipmentsReport{}

		w := doRequest(newTestRouter(newShipmentsTestHandler(get)), http.MethodGet, "/api/v1/reports/shipments?"+shipmentsRange+q)

		require.Equal(t, http.StatusUnprocessableEntity, w.Code, q)
		assert.Contains(t, w.Body.String(), `"code":"`+code+`"`)
		assert.False(t, get.called, q)
	}
}

func TestGetShipmentsReportHandler_missingRequiredParam(t *testing.T) {
	w := doRequest(newTestRouter(newShipmentsTestHandler(&stubGetShipmentsReport{})), http.MethodGet, "/api/v1/reports/shipments?from=2026-01-01T00:00:00Z")

	require.Equal(t, http.StatusBadRequest, w.Code)
}
