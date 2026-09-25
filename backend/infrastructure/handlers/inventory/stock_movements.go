package inventory

import (
	"net/http"

	"github.com/gin-gonic/gin"

	geninventory "github.com/adirkuhn/commerceos/api/inventory"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) ListStockMovements(c *gin.Context, skuId geninventory.SkuId) {
	movements, err := h.listStockMovements.Execute(c.Request.Context(), skuId)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]geninventory.StockMovement, len(movements))
	for i, m := range movements {
		resp[i] = toGenStockMovement(m)
	}
	c.JSON(http.StatusOK, geninventory.StockMovementListResponse{Data: &resp})
}

func (h *Handler) RecordStockMovement(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.RecordStockMovementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	movement, err := h.recordStockMovement.Execute(c.Request.Context(), appinventory.RecordStockMovementInput{
		SKUID: skuId, WarehouseID: req.WarehouseId, BatchID: req.BatchId,
		Type: invDomain.MovementType(req.Type), QuantityDelta: req.QuantityDelta,
		Reason: strPtrOr(req.Reason), OccurredAt: timeOrNow(req.OccurredAt),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenStockMovement(*movement)
	c.JSON(http.StatusCreated, geninventory.StockMovementResponse{Data: &resp})
}

func (h *Handler) RecordRecount(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.RecordRecountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	movement, err := h.recordRecount.Execute(c.Request.Context(), appinventory.RecordRecountInput{
		SKUID: skuId, WarehouseID: req.WarehouseId, BatchID: req.BatchId,
		CountedQuantity: req.CountedQuantity, Reason: strPtrOr(req.Reason), OccurredAt: timeOrNow(req.OccurredAt),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	var resp *geninventory.StockMovement
	if movement != nil {
		g := toGenStockMovement(*movement)
		resp = &g
	}
	c.JSON(http.StatusOK, geninventory.StockMovementResponse{Data: resp})
}

func (h *Handler) TransferStock(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.TransferStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.transferStock.Execute(c.Request.Context(), appinventory.TransferStockInput{
		SKUID: skuId, Quantity: req.Quantity,
		FromWarehouseID: req.FromWarehouseId, FromBatchID: req.FromBatchId,
		ToWarehouseID: req.ToWarehouseId, ToBatchID: req.ToBatchId,
		Reason: strPtrOr(req.Reason), OccurredAt: timeOrNow(req.OccurredAt),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := geninventory.TransferStockResult{Out: toGenStockMovement(*result.Out), In: toGenStockMovement(*result.In)}
	c.JSON(http.StatusCreated, geninventory.TransferStockResultResponse{Data: &resp})
}

func (h *Handler) SplitCaseIntoUnits(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.SplitCaseIntoUnitsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.splitCaseIntoUnits.Execute(c.Request.Context(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: skuId, WarehouseID: req.WarehouseId, BatchID: req.BatchId, CasesToSplit: req.CasesToSplit,
		Reason: strPtrOr(req.Reason), OccurredAt: timeOrNow(req.OccurredAt),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := geninventory.SplitCaseIntoUnitsResult{
		CaseOut: toGenStockMovement(result.CaseOutMovement), UnitIn: toGenStockMovement(result.UnitInMovement),
	}
	c.JSON(http.StatusCreated, geninventory.SplitCaseIntoUnitsResultResponse{Data: &resp})
}

func (h *Handler) PortionFromBulk(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.PortionFromBulkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.portionFromBulk.Execute(c.Request.Context(), appinventory.PortionFromBulkInput{
		TargetSKUID: skuId, WarehouseID: req.WarehouseId, BatchID: req.BatchId, WeightGrams: req.WeightGrams,
		Reason: strPtrOr(req.Reason), OccurredAt: timeOrNow(req.OccurredAt),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := geninventory.PortionFromBulkResult{
		BulkOut: toGenStockMovement(result.BulkOutMovement), UnitIn: toGenStockMovement(result.UnitInMovement),
	}
	c.JSON(http.StatusCreated, geninventory.PortionFromBulkResultResponse{Data: &resp})
}

func (h *Handler) MarkDamagedOrLost(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.MarkDamagedOrLostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	discardedBy := ""
	if claims := middleware.ClaimsFromGin(c); claims != nil {
		discardedBy = claims.UserID
	}
	movement, _, err := h.markDamagedOrLost.Execute(c.Request.Context(), appinventory.MarkDamagedOrLostInput{
		SKUID: skuId, WarehouseID: req.WarehouseId, BatchID: req.BatchId,
		Quantity: req.Quantity, Type: invDomain.MovementType(req.Type),
		Reason: strPtrOr(req.Reason), OccurredAt: timeOrNow(req.OccurredAt), DiscardedBy: discardedBy,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenStockMovement(*movement)
	c.JSON(http.StatusCreated, geninventory.StockMovementResponse{Data: &resp})
}

func (h *Handler) ReceiveStock(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.ReceiveStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.receiveStock.Execute(c.Request.Context(), appinventory.ReceiveStockInput{
		SKUID: skuId, WarehouseID: req.WarehouseId, Quantity: req.Quantity, BatchID: req.BatchId,
		LotNumber: strPtrOr(req.LotNumber), ExpiryDate: genDateToTimePtr(req.ExpiryDate),
		Reason: strPtrOr(req.Reason), OccurredAt: timeOrNow(req.OccurredAt),
		PurchaseOrderLineItemID: req.PurchaseOrderLineItemId, UnitCostMinorUnits: req.UnitCostMinorUnits,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	var batch *geninventory.Batch
	if result.Batch != nil {
		b := toGenBatch(*result.Batch)
		batch = &b
	}
	resp := geninventory.ReceiveStockResult{Batch: batch, Movement: toGenStockMovement(*result.Movement)}
	c.JSON(http.StatusCreated, geninventory.ReceiveStockResultResponse{Data: &resp})
}
