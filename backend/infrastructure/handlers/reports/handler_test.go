package reports_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genreports "github.com/adirkuhn/commerceos/api/reports"
	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	reportshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/reports"
)

func int64Ptr(v int64) *int64 { return &v }

func init() { gin.SetMode(gin.TestMode) }

type stubGetSalesSummaryReport struct {
	out     *appreports.SalesSummaryReport
	err     error
	gotFrom time.Time
	gotTo   time.Time
}

func (s *stubGetSalesSummaryReport) Execute(_ context.Context, from, to time.Time) (*appreports.SalesSummaryReport, error) {
	s.gotFrom, s.gotTo = from, to
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.SalesSummaryReport{CurrencyCode: "EUR"}, nil
}

type stubGetProductPerformanceReport struct {
	out     *appreports.ProductPerformanceReport
	err     error
	gotFrom time.Time
	gotTo   time.Time
}

func (s *stubGetProductPerformanceReport) Execute(_ context.Context, from, to time.Time) (*appreports.ProductPerformanceReport, error) {
	s.gotFrom, s.gotTo = from, to
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.ProductPerformanceReport{CurrencyCode: "EUR"}, nil
}

type stubGetProductPerformanceMatrix struct {
	out *appreports.ProductPerformanceMatrix
	err error
}

func (s *stubGetProductPerformanceMatrix) Execute(_ context.Context, _, _ time.Time) (*appreports.ProductPerformanceMatrix, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.ProductPerformanceMatrix{CurrencyCode: "EUR"}, nil
}

type stubGetCustomerPerformanceReport struct {
	out *appreports.CustomerPerformanceReport
	err error
}

func (s *stubGetCustomerPerformanceReport) Execute(_ context.Context, _, _ time.Time) (*appreports.CustomerPerformanceReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.CustomerPerformanceReport{CurrencyCode: "EUR"}, nil
}

func newTestRouter(h *reportshandler.Handler) *gin.Engine {
	r := gin.New()
	genreports.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(rg *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func TestGetSalesSummaryReportHandler_success(t *testing.T) {
	revenueChange := 18.4
	get := &stubGetSalesSummaryReport{out: &appreports.SalesSummaryReport{
		CurrencyCode: "EUR", RevenueChangePercent: &revenueChange,
	}}
	h := reportshandler.NewHandler(get, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/sales-summary?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"currency_code":"EUR"`)
	assert.Contains(t, w.Body.String(), `"revenue_change_percent":18.4`)
	assert.Equal(t, "2026-08-01T00:00:00Z", get.gotFrom.Format(time.RFC3339))
	assert.Equal(t, "2026-08-31T00:00:00Z", get.gotTo.Format(time.RFC3339))
}

func TestGetSalesSummaryReportHandler_validationError(t *testing.T) {
	get := &stubGetSalesSummaryReport{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(get, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/sales-summary?from=2026-08-31T00:00:00Z&to=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetSalesSummaryReportHandler_missingRequiredParam(t *testing.T) {
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/sales-summary?from=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetProductPerformanceReportHandler_success(t *testing.T) {
	get := &stubGetProductPerformanceReport{out: &appreports.ProductPerformanceReport{
		CurrencyCode: "EUR",
		Rows: []appreports.ProductPerformanceRow{{
			ProductPerformance:  omsDomain.ProductPerformance{SKUCode: "SKU-A", ProductName: "Acai", QuantitySold: 5, RevenueMinorUnits: 2500, OrderCount: 2},
			CostPriceMinorUnits: int64Ptr(300), ProfitMinorUnits: int64Ptr(1000), MarginBasisPoints: int64Ptr(4000),
		}},
	}}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, get, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/product-performance?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"sku_code":"SKU-A"`)
	assert.Contains(t, w.Body.String(), `"quantity_sold":5`)
	assert.Contains(t, w.Body.String(), `"margin_basis_points":4000`)
}

func TestGetProductPerformanceReportHandler_validationError(t *testing.T) {
	get := &stubGetProductPerformanceReport{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, get, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/product-performance?from=2026-08-31T00:00:00Z&to=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetProductPerformanceMatrixHandler_success(t *testing.T) {
	get := &stubGetProductPerformanceMatrix{out: &appreports.ProductPerformanceMatrix{
		CurrencyCode: "EUR", MedianRevenueMinorUnits: 500, MedianMarginBasisPoints: 2000,
		Rows: []appreports.ProductPerformanceMatrixRow{{
			ProductPerformanceRow: appreports.ProductPerformanceRow{
				ProductPerformance: omsDomain.ProductPerformance{SKUCode: "SKU-A", RevenueMinorUnits: 1000},
				MarginBasisPoints:  int64Ptr(5000),
			},
			Quadrant: appreports.QuadrantWinners,
		}},
	}}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, get, &stubGetCustomerPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/product-performance-matrix?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"quadrant":"winners"`)
	assert.Contains(t, w.Body.String(), `"median_revenue_minor_units":500`)
}

func TestGetProductPerformanceMatrixHandler_validationError(t *testing.T) {
	get := &stubGetProductPerformanceMatrix{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, get, &stubGetCustomerPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/product-performance-matrix?from=2026-08-31T00:00:00Z&to=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetCustomerPerformanceReportHandler_success(t *testing.T) {
	get := &stubGetCustomerPerformanceReport{out: &appreports.CustomerPerformanceReport{
		CurrencyCode: "EUR",
		Rows: []omsDomain.CustomerPerformance{
			{CustomerID: "cust-1", CustomerName: "Jane Doe", OrderCount: 2, RevenueMinorUnits: 1500, AverageOrderValueMinorUnits: 750},
		},
	}}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, get)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/customer-performance?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"customer_name":"Jane Doe"`)
	assert.Contains(t, w.Body.String(), `"average_order_value_minor_units":750`)
}

func TestGetCustomerPerformanceReportHandler_validationError(t *testing.T) {
	get := &stubGetCustomerPerformanceReport{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, get)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/customer-performance?from=2026-08-31T00:00:00Z&to=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
