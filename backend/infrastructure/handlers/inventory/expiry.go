package inventory

import (
	"github.com/adirkuhn/commerceos/infrastructure/httperrs"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	geninventory "github.com/adirkuhn/commerceos/api/inventory"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) ListExpiringBatches(c *gin.Context, params geninventory.ListExpiringBatchesParams) {
	withinDays := 30
	if params.WithinDays != nil {
		withinDays = *params.WithinDays
	}
	batches, err := h.listExpiringBatches.Execute(c.Request.Context(), withinDays)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]geninventory.ExpiringBatch, len(batches))
	for i, b := range batches {
		resp[i] = toGenExpiringBatch(b)
	}
	c.JSON(http.StatusOK, geninventory.ExpiringBatchListResponse{Data: &resp})
}

func (h *Handler) ListPendingExpiredBatches(c *gin.Context) {
	batches, err := h.listPendingExpiredBatches.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]geninventory.ExpiredBatchReview, len(batches))
	for i, b := range batches {
		resp[i] = toGenExpiredBatchReview(b)
	}
	c.JSON(http.StatusOK, geninventory.ExpiredBatchReviewListResponse{Data: &resp})
}

func (h *Handler) WriteOffExpiredBatch(c *gin.Context, id geninventory.Id) {
	performedBy := ""
	if claims := middleware.ClaimsFromGin(c); claims != nil {
		performedBy = claims.UserID
	}
	err := h.writeOffExpiredBatch.Execute(c.Request.Context(), id, performedBy)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, geninventory.EmptyResponse{})
}

func (h *Handler) ExtendBatchExpiry(c *gin.Context, id geninventory.Id) {
	var req geninventory.ExtendBatchExpiryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	err := h.extendBatchExpiry.Execute(c.Request.Context(), id, req.NewExpiryDate.Time)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, geninventory.EmptyResponse{})
}

func (h *Handler) DismissExpiredBatch(c *gin.Context, id geninventory.Id) {
	err := h.dismissExpiredBatch.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, geninventory.EmptyResponse{})
}

func (h *Handler) DiscardBatch(c *gin.Context, skuId geninventory.SkuId) {
	warehouseID := c.PostForm("warehouse_id")
	quantity, err := strconv.ParseInt(c.PostForm("quantity"), 10, 64)
	if warehouseID == "" || err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", httperrs.ErrRequiredFieldsMissing)
		return
	}
	var batchID *string
	if v := c.PostForm("batch_id"); v != "" {
		batchID = &v
	}

	discardedBy := ""
	if claims := middleware.ClaimsFromGin(c); claims != nil {
		discardedBy = claims.UserID
	}
	in := appinventory.DiscardBatchInput{
		SKUID: skuId, WarehouseID: warehouseID, BatchID: batchID, Quantity: quantity,
		Reason: c.PostForm("reason"), DiscardedBy: discardedBy,
	}
	if fileHeader, ferr := c.FormFile("photo"); ferr == nil {
		f, ferr := fileHeader.Open()
		if ferr != nil {
			middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", ferr.Error())
			return
		}
		defer f.Close()
		in.PhotoFilename = fileHeader.Filename
		in.PhotoContentType = fileHeader.Header.Get("Content-Type")
		in.PhotoData = f
	}

	record, err := h.discardBatch.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenDiscardRecord(*record)
	c.JSON(http.StatusCreated, geninventory.DiscardRecordResponse{Data: &resp})
}

func (h *Handler) ListDiscards(c *gin.Context, params geninventory.ListDiscardsParams) {
	in := appinventory.ListDiscardsInput{WarehouseID: params.WarehouseId, SKUID: params.SkuId, From: params.From, To: params.To}
	if params.Type != nil {
		t := invDomain.MovementType(*params.Type)
		in.Type = &t
	}
	if params.Limit != nil {
		in.Limit = *params.Limit
	}
	if params.Offset != nil {
		in.Offset = *params.Offset
	}
	result, err := h.listDiscards.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]geninventory.DiscardRecordReport, len(result.Discards))
	for i, d := range result.Discards {
		resp[i] = toGenDiscardRecordReport(d)
	}
	total := int(result.Total)
	c.JSON(http.StatusOK, geninventory.DiscardRecordListResponse{Data: &resp, Total: &total})
}
