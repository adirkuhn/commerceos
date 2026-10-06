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

type seasonPlanUC interface {
	Execute(ctx context.Context, in appprocurement.SeasonPlanInput) (*appprocurement.SeasonPlan, error)
}

type seasonEventsUC interface {
	Execute(year int) []appprocurement.SeasonEventOccurrence
}

type Handler struct {
	recommendations reorderRecommendationsUC
	seasonPlan      seasonPlanUC
	seasonEvents    seasonEventsUC
}

func NewHandler(recommendations reorderRecommendationsUC, seasonPlan seasonPlanUC, seasonEvents seasonEventsUC) *Handler {
	return &Handler{recommendations: recommendations, seasonPlan: seasonPlan, seasonEvents: seasonEvents}
}

var _ genpurchaseplanning.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	repo := repositories.NewPurchasePlanningRepo(db)
	genpurchaseplanning.RegisterHandlers(rg, NewHandler(
		appprocurement.NewGetReorderRecommendationsUseCase(db, repo, repo, repo, time.Now),
		appprocurement.NewGetSeasonPlanUseCase(db, repo, repo, repo, time.Now),
		appprocurement.NewListSeasonEventsUseCase(time.Now),
	))
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
	case errors.Is(err, appprocurement.ErrInvalidPlanningWindow), errors.Is(err, appprocurement.ErrInvalidSeason),
		errors.Is(err, appprocurement.ErrSeasonOver), errors.Is(err, appprocurement.ErrUnknownSeasonEvent):
		middleware.Fail(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err.Error())
	default:
		log.Printf("purchaseplanning: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
