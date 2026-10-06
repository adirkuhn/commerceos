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
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type reorderRecommendationsUC interface {
	Execute(ctx context.Context, in appprocurement.ReorderRecommendationsInput) (*appprocurement.ReorderRecommendations, error)
}

type seasonPlanUC interface {
	Execute(ctx context.Context, in appprocurement.SeasonPlanInput) (*appprocurement.SeasonPlan, error)
}

type seasonEventsUC interface {
	Execute(year int) []appprocurement.SeasonEventOccurrence
}

type planOrderUC interface {
	Execute(ctx context.Context, in appprocurement.PlanOrderInput) (*procurementDomain.PurchaseOrder, error)
}

type UseCases struct {
	Recommendations reorderRecommendationsUC
	SeasonPlan      seasonPlanUC
	SeasonEvents    seasonEventsUC
	PlanOrder       planOrderUC
}

type Handler struct {
	recommendations reorderRecommendationsUC
	seasonPlan      seasonPlanUC
	seasonEvents    seasonEventsUC
	planOrder       planOrderUC
}

func NewHandler(ucs UseCases) *Handler {
	return &Handler{
		recommendations: ucs.Recommendations, seasonPlan: ucs.SeasonPlan,
		seasonEvents: ucs.SeasonEvents, planOrder: ucs.PlanOrder,
	}
}

var _ genpurchaseplanning.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	repo := repositories.NewPurchasePlanningRepo(db)
	genpurchaseplanning.RegisterHandlers(rg, NewHandler(UseCases{
		Recommendations: appprocurement.NewGetReorderRecommendationsUseCase(db, repo, repo, repo, time.Now),
		SeasonPlan:      appprocurement.NewGetSeasonPlanUseCase(db, repo, repo, repo, time.Now),
		SeasonEvents:    appprocurement.NewListSeasonEventsUseCase(time.Now),
		PlanOrder: appprocurement.NewCreatePurchaseOrderFromPlanUseCase(db, appprocurement.PlanOrderDeps{
			Catalog: repo, Supply: repo, Suppliers: repositories.NewSupplierRepo(db),
			Settings:       repositories.NewTenantSettingsRepo(db),
			PurchaseOrders: repositories.NewPurchaseOrderRepo(db), LineItems: repositories.NewPurchaseOrderLineItemRepo(db),
		}),
	}))
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
	case errors.Is(err, suppliersDomain.ErrSupplierNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, appprocurement.ErrInvalidPlanOrder), errors.Is(err, appprocurement.ErrSKUNotPlannable):
		middleware.Fail(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err.Error())
	case errors.Is(err, appprocurement.ErrInvalidPlanningWindow), errors.Is(err, appprocurement.ErrInvalidSeason),
		errors.Is(err, appprocurement.ErrSeasonOver), errors.Is(err, appprocurement.ErrUnknownSeasonEvent):
		middleware.Fail(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err.Error())
	default:
		log.Printf("purchaseplanning: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
