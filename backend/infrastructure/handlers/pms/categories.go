package pms

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ── Category handlers ─────────────────────────────────────────────────────────

func (h *Handler) ListCategories(c *gin.Context) {
	categories, err := h.listCategories.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genpms.Category, len(categories))
	for i, cat := range categories {
		resp[i] = toGenCategory(cat)
	}
	c.JSON(http.StatusOK, genpms.CategoryListResponse{Data: &resp})
}

func (h *Handler) CreateCategory(c *gin.Context) {
	var req genpms.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	category, err := h.createCategory.Execute(c.Request.Context(), apppms.CreateCategoryInput{
		Name:     req.Name,
		ParentID: req.ParentId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenCategory(*category)
	c.JSON(http.StatusCreated, genpms.CategoryResponse{Data: &resp})
}

func (h *Handler) UpdateCategory(c *gin.Context, id genpms.Id) {
	var req genpms.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	category, err := h.updateCategory.Execute(c.Request.Context(), apppms.UpdateCategoryInput{
		ID:                      id,
		Name:                    req.Name,
		ParentID:                req.ParentId,
		TargetMarkupBasisPoints: req.TargetMarkupBasisPoints,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenCategory(*category)
	c.JSON(http.StatusOK, genpms.CategoryResponse{Data: &resp})
}

func (h *Handler) DeleteCategory(c *gin.Context, id genpms.Id) {
	err := h.deleteCategory.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.EmptyResponse{})
}

// ── Domain → generated-type mappers ─────────────────────────────────────────

func toGenCategory(c pmsDomain.Category) genpms.Category {
	return genpms.Category{
		Id:                      c.ID,
		TenantId:                c.TenantID,
		ParentId:                c.ParentID,
		Name:                    c.Name,
		Slug:                    c.Slug,
		Position:                c.Position,
		TargetMarkupBasisPoints: c.TargetMarkupBasisPoints,
		CreatedAt:               c.CreatedAt,
		UpdatedAt:               c.UpdatedAt,
	}
}
