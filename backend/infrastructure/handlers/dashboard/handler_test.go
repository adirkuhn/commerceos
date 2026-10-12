package dashboard_test

import (
	"context"
	"encoding/json"
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
	}}, &stubGetTodayQueue{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/dashboard/stats")

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "84293")
	assert.Contains(t, body, "EUR")
	assert.Contains(t, body, "woocommerce")
	assert.Contains(t, body, "WHP-001")
}

func TestGetDashboardStatsHandler_propagatesError(t *testing.T) {
	h := dashboardhandler.NewHandler(&stubGetDashboardStats{err: assert.AnError}, &stubGetTodayQueue{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/dashboard/stats")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

type stubGetTodayQueue struct {
	out *appdashboard.TodayQueue
	err error
}

func (s *stubGetTodayQueue) Execute(_ context.Context) (*appdashboard.TodayQueue, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appdashboard.TodayQueue{}, nil
}

func TestGetTodayQueueHandler_returnsCounts(t *testing.T) {
	h := dashboardhandler.NewHandler(&stubGetDashboardStats{}, &stubGetTodayQueue{out: &appdashboard.TodayQueue{
		ToPick: 4, ToPack: 7, StockIssues: 2, UnmatchedSKUs: 3, ExpiringBatches: 5, ExpiredBatches: 1, ShipToday: 8,
	}})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/dashboard/today")

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data  gendashboard.TodayQueue `json:"data"`
		Error any                     `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Nil(t, resp.Error)
	assert.Equal(t, gendashboard.TodayQueue{
		ToPick: 4, ToPack: 7, StockIssues: 2, UnmatchedSkus: 3, ExpiringBatches: 5, ExpiredBatches: 1, ShipToday: 8,
	}, resp.Data)
}

func TestGetTodayQueueHandler_zeroCountsAreStillSent(t *testing.T) {
	h := dashboardhandler.NewHandler(&stubGetDashboardStats{}, &stubGetTodayQueue{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/dashboard/today")

	require.Equal(t, http.StatusOK, w.Code)
	for _, field := range []string{"to_pick", "to_pack", "stock_issues", "unmatched_skus", "expiring_batches", "expired_batches", "ship_today"} {
		assert.Contains(t, w.Body.String(), `"`+field+`":0`)
	}
}

func TestGetTodayQueueHandler_propagatesError(t *testing.T) {
	h := dashboardhandler.NewHandler(&stubGetDashboardStats{}, &stubGetTodayQueue{err: assert.AnError})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/dashboard/today")

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "INTERNAL")
	assert.NotContains(t, w.Body.String(), assert.AnError.Error(), "internal error text must not leak")
}
