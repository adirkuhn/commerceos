package promotions

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genpromotions "github.com/adirkuhn/commerceos/api/promotions"
	apppromotions "github.com/adirkuhn/commerceos/application/promotions"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type createPromotionUC interface {
	Execute(ctx context.Context, in apppromotions.CreatePromotionInput) (*promotionsDomain.Promotion, error)
}
type updatePromotionUC interface {
	Execute(ctx context.Context, in apppromotions.UpdatePromotionInput) (*promotionsDomain.Promotion, error)
}
type deletePromotionUC interface {
	Execute(ctx context.Context, id string) error
}
type getPromotionUC interface {
	Execute(ctx context.Context, id string) (*apppromotions.GetPromotionResult, error)
}
type listPromotionsUC interface {
	Execute(ctx context.Context) ([]promotionsDomain.Promotion, error)
}

type Handler struct {
	createPromotion createPromotionUC
	updatePromotion updatePromotionUC
	deletePromotion deletePromotionUC
	getPromotion    getPromotionUC
	listPromotions  listPromotionsUC
}

func NewHandler(
	createPromotion createPromotionUC,
	updatePromotion updatePromotionUC,
	deletePromotion deletePromotionUC,
	getPromotion getPromotionUC,
	listPromotions listPromotionsUC,
) *Handler {
	return &Handler{
		createPromotion: createPromotion, updatePromotion: updatePromotion, deletePromotion: deletePromotion,
		getPromotion: getPromotion, listPromotions: listPromotions,
	}
}

var _ genpromotions.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	promoRepo := repositories.NewPromotionRepo(db)
	lineRepo := repositories.NewPromotionSKURepo(db)
	skuRepo := repositories.NewSKURepo(db)
	h := NewHandler(
		apppromotions.NewCreatePromotionUseCase(db, promoRepo, lineRepo, skuRepo),
		apppromotions.NewUpdatePromotionUseCase(db, promoRepo, lineRepo, skuRepo),
		apppromotions.NewDeletePromotionUseCase(db, promoRepo),
		apppromotions.NewGetPromotionUseCase(db, promoRepo, lineRepo, skuRepo),
		apppromotions.NewListPromotionsUseCase(db, promoRepo),
	)
	genpromotions.RegisterHandlers(rg, h)
}

func (h *Handler) ListPromotions(c *gin.Context) {
	promos, err := h.listPromotions.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenPromotions(promos))
}

func (h *Handler) CreatePromotion(c *gin.Context) {
	var req genpromotions.CreatePromotionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	promo, err := h.createPromotion.Execute(c.Request.Context(), apppromotions.CreatePromotionInput{
		Name: req.Name, Type: promotionsDomain.PromotionType(req.Type), StartsAt: req.StartsAt,
		EndsAt: req.EndsAt, Status: promotionsDomain.PromotionStatus(genStatusOr(req.Status)),
		Lines: toLineInputs(req.Lines),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genpromotions.PromotionDetail{
		Id: promo.ID, Name: promo.Name, Type: genpromotions.PromotionType(promo.Type),
		StartsAt: promo.StartsAt, EndsAt: promo.EndsAt, Status: genpromotions.PromotionStatus(promo.Status),
		CreatedAt: promo.CreatedAt, UpdatedAt: promo.UpdatedAt,
	})
}

func (h *Handler) GetPromotion(c *gin.Context, id genpromotions.Id) {
	result, err := h.getPromotion.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenPromotionDetail(result.Promotion, result.Lines))
}

func (h *Handler) UpdatePromotion(c *gin.Context, id genpromotions.Id) {
	var req genpromotions.UpdatePromotionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	promo, err := h.updatePromotion.Execute(c.Request.Context(), apppromotions.UpdatePromotionInput{
		ID: id, Name: req.Name, Type: promotionsDomain.PromotionType(req.Type), StartsAt: req.StartsAt,
		EndsAt: req.EndsAt, Status: promotionsDomain.PromotionStatus(req.Status),
		Lines: toLineInputs(req.Lines),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genpromotions.PromotionDetail{
		Id: promo.ID, Name: promo.Name, Type: genpromotions.PromotionType(promo.Type),
		StartsAt: promo.StartsAt, EndsAt: promo.EndsAt, Status: genpromotions.PromotionStatus(promo.Status),
		CreatedAt: promo.CreatedAt, UpdatedAt: promo.UpdatedAt,
	})
}

func (h *Handler) DeletePromotion(c *gin.Context, id genpromotions.Id) {
	if err := mapError(c, h.deletePromotion.Execute(c.Request.Context(), id)); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func genStatusOr(s *genpromotions.PromotionStatus) genpromotions.PromotionStatus {
	if s == nil {
		return ""
	}
	return *s
}

func toLineInputs(lines []genpromotions.PromotionLineInput) []apppromotions.PromotionLineInput {
	out := make([]apppromotions.PromotionLineInput, len(lines))
	for i, l := range lines {
		out[i] = apppromotions.PromotionLineInput{
			SKUID: l.SkuId, DiscountType: promotionsDomain.PromotionType(l.DiscountType), DiscountValue: l.DiscountValue,
		}
	}
	return out
}

func toGenPromotion(p promotionsDomain.Promotion) genpromotions.Promotion {
	return genpromotions.Promotion{
		Id: p.ID, Name: p.Name, Type: genpromotions.PromotionType(p.Type),
		StartsAt: p.StartsAt, EndsAt: p.EndsAt, Status: genpromotions.PromotionStatus(p.Status),
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toGenPromotions(promos []promotionsDomain.Promotion) []genpromotions.Promotion {
	out := make([]genpromotions.Promotion, len(promos))
	for i, p := range promos {
		out[i] = toGenPromotion(p)
	}
	return out
}

func toGenPromotionDetail(p promotionsDomain.Promotion, lines []apppromotions.PromotionLineView) genpromotions.PromotionDetail {
	genLines := make([]genpromotions.PromotionLine, len(lines))
	for i, l := range lines {
		genLines[i] = genpromotions.PromotionLine{
			Id: l.ID, SkuId: l.SKUID, SkuCode: l.SKUCode, DiscountType: genpromotions.PromotionType(l.DiscountType), DiscountValue: l.DiscountValue,
		}
	}
	return genpromotions.PromotionDetail{
		Id: p.ID, Name: p.Name, Type: genpromotions.PromotionType(p.Type),
		StartsAt: p.StartsAt, EndsAt: p.EndsAt, Status: genpromotions.PromotionStatus(p.Status),
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, Lines: &genLines,
	}
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, promotionsDomain.ErrPromotionNotFound),
		errors.Is(err, pmsDomain.ErrSKUNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, promotionsDomain.ErrPromotionNameRequired),
		errors.Is(err, promotionsDomain.ErrInvalidPromotionType),
		errors.Is(err, promotionsDomain.ErrInvalidPromotionStatus),
		errors.Is(err, promotionsDomain.ErrPromotionEndBeforeStart),
		errors.Is(err, promotionsDomain.ErrPromotionRequiresAtLeastOneSKU),
		errors.Is(err, promotionsDomain.ErrDuplicatePromotionSKU),
		errors.Is(err, promotionsDomain.ErrInvalidDiscountValue):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		log.Printf("promotions: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
