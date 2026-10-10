package pms

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ── Variant handlers ─────────────────────────────────────────────────────────

func (h *Handler) CreateVariant(c *gin.Context, id genpms.Id) {
	var req genpms.CreateVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	position := 0
	if req.Position != nil {
		position = *req.Position
	}
	variant, err := h.createVariant.Execute(c.Request.Context(), apppms.CreateVariantInput{
		ProductID: id,
		Name:      req.Name,
		Position:  position,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenVariant(*variant)
	c.JSON(http.StatusCreated, genpms.VariantResponse{Data: &resp})
}

func (h *Handler) UpdateVariant(c *gin.Context, id genpms.Id) {
	var req genpms.UpdateVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	position := 0
	if req.Position != nil {
		position = *req.Position
	}
	variant, err := h.updateVariant.Execute(c.Request.Context(), apppms.UpdateVariantInput{
		ID:              id,
		Name:            req.Name,
		Position:        position,
		FeaturedMediaID: req.FeaturedMediaId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenVariant(*variant)
	c.JSON(http.StatusOK, genpms.VariantResponse{Data: &resp})
}

func (h *Handler) DeleteVariant(c *gin.Context, id genpms.Id) {
	err := h.deleteVariant.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.EmptyResponse{})
}

func toGenVariant(v pmsDomain.Variant) genpms.Variant {
	return genpms.Variant{
		Id:              v.ID,
		TenantId:        v.TenantID,
		ProductId:       v.ProductID,
		Name:            v.Name,
		Position:        v.Position,
		FeaturedMediaId: v.FeaturedMediaID,
		CreatedAt:       v.CreatedAt,
		UpdatedAt:       v.UpdatedAt,
	}
}
