// Package collections serves the admin Collections API (api/collections).
package collections

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	gencollections "github.com/adirkuhn/commerceos/api/collections"
	appcollections "github.com/adirkuhn/commerceos/application/collections"
	"github.com/adirkuhn/commerceos/domain/collections"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type UseCases struct {
	List interface {
		Execute(ctx context.Context, publishedOnly bool, page, perPage int) ([]collections.Collection, int64, error)
	}
	Get interface {
		Execute(ctx context.Context, id string) (*appcollections.CollectionDetail, error)
	}
	Create interface {
		Execute(ctx context.Context, d collections.Draft) (*collections.Collection, error)
	}
	Update interface {
		Execute(ctx context.Context, id string, d collections.Draft) (*collections.Collection, error)
	}
	Delete interface {
		Execute(ctx context.Context, id string) error
	}
	SetProducts interface {
		Execute(ctx context.Context, id string, productIDs []string) error
	}
}

type Handler struct{ uc UseCases }

func NewHandler(uc UseCases) *Handler { return &Handler{uc: uc} }

var _ gencollections.ServerInterface = (*Handler)(nil)

// Register mounts the API on a JWT group: reads for every role, writes for
// owner/admin.
func Register(rg gin.IRouter, h *Handler) {
	g := rg.Group("")
	g.Use(middleware.RequireRoleForWrites(middleware.RoleCustomerOwner, middleware.RoleCustomerAdmin, middleware.RoleSuperAdmin))
	gencollections.RegisterHandlers(g, h)
}

// NewUseCases wires the use cases on the real adapters.
func NewUseCases(db *gorm.DB) UseCases {
	deps := appcollections.Deps{DB: db, Repo: repositories.NewCollectionRepo(db), Outbox: repositories.NewOutboxRepository(db), Now: time.Now}
	return UseCases{
		List: appcollections.NewListCollectionsUseCase(deps), Get: appcollections.NewGetCollectionUseCase(deps),
		Create: appcollections.NewCreateCollectionUseCase(deps), Update: appcollections.NewUpdateCollectionUseCase(deps),
		Delete: appcollections.NewDeleteCollectionUseCase(deps), SetProducts: appcollections.NewSetCollectionProductsUseCase(deps),
	}
}

func fail(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, collections.ErrCollectionNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, collections.ErrSlugTaken), errors.Is(err, collections.ErrNotManual):
		middleware.Fail(c, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, collections.ErrInvalidCollection):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		log.Printf("collections: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return true
}

func (h *Handler) ListCollections(c *gin.Context, params gencollections.ListCollectionsParams) {
	page, perPage := 1, 20
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	list, total, err := h.uc.List.Execute(c.Request.Context(), false, page, perPage)
	if fail(c, err) {
		return
	}
	data := make([]gencollections.Collection, len(list))
	for i, col := range list {
		data[i] = toGen(col, nil)
	}
	t := int(total)
	c.JSON(http.StatusOK, gencollections.CollectionListResponse{Data: &data, Total: &t})
}

func (h *Handler) GetCollection(c *gin.Context, id uuid.UUID) {
	d, err := h.uc.Get.Execute(c.Request.Context(), id.String())
	if fail(c, err) {
		return
	}
	middleware.OK(c, toGen(d.Collection, d.ProductIDs))
}

func bindDraft(c *gin.Context) (collections.Draft, bool) {
	var req gencollections.CollectionInput
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return collections.Draft{}, false
	}
	return fromInput(req), true
}

func (h *Handler) CreateCollection(c *gin.Context) {
	d, ok := bindDraft(c)
	if !ok {
		return
	}
	col, err := h.uc.Create.Execute(c.Request.Context(), d)
	if fail(c, err) {
		return
	}
	c.JSON(http.StatusCreated, middleware.APIResponse{Data: toGen(*col, nil)})
}

func (h *Handler) UpdateCollection(c *gin.Context, id uuid.UUID) {
	d, ok := bindDraft(c)
	if !ok {
		return
	}
	col, err := h.uc.Update.Execute(c.Request.Context(), id.String(), d)
	if fail(c, err) {
		return
	}
	middleware.OK(c, toGen(*col, nil))
}

func (h *Handler) DeleteCollection(c *gin.Context, id uuid.UUID) {
	if fail(c, h.uc.Delete.Execute(c.Request.Context(), id.String())) {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) SetCollectionProducts(c *gin.Context, id uuid.UUID) {
	var req gencollections.SetProductsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	ids := make([]string, len(req.ProductIds))
	for i, p := range req.ProductIds {
		ids[i] = p.String()
	}
	if fail(c, h.uc.SetProducts.Execute(c.Request.Context(), id.String(), ids)) {
		return
	}
	h.GetCollection(c, id)
}
