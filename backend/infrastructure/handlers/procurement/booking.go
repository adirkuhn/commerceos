package procurement

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genprocurement "github.com/adirkuhn/commerceos/api/procurement"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ── Booking (scanner app) route handlers ─────────────────────────────────────

func (h *Handler) StartOrResumeBooking(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.StartOrResumeBookingRequest
	// Body is optional (a resume never needs warehouse_id) -- a missing/
	// empty body is not a binding error, only malformed JSON is.
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
			return
		}
	}
	booking, err := h.startOrResumeBooking.Execute(c.Request.Context(), appprocurement.StartOrResumeBookingInput{
		PurchaseOrderID: id, WarehouseID: req.WarehouseId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenBooking(*booking)
	c.JSON(http.StatusOK, genprocurement.BookingResponse{Data: &resp})
}

func (h *Handler) GetBooking(c *gin.Context, id genprocurement.Id) {
	detail, err := h.getBooking.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	lineItems := make([]genprocurement.PurchaseOrderLineItem, len(detail.LineItems))
	for i, li := range detail.LineItems {
		lineItems[i] = toGenLineItem(li)
	}
	scans := make([]genprocurement.BookingScan, len(detail.Scans))
	for i, s := range detail.Scans {
		scans[i] = toGenBookingScan(s)
	}
	resp := genprocurement.BookingDetail{Booking: toGenBooking(detail.Booking), LineItems: lineItems, Scans: scans}
	c.JSON(http.StatusOK, genprocurement.BookingDetailResponse{Data: &resp})
}

func (h *Handler) ScanBookingItem(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.ScanBookingItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	claims := middleware.ClaimsFromGin(c)
	scannedBy := ""
	if claims != nil {
		scannedBy = claims.UserID
	}
	actionType := procurementDomain.BookingScanActionReceive
	if req.ActionType != nil {
		actionType = procurementDomain.BookingScanActionType(*req.ActionType)
	}
	result, err := h.scanBookingItem.Execute(c.Request.Context(), appprocurement.ScanBookingItemInput{
		BookingID: id, Code: strPtrOr(req.Code), SKUID: strPtrOr(req.SkuId), ActionType: actionType,
		Quantity: req.Quantity, BatchID: req.BatchId,
		LotNumber: strPtrOr(req.LotNumber), ExpiryDate: genDateToTimePtr(req.ExpiryDate), Reason: strPtrOr(req.Reason), ScannedBy: scannedBy,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := genprocurement.ScanBookingItemResult{Scan: toGenBookingScan(result.Scan)}
	c.JSON(http.StatusCreated, genprocurement.ScanBookingItemResultResponse{Data: &resp})
}

func (h *Handler) ResolveBookingScanCode(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.ResolveBookingScanCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	result, err := h.resolveBookingScanCode.Execute(c.Request.Context(), id, req.Code)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := genprocurement.ResolveBookingScanCodeResult{SkuId: result.SKUID, ProductName: result.ProductName, UnitsPerScan: result.UnitsPerScan, WeightGrams: result.WeightGrams}
	c.JSON(http.StatusOK, genprocurement.ResolveBookingScanCodeResultResponse{Data: &resp})
}

func (h *Handler) DeleteBookingScan(c *gin.Context, id genprocurement.Id, scanId genprocurement.ScanId) {
	err := h.deleteBookingScan.Execute(c.Request.Context(), id, scanId)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genprocurement.EmptyResponse{})
}

func (h *Handler) PauseBooking(c *gin.Context, id genprocurement.Id) {
	booking, err := h.pauseBooking.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenBooking(*booking)
	c.JSON(http.StatusOK, genprocurement.BookingResponse{Data: &resp})
}

func (h *Handler) CompleteBooking(c *gin.Context, id genprocurement.Id) {
	claims := middleware.ClaimsFromGin(c)
	completedBy := ""
	if claims != nil {
		completedBy = claims.UserID
	}
	result, err := h.completeBooking.Execute(c.Request.Context(), id, completedBy)
	if err := mapError(c, err); err != nil {
		return
	}
	missing := make([]genprocurement.MissingBookingItem, len(result.MissingItems))
	for i, m := range result.MissingItems {
		missing[i] = genprocurement.MissingBookingItem{
			LineItemId: m.LineItemID, RawDescription: m.RawDescription, ExpectedQty: m.ExpectedQty, ScannedQty: m.ScannedQty,
		}
	}
	resp := genprocurement.CompleteBookingResult{Booking: toGenBooking(result.Booking), MissingItems: missing}
	c.JSON(http.StatusOK, genprocurement.CompleteBookingResultResponse{Data: &resp})
}

func (h *Handler) ListBookingDiscrepancies(c *gin.Context) {
	report, err := h.listBookingDiscrepancies.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	missing := make([]genprocurement.MissingBookingItemReport, len(report.MissingItems))
	for i, m := range report.MissingItems {
		missing[i] = genprocurement.MissingBookingItemReport{
			BookingId: m.BookingID, PurchaseOrderId: m.PurchaseOrderID, CompletedAt: m.CompletedAt,
			LineItemId: m.LineItemID, SkuCode: m.SKUCode, ProductName: m.ProductName,
			RawDescription: m.RawDescription, ExpectedQty: m.ExpectedQty, ScannedQty: m.ScannedQty, MissingQty: m.MissingQty,
		}
	}
	damagedLost := make([]genprocurement.DamagedLostBookingItemReport, len(report.DamagedLost))
	for i, d := range report.DamagedLost {
		damagedLost[i] = genprocurement.DamagedLostBookingItemReport{
			BookingId: d.BookingID, PurchaseOrderId: d.PurchaseOrderID, CompletedAt: d.CompletedAt,
			ScanId: d.ScanID, SkuId: d.SKUID, SkuCode: d.SKUCode, ProductName: d.ProductName,
			ActionType: genprocurement.DamagedLostBookingItemReportActionType(d.ActionType),
			Quantity:   d.Quantity, Reason: d.Reason, ScannedAt: d.ScannedAt, ScannedBy: d.ScannedBy,
		}
	}
	resp := genprocurement.BookingDiscrepancyReport{MissingItems: missing, DamagedLost: damagedLost}
	c.JSON(http.StatusOK, genprocurement.BookingDiscrepancyReportResponse{Data: &resp})
}
