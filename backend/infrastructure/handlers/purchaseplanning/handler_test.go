package purchaseplanning_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genpurchaseplanning "github.com/adirkuhn/commerceos/api/purchaseplanning"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/domain/procurement/planning"
	purchaseplanninghandler "github.com/adirkuhn/commerceos/infrastructure/handlers/purchaseplanning"
)

func init() { gin.SetMode(gin.TestMode) }

type stubRecommendations struct {
	out    *appprocurement.ReorderRecommendations
	err    error
	got    appprocurement.ReorderRecommendationsInput
	called bool
}

func (s *stubRecommendations) Execute(_ context.Context, in appprocurement.ReorderRecommendationsInput) (*appprocurement.ReorderRecommendations, error) {
	s.got, s.called = in, true
	return s.out, s.err
}

type stubSeasonPlan struct {
	out    *appprocurement.SeasonPlan
	err    error
	got    appprocurement.SeasonPlanInput
	called bool
}

func (s *stubSeasonPlan) Execute(_ context.Context, in appprocurement.SeasonPlanInput) (*appprocurement.SeasonPlan, error) {
	s.got, s.called = in, true
	return s.out, s.err
}

type stubSeasonEvents struct {
	out []appprocurement.SeasonEventOccurrence
	got int
}

func (s *stubSeasonEvents) Execute(year int) []appprocurement.SeasonEventOccurrence {
	s.got = year
	return s.out
}

// newTestRouter mounts the handler behind the same OpenAPI request
// validator cmd/api/main.go uses, so the contract itself is under test.
func newTestRouter(t *testing.T, uc *stubRecommendations) *gin.Engine {
	t.Helper()
	return newSeasonRouter(t, uc, &stubSeasonPlan{}, &stubSeasonEvents{})
}

func newSeasonRouter(t *testing.T, uc *stubRecommendations, plan *stubSeasonPlan, events *stubSeasonEvents) *gin.Engine {
	t.Helper()
	spec, err := genpurchaseplanning.GetSpec()
	require.NoError(t, err)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(ginmiddleware.OapiRequestValidatorWithOptions(spec, &ginmiddleware.Options{SilenceServersWarning: true}))
	genpurchaseplanning.RegisterHandlers(g, purchaseplanninghandler.NewHandler(uc, plan, events))
	return r
}

func get(r *gin.Engine, query string) *httptest.ResponseRecorder {
	return getPath(r, "/api/v1/procurement/recommendations"+query)
}

func getPath(r *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestGetReorderRecommendations_MapsResult(t *testing.T) {
	asOf := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	orderBy := asOf.AddDate(0, 0, 3)
	supplierID := "sup-1"
	cost, total := int64(450), int64(450*172)
	uc := &stubRecommendations{out: &appprocurement.ReorderRecommendations{
		AsOf: asOf, CoverageDays: 30, SafetyDays: 7,
		Suppliers: []appprocurement.SupplierPlan{{
			SupplierID: &supplierID, SupplierName: "Bauducco NL", LeadTimeDays: 21,
			EstimatedCostMinorUnits: &total, CurrencyCode: "EUR", EarliestOrderByDate: &orderBy,
			Lines: []appprocurement.PlannedLine{{
				SKUID: "sku-1", SKUCode: "PAN-1", ProductName: "Panettone", VariantName: "Default",
				OnHand: 12, OnOrder: planning.OpenOrderQuantity{Sent: 50, Draft: 8},
				RecentUnits: 118, LastYearUnits: 820, ForecastDays: 51,
				Forecast: planning.Forecast{Units: 968, Method: planning.ForecastMethodSeasonal, GrowthFactor: 1.18},
				Reorder: planning.Reorder{
					DailyDemand: 18.98, SafetyStock: 133, NeededUnits: 1031, PurchaseQuantity: 172, OrderUnits: 1032,
					StockoutDate: &orderBy, OrderByDate: &orderBy,
				},
				UnitsPerPurchasedItem: 6, UnitCostMinorUnits: &cost, EstimatedCostMinorUnits: &total, CurrencyCode: "EUR",
			}},
		}},
	}}

	w := get(newTestRouter(t, uc), "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body struct {
		Data genpurchaseplanning.ReorderRecommendations `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "2026-10-06", body.Data.AsOf.String())
	require.Len(t, body.Data.Suppliers, 1)
	group := body.Data.Suppliers[0]
	assert.Equal(t, "sup-1", *group.SupplierId)
	assert.Equal(t, int64(77400), *group.EstimatedCostMinorUnits)
	line := group.Lines[0]
	assert.Equal(t, "PAN-1", line.SkuCode)
	assert.Equal(t, genpurchaseplanning.Seasonal, line.Forecast.Method)
	assert.Equal(t, int64(968), line.Forecast.Units)
	assert.Equal(t, 51, line.Forecast.Days)
	assert.Equal(t, int64(820), *line.Forecast.LastYearUnits)
	assert.Equal(t, int64(118), *line.Forecast.RecentUnits)
	assert.Equal(t, genpurchaseplanning.OnOrder{Sent: 50, Draft: 8}, line.OnOrder)
	assert.Equal(t, int64(172), line.PurchaseQuantity)
	assert.Equal(t, "2026-10-09", line.OrderByDate.String())
}

func TestGetReorderRecommendations_DefaultsAndFilters(t *testing.T) {
	uc := &stubRecommendations{out: &appprocurement.ReorderRecommendations{}}
	r := newTestRouter(t, uc)

	require.Equal(t, http.StatusOK, get(r, "").Code)
	assert.Equal(t, appprocurement.ReorderRecommendationsInput{
		CoverageDays: appprocurement.DefaultPlanningCoverageDays, SafetyDays: appprocurement.DefaultPlanningSafetyDays,
	}, uc.got)

	supplierID := "6f1c0a6e-0d4b-4b55-9a51-1c3f1d0f7a11"
	require.Equal(t, http.StatusOK, get(r, fmt.Sprintf("?coverage_days=60&safety_days=0&supplier_id=%s", supplierID)).Code)
	assert.Equal(t, 60, uc.got.CoverageDays)
	assert.Equal(t, 0, uc.got.SafetyDays)
	require.NotNil(t, uc.got.SupplierID)
	assert.Equal(t, supplierID, *uc.got.SupplierID)
}

// Empty result is an empty list, never null.
func TestGetReorderRecommendations_EmptySuppliersIsAList(t *testing.T) {
	w := get(newTestRouter(t, &stubRecommendations{out: &appprocurement.ReorderRecommendations{}}), "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"suppliers":[]`)
}

func TestGetReorderRecommendations_ContractRejectsBadQuery(t *testing.T) {
	for _, q := range []string{"?coverage_days=0", "?coverage_days=366", "?safety_days=-1", "?safety_days=91", "?supplier_id=nope"} {
		uc := &stubRecommendations{}
		w := get(newTestRouter(t, uc), q)
		assert.Equal(t, http.StatusBadRequest, w.Code, q)
		assert.False(t, uc.called, q)
	}
}

func TestGetReorderRecommendations_MapsErrors(t *testing.T) {
	cases := map[error]int{
		fmt.Errorf("x: %w", appprocurement.ErrInvalidPlanningWindow): http.StatusUnprocessableEntity,
		errors.New("db down"): http.StatusInternalServerError,
	}
	for err, status := range cases {
		w := get(newTestRouter(t, &stubRecommendations{err: err}), "")
		assert.Equal(t, status, w.Code, err.Error())
		assert.NotContains(t, w.Body.String(), "db down", "internal errors don't leak")
	}
}
