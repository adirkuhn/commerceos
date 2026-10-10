package procurement

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genprocurement "github.com/adirkuhn/commerceos/api/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) ListReceivingRecords(c *gin.Context, id genprocurement.Id) {
	records, err := h.listReceivingRecords.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genprocurement.ReceivingRecord, len(records))
	for i, r := range records {
		resp[i] = toGenReceivingRecord(r)
	}
	c.JSON(http.StatusOK, genprocurement.ReceivingRecordListResponse{Data: &resp})
}

func (h *Handler) GetReceivingRecord(c *gin.Context, id genprocurement.Id) {
	detail, err := h.getReceivingRecord.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	discrepancies := make([]genprocurement.ReceivingDiscrepancy, len(detail.Discrepancies))
	for i, d := range detail.Discrepancies {
		discrepancies[i] = toGenReceivingDiscrepancy(d)
	}
	resp := genprocurement.ReceivingRecordDetail{
		ReceivingRecord: toGenReceivingRecord(detail.ReceivingRecord),
		Discrepancies:   discrepancies,
	}
	c.JSON(http.StatusOK, genprocurement.ReceivingRecordDetailResponse{Data: &resp})
}

func (h *Handler) CompleteReceiving(c *gin.Context, id genprocurement.Id) {
	rec, err := h.completeReceiving.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenReceivingRecord(*rec)
	c.JSON(http.StatusOK, genprocurement.ReceivingRecordResponse{Data: &resp})
}

func (h *Handler) AcknowledgeReceivingDiscrepancy(c *gin.Context, id genprocurement.Id) {
	d, err := h.acknowledgeReceivingDiscrepancy.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenReceivingDiscrepancy(*d)
	c.JSON(http.StatusOK, genprocurement.ReceivingDiscrepancyResponse{Data: &resp})
}

func (h *Handler) LinkReceivingDiscrepancies(c *gin.Context) {
	var req genprocurement.LinkReceivingDiscrepanciesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	err := h.linkReceivingDiscrepancies.Execute(c.Request.Context(), req.UnexpectedDiscrepancyId, req.MissingDiscrepancyId)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genprocurement.EmptyResponse{})
}
