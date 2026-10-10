package purchaseplanning

import (
	"github.com/adirkuhn/commerceos/infrastructure/httperrs"
	"net/http"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"

	genpurchaseplanning "github.com/adirkuhn/commerceos/api/purchaseplanning"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/domain/procurement/planning"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) ListSeasonEvents(c *gin.Context, params genpurchaseplanning.ListSeasonEventsParams) {
	occurrences := h.seasonEvents.Execute(intOr(params.Year, 0))
	out := make([]genpurchaseplanning.SeasonEvent, len(occurrences))
	for i, o := range occurrences {
		out[i] = genpurchaseplanning.SeasonEvent{
			Key: genpurchaseplanning.SeasonEventKey(o.Key), Year: o.Year,
			Date: openapi_types.Date{Time: o.Date}, Window: toGenWindow(o.Window),
		}
	}
	middleware.OK(c, out)
}

func (h *Handler) GetSeasonPlan(c *gin.Context, params genpurchaseplanning.GetSeasonPlanParams) {
	if (params.From == nil) != (params.To == nil) {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", httperrs.ErrDateRangeIncomplete)
		return
	}
	in := appprocurement.SeasonPlanInput{
		Year:        intOr(params.Year, 0),
		SafetyDays:  intOr(params.SafetyDays, appprocurement.DefaultPlanningSafetyDays),
		AllProducts: params.AllProducts != nil && *params.AllProducts,
	}
	if params.Event != nil {
		key := planning.SeasonEventKey(*params.Event)
		in.Event = &key
	}
	if params.From != nil {
		in.Custom = &planning.Window{Start: params.From.Time, End: params.To.Time}
	}
	if params.SupplierId != nil {
		id := params.SupplierId.String()
		in.SupplierID = &id
	}
	plan, err := h.seasonPlan.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	out := genpurchaseplanning.SeasonPlan{
		AsOf: openapi_types.Date{Time: plan.AsOf}, Year: plan.Year, SafetyDays: plan.SafetyDays,
		Window: toGenWindow(plan.Window), LastYearWindow: toGenWindow(plan.LastYearWindow),
		Suppliers: toGenSuppliers(plan.Suppliers),
	}
	if plan.Event != nil {
		key := genpurchaseplanning.SeasonEventKey(*plan.Event)
		out.Event = &key
	}
	middleware.OK(c, out)
}

func toGenWindow(w planning.Window) genpurchaseplanning.DateWindow {
	return genpurchaseplanning.DateWindow{Start: openapi_types.Date{Time: w.Start}, End: openapi_types.Date{Time: w.End}}
}
