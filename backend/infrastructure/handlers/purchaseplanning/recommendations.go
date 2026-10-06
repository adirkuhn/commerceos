package purchaseplanning

import (
	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"

	genpurchaseplanning "github.com/adirkuhn/commerceos/api/purchaseplanning"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/domain/procurement/planning"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) GetReorderRecommendations(c *gin.Context, params genpurchaseplanning.GetReorderRecommendationsParams) {
	in := appprocurement.ReorderRecommendationsInput{
		CoverageDays: intOr(params.CoverageDays, appprocurement.DefaultPlanningCoverageDays),
		SafetyDays:   intOr(params.SafetyDays, appprocurement.DefaultPlanningSafetyDays),
	}
	if params.SupplierId != nil {
		id := params.SupplierId.String()
		in.SupplierID = &id
	}
	out, err := h.recommendations.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenRecommendations(out))
}

func toGenRecommendations(r *appprocurement.ReorderRecommendations) genpurchaseplanning.ReorderRecommendations {
	return genpurchaseplanning.ReorderRecommendations{
		AsOf: openapi_types.Date{Time: r.AsOf}, CoverageDays: r.CoverageDays, SafetyDays: r.SafetyDays,
		Suppliers: toGenSuppliers(r.Suppliers),
	}
}

func toGenLine(l appprocurement.PlannedLine) genpurchaseplanning.RecommendationLine {
	recent, lastYear := l.RecentUnits, l.LastYearUnits
	return genpurchaseplanning.RecommendationLine{
		SkuId: l.SKUID, SkuCode: l.SKUCode, ProductName: l.ProductName, VariantName: l.VariantName,
		IsWeightPool: l.IsWeightPool, OnHand: l.OnHand,
		OnOrder: genpurchaseplanning.OnOrder{Sent: l.OnOrder.Sent, Draft: l.OnOrder.Draft},
		Forecast: genpurchaseplanning.Forecast{
			Units: l.Forecast.Units, Method: genpurchaseplanning.ForecastMethod(l.Forecast.Method),
			GrowthFactor: float32(l.Forecast.GrowthFactor), Days: l.ForecastDays,
			RecentUnits: &recent, LastYearUnits: &lastYear,
		},
		PreSeason: toGenForecast(l.PreSeason), SeasonalLift: toFloat32(l.SeasonalLift),
		DailyDemand: float32(l.Reorder.DailyDemand), SafetyStock: l.Reorder.SafetyStock,
		NeededUnits: l.Reorder.NeededUnits, PurchaseQuantity: l.Reorder.PurchaseQuantity,
		UnitsPerPurchasedItem: l.UnitsPerPurchasedItem, OrderUnits: l.Reorder.OrderUnits,
		UnitCostMinorUnits: l.UnitCostMinorUnits, EstimatedCostMinorUnits: l.EstimatedCostMinorUnits,
		CurrencyCode: l.CurrencyCode, StockoutDate: toDate(l.Reorder.StockoutDate),
		OrderByDate: toDate(l.Reorder.OrderByDate), LastOrderedAt: l.LastOrderedAt,
	}
}

func toGenSuppliers(plans []appprocurement.SupplierPlan) []genpurchaseplanning.SupplierRecommendation {
	out := make([]genpurchaseplanning.SupplierRecommendation, len(plans))
	for i, s := range plans {
		lines := make([]genpurchaseplanning.RecommendationLine, len(s.Lines))
		for j, l := range s.Lines {
			lines[j] = toGenLine(l)
		}
		out[i] = genpurchaseplanning.SupplierRecommendation{
			SupplierId: s.SupplierID, SupplierName: s.SupplierName,
			LeadTimeDays: s.LeadTimeDays, LeadTimeAssumed: s.LeadTimeAssumed, Lines: lines,
			EstimatedCostMinorUnits: s.EstimatedCostMinorUnits, CurrencyCode: s.CurrencyCode,
			EarliestOrderByDate: toDate(s.EarliestOrderByDate),
		}
	}
	return out
}

// toGenForecast maps a forecast without the line-level history fields
// (a pre-season forecast has none of its own).
func toGenForecast(f *planning.Forecast) *genpurchaseplanning.Forecast {
	if f == nil {
		return nil
	}
	return &genpurchaseplanning.Forecast{
		Units: f.Units, Method: genpurchaseplanning.ForecastMethod(f.Method), GrowthFactor: float32(f.GrowthFactor),
	}
}

func toFloat32(v *float64) *float32 {
	if v == nil {
		return nil
	}
	f := float32(*v)
	return &f
}
