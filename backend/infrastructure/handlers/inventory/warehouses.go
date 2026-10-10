package inventory

import (
	"github.com/adirkuhn/commerceos/infrastructure/httperrs"
	"net/http"

	"github.com/gin-gonic/gin"

	geninventory "github.com/adirkuhn/commerceos/api/inventory"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ── Warehouse route handlers ─────────────────────────────────────────────────

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
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	if req.Name == "" {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", httperrs.ErrRequiredFieldsMissing)
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
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	if req.Name == "" {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", httperrs.ErrRequiredFieldsMissing)
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
