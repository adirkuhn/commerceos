// Package purchaseplanning wires the admin-only (JWT) Purchase Planner API
// (see api/purchaseplanning's doc comment) to its use cases.
package purchaseplanning

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/gorm"

	genpurchaseplanning "github.com/adirkuhn/commerceos/api/purchaseplanning"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type reorderRecommendationsUC interface {
	Execute(ctx context.Context, in appprocurement.ReorderRecommendationsInput) (*appprocurement.ReorderRecommendations, error)
}

type Handler struct {
	recommendations reorderRecommendationsUC
}

func NewHandler(recommendations reorderRecommendationsUC) *Handler {
	return &Handler{recommendations: recommendations}
}

var _ genpurchaseplanning.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	repo := repositories.NewPurchasePlanningRepo(db)
	genpurchaseplanning.RegisterHandlers(rg, NewHandler(
		appprocurement.NewGetReorderRecommendationsUseCase(db, repo, repo, repo, time.Now),
	))
}

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
	out := genpurchaseplanning.ReorderRecommendations{
		AsOf: openapi_types.Date{Time: r.AsOf}, CoverageDays: r.CoverageDays, SafetyDays: r.SafetyDays,
		Suppliers: make([]genpurchaseplanning.SupplierRecommendation, len(r.Suppliers)),
	}
	for i, s := range r.Suppliers {
		lines := make([]genpurchaseplanning.RecommendationLine, len(s.Lines))
		for j, l := range s.Lines {
			lines[j] = toGenLine(l)
		}
		out.Suppliers[i] = genpurchaseplanning.SupplierRecommendation{
			SupplierId: s.SupplierID, SupplierName: s.SupplierName,
			LeadTimeDays: s.LeadTimeDays, LeadTimeAssumed: s.LeadTimeAssumed, Lines: lines,
			EstimatedCostMinorUnits: s.EstimatedCostMinorUnits, CurrencyCode: s.CurrencyCode,
			EarliestOrderByDate: toDate(s.EarliestOrderByDate),
		}
	}
	return out
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
		DailyDemand: float32(l.Reorder.DailyDemand), SafetyStock: l.Reorder.SafetyStock,
		NeededUnits: l.Reorder.NeededUnits, PurchaseQuantity: l.Reorder.PurchaseQuantity,
		UnitsPerPurchasedItem: l.UnitsPerPurchasedItem, OrderUnits: l.Reorder.OrderUnits,
		UnitCostMinorUnits: l.UnitCostMinorUnits, EstimatedCostMinorUnits: l.EstimatedCostMinorUnits,
		CurrencyCode: l.CurrencyCode, StockoutDate: toDate(l.Reorder.StockoutDate),
		OrderByDate: toDate(l.Reorder.OrderByDate), LastOrderedAt: l.LastOrderedAt,
	}
}

func toDate(t *time.Time) *openapi_types.Date {
	if t == nil {
		return nil
	}
	return &openapi_types.Date{Time: *t}
}

func intOr(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, appprocurement.ErrInvalidPlanningWindow):
		middleware.Fail(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err.Error())
	default:
		log.Printf("purchaseplanning: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
