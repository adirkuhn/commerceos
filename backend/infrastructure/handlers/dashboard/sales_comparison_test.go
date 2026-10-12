package dashboard_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appdashboard "github.com/adirkuhn/commerceos/application/dashboard"
	dashboardDomain "github.com/adirkuhn/commerceos/domain/dashboard"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	dashboardhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/dashboard"
)

type stubGetSalesComparison struct {
	out       *appdashboard.SalesComparison
	err       error
	gotPeriod dashboardDomain.SalesPeriod
}

func (s *stubGetSalesComparison) Execute(_ context.Context, period dashboardDomain.SalesPeriod) (*appdashboard.SalesComparison, error) {
	s.gotPeriod = period
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appdashboard.SalesComparison{Period: period, CurrencyCode: "USD"}, nil
}

func TestGetSalesComparisonHandler_returnsTheComparison(t *testing.T) {
	from := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	change := 20.0
	uc := &stubGetSalesComparison{out: &appdashboard.SalesComparison{
		Period: dashboardDomain.SalesPeriodThisWeek, CurrencyCode: "EUR", ChangePercent: &change,
		Current: appdashboard.SalesFigures{
			Range: dashboardDomain.DateRange{From: from, To: from.Add(57 * time.Hour)}, RevenueMinorUnits: 12000, OrderCount: 6,
			ByChannel: []omsDomain.ChannelRevenue{{Source: omsDomain.SalesOrderSourcePOS, TotalMinorUnits: 12000}},
		},
		Previous: appdashboard.SalesFigures{
			Range: dashboardDomain.DateRange{From: from.AddDate(0, 0, -7), To: from}, RevenueMinorUnits: 10000, OrderCount: 5,
			ByChannel: []omsDomain.ChannelRevenue{},
		},
	}}
	h := dashboardhandler.NewHandler(&stubGetDashboardStats{}, &stubGetTodayQueue{}, uc)

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/dashboard/sales?period=this_week")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, dashboardDomain.SalesPeriodThisWeek, uc.gotPeriod)
	var body struct {
		Data struct {
			Period        string   `json:"period"`
			CurrencyCode  string   `json:"currency_code"`
			ChangePercent *float64 `json:"change_percent"`
			Current       struct {
				From              time.Time `json:"from"`
				RevenueMinorUnits int64     `json:"revenue_minor_units"`
				OrderCount        int64     `json:"order_count"`
				ByChannel         []struct {
					Source          string `json:"source"`
					TotalMinorUnits int64  `json:"total_minor_units"`
				} `json:"by_channel"`
			} `json:"current"`
			Previous struct {
				RevenueMinorUnits int64             `json:"revenue_minor_units"`
				ByChannel         []json.RawMessage `json:"by_channel"`
			} `json:"previous"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "this_week", body.Data.Period)
	assert.Equal(t, "EUR", body.Data.CurrencyCode)
	require.NotNil(t, body.Data.ChangePercent)
	assert.Equal(t, 20.0, *body.Data.ChangePercent)
	assert.True(t, body.Data.Current.From.Equal(from))
	assert.Equal(t, int64(12000), body.Data.Current.RevenueMinorUnits)
	assert.Equal(t, int64(6), body.Data.Current.OrderCount)
	require.Len(t, body.Data.Current.ByChannel, 1)
	assert.Equal(t, "pos", body.Data.Current.ByChannel[0].Source)
	assert.Equal(t, int64(10000), body.Data.Previous.RevenueMinorUnits)
	assert.NotNil(t, body.Data.Previous.ByChannel)
}

func TestGetSalesComparisonHandler_nullChangePercentWithoutPreviousSales(t *testing.T) {
	h := dashboardhandler.NewHandler(&stubGetDashboardStats{}, &stubGetTodayQueue{}, &stubGetSalesComparison{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/dashboard/sales?period=today")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"change_percent":null`)
}

func TestGetSalesComparisonHandler_rejectsAnUnknownPeriod(t *testing.T) {
	uc := &stubGetSalesComparison{}
	h := dashboardhandler.NewHandler(&stubGetDashboardStats{}, &stubGetTodayQueue{}, uc)

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/dashboard/sales?period=this_year")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "SALES_PERIOD_INVALID")
	assert.Empty(t, uc.gotPeriod, "use case not called")
}

func TestGetSalesComparisonHandler_requiresAPeriod(t *testing.T) {
	h := dashboardhandler.NewHandler(&stubGetDashboardStats{}, &stubGetTodayQueue{}, &stubGetSalesComparison{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/dashboard/sales")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetSalesComparisonHandler_propagatesError(t *testing.T) {
	h := dashboardhandler.NewHandler(&stubGetDashboardStats{}, &stubGetTodayQueue{}, &stubGetSalesComparison{err: assert.AnError})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/dashboard/sales?period=this_month")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}
