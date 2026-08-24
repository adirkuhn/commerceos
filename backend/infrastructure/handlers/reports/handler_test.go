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
	appinsights "github.com/adirkuhn/commerceos/application/insights"
	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
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

type stubGetCategoryPerformanceReport struct {
	out *appreports.CategoryPerformanceReport
	err error
}

func (s *stubGetCategoryPerformanceReport) Execute(_ context.Context, _, _ time.Time) (*appreports.CategoryPerformanceReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.CategoryPerformanceReport{CurrencyCode: "EUR"}, nil
}

type stubGetSlowMovingProductsReport struct {
	out *appreports.SlowMovingProductsReport
	err error
}

func (s *stubGetSlowMovingProductsReport) Execute(_ context.Context, _, _ time.Time) (*appreports.SlowMovingProductsReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.SlowMovingProductsReport{}, nil
}

type stubGetProfitabilityReport struct {
	out *appreports.ProfitabilityReport
	err error
}

func (s *stubGetProfitabilityReport) Execute(_ context.Context, _, _ time.Time) (*appreports.ProfitabilityReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.ProfitabilityReport{CurrencyCode: "EUR"}, nil
}

type stubGetInventorySalesReport struct {
	out *appreports.InventorySalesReport
	err error
}

func (s *stubGetInventorySalesReport) Execute(_ context.Context, _, _ time.Time) (*appreports.InventorySalesReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.InventorySalesReport{}, nil
}

type stubGetBusinessInsights struct {
	out *appinsights.BusinessInsightsReport
	err error
}

func (s *stubGetBusinessInsights) Execute(_ context.Context, _, _ time.Time) (*appinsights.BusinessInsightsReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appinsights.BusinessInsightsReport{}, nil
}

type stubGetSalesTrendsReport struct {
	out            *appreports.SalesTrendsReport
	err            error
	gotGranularity omsDomain.SalesTrendGranularity
}

func (s *stubGetSalesTrendsReport) Execute(_ context.Context, _, _ time.Time, granularity omsDomain.SalesTrendGranularity) (*appreports.SalesTrendsReport, error) {
	s.gotGranularity = granularity
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.SalesTrendsReport{CurrencyCode: "EUR", Granularity: granularity}, nil
}

type stubGetProductLifecycleReport struct {
	out *appreports.ProductLifecycleReport
	err error
}

func (s *stubGetProductLifecycleReport) Execute(_ context.Context, _, _ time.Time) (*appreports.ProductLifecycleReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.ProductLifecycleReport{CurrencyCode: "EUR"}, nil
}

type stubGetCustomerCohortsReport struct {
	out *appreports.CustomerCohortsReport
	err error
}

func (s *stubGetCustomerCohortsReport) Execute(_ context.Context, _, _ time.Time) (*appreports.CustomerCohortsReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.CustomerCohortsReport{CurrencyCode: "EUR"}, nil
}

type stubGetPromotionPerformanceReport struct {
	out            *appreports.PromotionPerformanceReport
	err            error
	gotPromotionID string
}

func (s *stubGetPromotionPerformanceReport) Execute(_ context.Context, promotionID string) (*appreports.PromotionPerformanceReport, error) {
	s.gotPromotionID = promotionID
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.PromotionPerformanceReport{PromotionID: promotionID, CurrencyCode: "EUR"}, nil
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
	h := reportshandler.NewHandler(get, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/sales-summary?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"currency_code":"EUR"`)
	assert.Contains(t, w.Body.String(), `"revenue_change_percent":18.4`)
	assert.Equal(t, "2026-08-01T00:00:00Z", get.gotFrom.Format(time.RFC3339))
	assert.Equal(t, "2026-08-31T00:00:00Z", get.gotTo.Format(time.RFC3339))
}

func TestGetSalesSummaryReportHandler_validationError(t *testing.T) {
	get := &stubGetSalesSummaryReport{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(get, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/sales-summary?from=2026-08-31T00:00:00Z&to=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetSalesSummaryReportHandler_missingRequiredParam(t *testing.T) {
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
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
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, get, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/product-performance?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"sku_code":"SKU-A"`)
	assert.Contains(t, w.Body.String(), `"quantity_sold":5`)
	assert.Contains(t, w.Body.String(), `"margin_basis_points":4000`)
}

func TestGetProductPerformanceReportHandler_validationError(t *testing.T) {
	get := &stubGetProductPerformanceReport{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, get, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
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
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, get, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/product-performance-matrix?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"quadrant":"winners"`)
	assert.Contains(t, w.Body.String(), `"median_revenue_minor_units":500`)
}

func TestGetProductPerformanceMatrixHandler_validationError(t *testing.T) {
	get := &stubGetProductPerformanceMatrix{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, get, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
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
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, get, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/customer-performance?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"customer_name":"Jane Doe"`)
	assert.Contains(t, w.Body.String(), `"average_order_value_minor_units":750`)
}

func TestGetCustomerPerformanceReportHandler_validationError(t *testing.T) {
	get := &stubGetCustomerPerformanceReport{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, get, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/customer-performance?from=2026-08-31T00:00:00Z&to=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetCategoryPerformanceReportHandler_success(t *testing.T) {
	get := &stubGetCategoryPerformanceReport{out: &appreports.CategoryPerformanceReport{
		CurrencyCode: "EUR",
		Rows: []omsDomain.CategoryPerformance{
			{CategoryName: "Beverages", QuantitySold: 2, RevenueMinorUnits: 1000, OrderCount: 1},
		},
	}}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, get, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/category-performance?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"category_name":"Beverages"`)
	assert.Contains(t, w.Body.String(), `"order_count":1`)
}

func TestGetCategoryPerformanceReportHandler_validationError(t *testing.T) {
	get := &stubGetCategoryPerformanceReport{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, get, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/category-performance?from=2026-08-31T00:00:00Z&to=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetSlowMovingProductsReportHandler_success(t *testing.T) {
	get := &stubGetSlowMovingProductsReport{out: &appreports.SlowMovingProductsReport{
		Rows: []appreports.SlowMovingProductRow{
			{SKUID: "sku-1", SKUCode: "SLOW-1", ProductName: "Slow Widget", StockOnHand: 42, QuantitySoldInPeriod: 0},
		},
	}}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, get, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/slow-moving-products?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"sku_code":"SLOW-1"`)
	assert.Contains(t, w.Body.String(), `"stock_on_hand":42`)
}

func TestGetSlowMovingProductsReportHandler_validationError(t *testing.T) {
	get := &stubGetSlowMovingProductsReport{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, get, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/slow-moving-products?from=2026-08-31T00:00:00Z&to=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetProfitabilityReportHandler_success(t *testing.T) {
	get := &stubGetProfitabilityReport{out: &appreports.ProfitabilityReport{
		CurrencyCode: "EUR",
		Summary: omsDomain.ProfitabilitySummary{
			RevenueMinorUnits: 1500, CostKnownRevenueMinorUnits: 1000, CostMinorUnits: 600, ProfitMinorUnits: 400, MarginBasisPoints: 4000,
		},
	}}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, get, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/profitability?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"profit_minor_units":400`)
	assert.Contains(t, w.Body.String(), `"margin_basis_points":4000`)
}

func TestGetProfitabilityReportHandler_validationError(t *testing.T) {
	get := &stubGetProfitabilityReport{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, get, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/profitability?from=2026-08-31T00:00:00Z&to=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetInventorySalesReportHandler_success(t *testing.T) {
	turnover := int64(2500)
	get := &stubGetInventorySalesReport{out: &appreports.InventorySalesReport{
		Rows: []appreports.InventorySalesRow{
			{SKUID: "sku-1", SKUCode: "TURN-1", ProductName: "Widget", StockOnHand: 40, QuantitySoldInPeriod: 10, TurnoverBasisPoints: &turnover},
		},
	}}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, get, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/inventory-sales?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"sku_code":"TURN-1"`)
	assert.Contains(t, w.Body.String(), `"turnover_basis_points":2500`)
}

func TestGetInventorySalesReportHandler_validationError(t *testing.T) {
	get := &stubGetInventorySalesReport{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, get, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/inventory-sales?from=2026-08-31T00:00:00Z&to=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetBusinessInsightsHandler_success(t *testing.T) {
	get := &stubGetBusinessInsights{out: &appinsights.BusinessInsightsReport{
		Insights: []appinsights.Insight{
			{Code: "revenue_declining", Severity: appinsights.InsightSeverityWarning, Message: "Revenue is down 12.5% vs the previous period."},
		},
	}}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, get, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/business-insights?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"code":"revenue_declining"`)
	assert.Contains(t, w.Body.String(), `"severity":"warning"`)
}

func TestGetBusinessInsightsHandler_validationError(t *testing.T) {
	get := &stubGetBusinessInsights{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, get, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/business-insights?from=2026-08-31T00:00:00Z&to=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetSalesTrendsReportHandler_success(t *testing.T) {
	get := &stubGetSalesTrendsReport{out: &appreports.SalesTrendsReport{
		CurrencyCode: "EUR", Granularity: omsDomain.SalesTrendGranularityWeek,
		Points:    []omsDomain.SalesTrendPoint{{RevenueMinorUnits: 1000, OrderCount: 2, UnitsSold: 5}},
		ByWeekday: []omsDomain.WeekdaySales{{Weekday: time.Monday, RevenueMinorUnits: 1000, OrderCount: 2}},
	}}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, get, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/sales-trends?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z&granularity=week")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"granularity":"week"`)
	assert.Contains(t, w.Body.String(), `"revenue_minor_units":1000`)
	assert.Equal(t, omsDomain.SalesTrendGranularityWeek, get.gotGranularity)
}

func TestGetSalesTrendsReportHandler_defaultsGranularityWhenOmitted(t *testing.T) {
	get := &stubGetSalesTrendsReport{}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, get, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/sales-trends?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, omsDomain.SalesTrendGranularityDay, get.gotGranularity)
}

func TestGetSalesTrendsReportHandler_validationError(t *testing.T) {
	get := &stubGetSalesTrendsReport{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, get, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/sales-trends?from=2026-08-31T00:00:00Z&to=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetProductLifecycleReportHandler_success(t *testing.T) {
	get := &stubGetProductLifecycleReport{out: &appreports.ProductLifecycleReport{
		CurrencyCode: "EUR",
		Rows: []appreports.ProductLifecycleRow{
			{SKUCode: "SKU-A", ProductName: "Acai", CurrentQuantitySold: 10, PreviousQuantitySold: 4, Stage: appreports.LifecycleGrowing},
		},
	}}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, get, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/product-lifecycle?from=2026-08-01T00:00:00Z&to=2026-08-31T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"sku_code":"SKU-A"`)
	assert.Contains(t, w.Body.String(), `"stage":"growing"`)
}

func TestGetProductLifecycleReportHandler_validationError(t *testing.T) {
	get := &stubGetProductLifecycleReport{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, get, &stubGetCustomerCohortsReport{}, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/product-lifecycle?from=2026-08-31T00:00:00Z&to=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetCustomerCohortsReportHandler_success(t *testing.T) {
	cohortMonth := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	get := &stubGetCustomerCohortsReport{out: &appreports.CustomerCohortsReport{
		CurrencyCode: "EUR",
		Cohorts: []appreports.CustomerCohortRow{
			{
				CohortMonth: cohortMonth, CohortSize: 4,
				Retention: []appreports.CohortRetentionCell{
					{MonthsSinceCohort: 0, RetainedCustomers: 4, RevenueMinorUnits: 4000, RetentionRateBasisPoints: 10000},
				},
			},
		},
	}}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, get, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/customer-cohorts?from=2026-06-01T00:00:00Z&to=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"cohort_size":4`)
	assert.Contains(t, w.Body.String(), `"retention_rate_basis_points":10000`)
}

func TestGetCustomerCohortsReportHandler_validationError(t *testing.T) {
	get := &stubGetCustomerCohortsReport{err: appreports.ErrToBeforeFrom}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, get, &stubGetPromotionPerformanceReport{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/customer-cohorts?from=2026-08-31T00:00:00Z&to=2026-08-01T00:00:00Z")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetPromotionPerformanceReportHandler_success(t *testing.T) {
	revenueChange := 100.0
	get := &stubGetPromotionPerformanceReport{out: &appreports.PromotionPerformanceReport{
		PromotionID: "promo-1", PromotionName: "Summer Acai", CurrencyCode: "EUR",
		During:               appreports.PromotionPerformancePeriod{RevenueMinorUnits: 2000, UnitsSold: 20},
		Before:               appreports.PromotionPerformancePeriod{RevenueMinorUnits: 1000, UnitsSold: 10},
		RevenueChangePercent: &revenueChange,
	}}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, get)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/promotions/promo-1/performance")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "promo-1", get.gotPromotionID)
	assert.Contains(t, w.Body.String(), `"promotion_name":"Summer Acai"`)
	assert.Contains(t, w.Body.String(), `"revenue_change_percent":100`)
}

func TestGetPromotionPerformanceReportHandler_notFound(t *testing.T) {
	get := &stubGetPromotionPerformanceReport{err: promotionsDomain.ErrPromotionNotFound}
	h := reportshandler.NewHandler(&stubGetSalesSummaryReport{}, &stubGetProductPerformanceReport{}, &stubGetProductPerformanceMatrix{}, &stubGetCustomerPerformanceReport{}, &stubGetCategoryPerformanceReport{}, &stubGetSlowMovingProductsReport{}, &stubGetProfitabilityReport{}, &stubGetInventorySalesReport{}, &stubGetBusinessInsights{}, &stubGetSalesTrendsReport{}, &stubGetProductLifecycleReport{}, &stubGetCustomerCohortsReport{}, get)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/reports/promotions/missing/performance")

	require.Equal(t, http.StatusNotFound, w.Code)
}
