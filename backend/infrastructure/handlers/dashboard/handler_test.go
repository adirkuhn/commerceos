package dashboard_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gendashboard "github.com/adirkuhn/commerceos/api/dashboard"
	appdashboard "github.com/adirkuhn/commerceos/application/dashboard"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	dashboardhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/dashboard"
)

func init() { gin.SetMode(gin.TestMode) }

type stubGetDashboardStats struct {
	out *appdashboard.DashboardStats
	err error
}

func (s *stubGetDashboardStats) Execute(_ context.Context) (*appdashboard.DashboardStats, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appdashboard.DashboardStats{CurrencyCode: "USD"}, nil
}

func newTestRouter(h *dashboardhandler.Handler) *gin.Engine {
	r := gin.New()
	gendashboard.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(rg *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func TestGetDashboardStatsHandler_returnsStats(t *testing.T) {
	h := dashboardhandler.NewHandler(&stubGetDashboardStats{out: &appdashboard.DashboardStats{
		TotalStockUnits: 84293, CurrencyCode: "EUR", RevenueThisMonthMinorUnits: 11600000,
		SalesByChannelThisMonth: []omsDomain.ChannelRevenue{{Source: omsDomain.SalesOrderSourceWooCommerce, TotalMinorUnits: 3150000}},
		TopProductsThisMonth:    []omsDomain.ProductSales{{SKUCode: "WHP-001", ProductName: "Headphones", QuantitySold: 234, RevenueMinorUnits: 5850000}},
	}})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/dashboard/stats")

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "84293")
	assert.Contains(t, body, "EUR")
	assert.Contains(t, body, "woocommerce")
	assert.Contains(t, body, "WHP-001")
}

func TestGetDashboardStatsHandler_propagatesError(t *testing.T) {
	h := dashboardhandler.NewHandler(&stubGetDashboardStats{err: assert.AnError})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/dashboard/stats")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}
