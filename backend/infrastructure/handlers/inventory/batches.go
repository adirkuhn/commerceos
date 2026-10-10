package inventory

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	geninventory "github.com/adirkuhn/commerceos/api/inventory"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ── Batch route handlers ─────────────────────────────────────────────────────

func (h *Handler) ListBatches(c *gin.Context, skuId geninventory.SkuId) {
	batches, err := h.listBatches.Execute(c.Request.Context(), skuId)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]geninventory.Batch, len(batches))
	for i, b := range batches {
		resp[i] = toGenBatch(b)
	}
	c.JSON(http.StatusOK, geninventory.BatchListResponse{Data: &resp})
}

func (h *Handler) CreateBatch(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.CreateBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	var receivedAt time.Time
	if req.ReceivedAt != nil {
		receivedAt = *req.ReceivedAt
	}
	batch, err := h.createBatch.Execute(c.Request.Context(), appinventory.CreateBatchInput{
		SKUID: skuId, WarehouseID: req.WarehouseId, LotNumber: strPtrOr(req.LotNumber),
		ExpiryDate: genDateToTimePtr(req.ExpiryDate), ReceivedAt: receivedAt,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenBatch(*batch)
	c.JSON(http.StatusCreated, geninventory.BatchResponse{Data: &resp})
}

func (h *Handler) UpdateBatch(c *gin.Context, id geninventory.Id) {
	var req geninventory.UpdateBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	batch, err := h.updateBatch.Execute(c.Request.Context(), appinventory.UpdateBatchInput{
		ID: id, WarehouseID: req.WarehouseId, LotNumber: strPtrOr(req.LotNumber),
		ExpiryDate: genDateToTimePtr(req.ExpiryDate),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenBatch(*batch)
	c.JSON(http.StatusOK, geninventory.BatchResponse{Data: &resp})
}

func (h *Handler) DeleteBatch(c *gin.Context, id geninventory.Id) {
	err := h.deleteBatch.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, geninventory.EmptyResponse{})
}

// ── Stock-level / movement route handlers ────────────────────────────────────

func (h *Handler) GetSKUStockLevel(c *gin.Context, skuId geninventory.SkuId) {
	level, err := h.getSKUStockLevel.Execute(c.Request.Context(), skuId)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := geninventory.StockLevel{SkuId: level.SKUID, Total: level.Total, Syncable: level.Syncable}
	c.JSON(http.StatusOK, geninventory.StockLevelResponse{Data: &resp})
}

func (h *Handler) ListBatchMovements(c *gin.Context, id geninventory.Id) {
	movements, err := h.listBatchMovements.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]geninventory.StockMovement, len(movements))
	for i, m := range movements {
		resp[i] = toGenStockMovement(m)
	}
	c.JSON(http.StatusOK, geninventory.StockMovementListResponse{Data: &resp})
}
