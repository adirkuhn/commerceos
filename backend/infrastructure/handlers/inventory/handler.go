package inventory

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	geninventory "github.com/adirkuhn/commerceos/api/inventory"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ── Use-case interfaces (defined at point of use — ISP) ──────────────────────

type listWarehousesUC interface {
	Execute(ctx context.Context) ([]invDomain.Warehouse, error)
}
type createWarehouseUC interface {
	Execute(ctx context.Context, in appinventory.CreateWarehouseInput) (*invDomain.Warehouse, error)
}
type updateWarehouseUC interface {
	Execute(ctx context.Context, in appinventory.UpdateWarehouseInput) (*invDomain.Warehouse, error)
}
type deleteWarehouseUC interface {
	Execute(ctx context.Context, id string) error
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listWarehouses  listWarehousesUC
	createWarehouse createWarehouseUC
	updateWarehouse updateWarehouseUC
	deleteWarehouse deleteWarehouseUC
}

// NewHandler constructs a Handler with injected use cases (useful for testing).
func NewHandler(
	listWarehouses listWarehousesUC,
	createWarehouse createWarehouseUC,
	updateWarehouse updateWarehouseUC,
	deleteWarehouse deleteWarehouseUC,
) *Handler {
	return &Handler{
		listWarehouses:  listWarehouses,
		createWarehouse: createWarehouse,
		updateWarehouse: updateWarehouse,
		deleteWarehouse: deleteWarehouse,
	}
}

var _ geninventory.ServerInterface = (*Handler)(nil)

// Register is the production wiring: creates repos + use cases, then
// mounts the generated routes (matching the OpenAPI spec exactly).
func Register(rg gin.IRouter, db *gorm.DB) {
	warehouseRepo := repositories.NewWarehouseRepo(db)

	h := NewHandler(
		appinventory.NewListWarehousesUseCase(db, warehouseRepo),
		appinventory.NewCreateWarehouseUseCase(db, warehouseRepo),
		appinventory.NewUpdateWarehouseUseCase(db, warehouseRepo),
		appinventory.NewDeleteWarehouseUseCase(db, warehouseRepo),
	)
	geninventory.RegisterHandlers(rg, h)
}

// ── Route handlers ────────────────────────────────────────────────────────────

func (h *Handler) ListWarehouses(c *gin.Context) {
	warehouses, err := h.listWarehouses.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]geninventory.Warehouse, len(warehouses))
	for i, w := range warehouses {
		resp[i] = toGenWarehouse(w)
	}
	c.JSON(http.StatusOK, geninventory.WarehouseListResponse{Data: &resp})
}

func (h *Handler) CreateWarehouse(c *gin.Context) {
	var req geninventory.CreateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.Name == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required")
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	warehouse, err := h.createWarehouse.Execute(c.Request.Context(), appinventory.CreateWarehouseInput{
		Name: req.Name, Address1: strPtrOr(req.Address1), Address2: strPtrOr(req.Address2), City: strPtrOr(req.City),
		Province: strPtrOr(req.Province), Zip: strPtrOr(req.Zip), Country: strPtrOr(req.Country), Phone: strPtrOr(req.Phone),
		Active: active,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenWarehouse(*warehouse)
	c.JSON(http.StatusCreated, geninventory.WarehouseResponse{Data: &resp})
}

func (h *Handler) UpdateWarehouse(c *gin.Context, id geninventory.Id) {
	var req geninventory.UpdateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.Name == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required")
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	warehouse, err := h.updateWarehouse.Execute(c.Request.Context(), appinventory.UpdateWarehouseInput{
		ID: id, Name: req.Name, Address1: strPtrOr(req.Address1), Address2: strPtrOr(req.Address2), City: strPtrOr(req.City),
		Province: strPtrOr(req.Province), Zip: strPtrOr(req.Zip), Country: strPtrOr(req.Country), Phone: strPtrOr(req.Phone),
		Active: active,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenWarehouse(*warehouse)
	c.JSON(http.StatusOK, geninventory.WarehouseResponse{Data: &resp})
}

func (h *Handler) DeleteWarehouse(c *gin.Context, id geninventory.Id) {
	err := h.deleteWarehouse.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, geninventory.EmptyResponse{})
}

// ── Domain → generated-type mapper ───────────────────────────────────────────

func toGenWarehouse(w invDomain.Warehouse) geninventory.Warehouse {
	return geninventory.Warehouse{
		Id:        w.ID,
		TenantId:  w.TenantID,
		Name:      w.Name,
		Address1:  w.Address1,
		Address2:  w.Address2,
		City:      w.City,
		Province:  w.Province,
		Zip:       w.Zip,
		Country:   w.Country,
		Phone:     w.Phone,
		Active:    w.Active,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}

func strPtrOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ── Error mapping ────────────────────────────────────────────────────────────

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, invDomain.ErrWarehouseNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	default:
		log.Printf("inventory: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
