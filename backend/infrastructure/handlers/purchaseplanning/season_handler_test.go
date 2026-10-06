package purchaseplanning_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genpurchaseplanning "github.com/adirkuhn/commerceos/api/purchaseplanning"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/domain/procurement/planning"
)

func d(y int, m time.Month, day int) time.Time { return time.Date(y, m, day, 0, 0, 0, 0, time.UTC) }

func TestListSeasonEvents_MapsOccurrences(t *testing.T) {
	events := &stubSeasonEvents{out: []appprocurement.SeasonEventOccurrence{{
		Key: planning.SeasonChristmas, Year: 2026, Date: d(2026, time.December, 25),
		Window: planning.Window{Start: d(2026, time.November, 13), End: d(2026, time.December, 27)},
	}}}
	r := newSeasonRouter(t, &stubRecommendations{}, &stubSeasonPlan{}, events)

	w := getPath(r, "/api/v1/procurement/season-events?year=2026")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 2026, events.got)
	var body struct {
		Data []genpurchaseplanning.SeasonEvent `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, genpurchaseplanning.Christmas, body.Data[0].Key)
	assert.Equal(t, "2026-12-25", body.Data[0].Date.String())
	assert.Equal(t, "2026-11-13", body.Data[0].Window.Start.String())
	assert.Equal(t, "2026-12-27", body.Data[0].Window.End.String())
}

func TestListSeasonEvents_DefaultsToNextOccurrence(t *testing.T) {
	events := &stubSeasonEvents{}
	w := getPath(newSeasonRouter(t, &stubRecommendations{}, &stubSeasonPlan{}, events), "/api/v1/procurement/season-events")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, events.got)
	assert.Contains(t, w.Body.String(), `"data":[]`)
}

func TestGetSeasonPlan_EventQuery(t *testing.T) {
	key := planning.SeasonChristmas
	lift := 18.6
	plan := &stubSeasonPlan{out: &appprocurement.SeasonPlan{
		AsOf: d(2026, time.October, 6), Event: &key, Year: 2026, SafetyDays: 7,
		Window:         planning.Window{Start: d(2026, time.November, 13), End: d(2026, time.December, 27)},
		LastYearWindow: planning.Window{Start: d(2025, time.November, 13), End: d(2025, time.December, 27)},
		Suppliers: []appprocurement.SupplierPlan{{SupplierName: "", LeadTimeDays: 14, LeadTimeAssumed: true, Lines: []appprocurement.PlannedLine{{
			SKUID: "sku-1", PreSeason: &planning.Forecast{Units: 48, Method: planning.ForecastMethodSeasonal, GrowthFactor: 1.18}, SeasonalLift: &lift,
			Forecast: planning.Forecast{Units: 968, Method: planning.ForecastMethodSeasonal, GrowthFactor: 1.18}, ForecastDays: 44,
		}}}},
	}}
	r := newSeasonRouter(t, &stubRecommendations{}, plan, &stubSeasonEvents{})

	w := getPath(r, "/api/v1/procurement/season-plan?event=christmas&year=2026&all_products=true")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, plan.got.Event)
	assert.Equal(t, planning.SeasonChristmas, *plan.got.Event)
	assert.Equal(t, 2026, plan.got.Year)
	assert.True(t, plan.got.AllProducts)
	assert.Equal(t, appprocurement.DefaultPlanningSafetyDays, plan.got.SafetyDays)
	assert.Nil(t, plan.got.Custom)

	var body struct {
		Data genpurchaseplanning.SeasonPlan `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, genpurchaseplanning.Christmas, *body.Data.Event)
	assert.Equal(t, "2026-11-13", body.Data.Window.Start.String())
	assert.Equal(t, "2025-11-13", body.Data.LastYearWindow.Start.String())
	line := body.Data.Suppliers[0].Lines[0]
	require.NotNil(t, line.PreSeason)
	assert.Equal(t, int64(48), line.PreSeason.Units)
	assert.InDelta(t, 18.6, *line.SeasonalLift, 0.001)
}

func TestGetSeasonPlan_CustomRangeQuery(t *testing.T) {
	plan := &stubSeasonPlan{out: &appprocurement.SeasonPlan{}}
	r := newSeasonRouter(t, &stubRecommendations{}, plan, &stubSeasonEvents{})

	w := getPath(r, "/api/v1/procurement/season-plan?from=2026-11-01&to=2026-11-11&safety_days=0")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, plan.got.Custom)
	assert.Equal(t, d(2026, time.November, 1), plan.got.Custom.Start)
	assert.Equal(t, d(2026, time.November, 11), plan.got.Custom.End)
	assert.Nil(t, plan.got.Event)
	assert.Equal(t, 0, plan.got.SafetyDays)
	assert.Contains(t, w.Body.String(), `"suppliers":[]`)
}

// Only one of from/to can't be a range.
func TestGetSeasonPlan_HalfARangeIsInvalid(t *testing.T) {
	plan := &stubSeasonPlan{}
	w := getPath(newSeasonRouter(t, &stubRecommendations{}, plan, &stubSeasonEvents{}), "/api/v1/procurement/season-plan?from=2026-11-01")
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.False(t, plan.called)
}

func TestGetSeasonPlan_ContractRejectsBadQuery(t *testing.T) {
	for _, q := range []string{"?event=halloween", "?event=christmas&year=1999", "?from=nope&to=2026-11-01", "?event=christmas&safety_days=91"} {
		plan := &stubSeasonPlan{}
		w := getPath(newSeasonRouter(t, &stubRecommendations{}, plan, &stubSeasonEvents{}), "/api/v1/procurement/season-plan"+q)
		assert.Equal(t, http.StatusBadRequest, w.Code, q)
		assert.False(t, plan.called, q)
	}
}

func TestGetSeasonPlan_MapsErrors(t *testing.T) {
	for _, err := range []error{appprocurement.ErrInvalidSeason, appprocurement.ErrSeasonOver, appprocurement.ErrUnknownSeasonEvent, appprocurement.ErrInvalidPlanningWindow} {
		w := getPath(newSeasonRouter(t, &stubRecommendations{}, &stubSeasonPlan{err: fmt.Errorf("x: %w", err)}, &stubSeasonEvents{}), "/api/v1/procurement/season-plan?event=christmas")
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code, err.Error())
	}
}
