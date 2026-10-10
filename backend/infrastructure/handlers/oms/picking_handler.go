package oms

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) StartOrResumePicking(c *gin.Context, id genoms.Id) {
	session, err := h.startOrResumePicking.Execute(c.Request.Context(), id, lockActorFrom(middleware.ClaimsFromGin(c)))
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPickingSession(*session)
	c.JSON(http.StatusOK, genoms.PickingSessionResponse{Data: &resp})
}

func (h *Handler) TouchPickingSession(c *gin.Context, id genoms.Id) {
	claims := middleware.ClaimsFromGin(c)
	userID := ""
	if claims != nil {
		userID = claims.UserID
	}
	err := h.touchPickingSession.Execute(c.Request.Context(), id, userID)
	if err := mapError(c, err); err != nil {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ResetPickingSession(c *gin.Context, id genoms.Id) {
	session, err := h.resetPickingSession.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPickingSession(*session)
	c.JSON(http.StatusOK, genoms.PickingSessionResponse{Data: &resp})
}

func (h *Handler) GetPickingSession(c *gin.Context, id genoms.Id) {
	detail, err := h.getPickingSession.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	lineItems := make([]genoms.SalesOrderLineItem, len(detail.LineItems))
	for i, li := range detail.LineItems {
		lineItems[i] = toGenSalesOrderLineItem(li)
	}
	scans := make([]genoms.PickScan, len(detail.Scans))
	for i, s := range detail.Scans {
		scans[i] = toGenPickScan(s)
	}
	resp := genoms.PickingSessionDetail{Session: toGenPickingSession(detail.Session), LineItems: lineItems, Scans: scans}
	c.JSON(http.StatusOK, genoms.PickingSessionDetailResponse{Data: &resp})
}

func (h *Handler) ScanPickItem(c *gin.Context, id genoms.Id) {
	var req genoms.ScanPickItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	claims := middleware.ClaimsFromGin(c)
	scannedBy := ""
	if claims != nil {
		scannedBy = claims.UserID
	}
	actionType := omsDomain.PickScanActionPick
	if req.ActionType != nil {
		actionType = omsDomain.PickScanActionType(*req.ActionType)
	}
	result, err := h.scanPickItem.Execute(c.Request.Context(), appoms.ScanPickItemInput{
		SessionID: id, Code: strOrEmpty(req.Code), SKUID: strOrEmpty(req.SkuId),
		ActionType: actionType, Quantity: int64OrZero(req.Quantity),
		Reason: strOrEmpty(req.Reason), ScannedBy: scannedBy,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := genoms.ScanPickItemResult{Scan: toGenPickScan(result.Scan)}
	c.JSON(http.StatusCreated, genoms.ScanPickItemResultResponse{Data: &resp})
}

func (h *Handler) SubstituteLineItem(c *gin.Context, id genoms.Id) {
	var req genoms.SubstituteLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	claims := middleware.ClaimsFromGin(c)
	scannedBy := ""
	if claims != nil {
		scannedBy = claims.UserID
	}
	lines := make([]appoms.SubstituteLineInput, len(req.Lines))
	for i, l := range req.Lines {
		returnToStock := false
		if l.ReturnToStock != nil {
			returnToStock = *l.ReturnToStock
		}
		lines[i] = appoms.SubstituteLineInput{
			SKUID: l.SkuId, SubstituteQuantity: l.SubstituteQuantity, OriginalQtyCovered: l.OriginalQuantityCovered,
			ReturnToStock: returnToStock,
		}
	}
	result, err := h.substituteLineItem.Execute(c.Request.Context(), appoms.SubstituteLineItemInput{
		SessionID: id, LineItemID: req.LineItemId, Lines: lines, Reason: strOrEmpty(req.Reason), ScannedBy: scannedBy,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	scans := make([]genoms.PackScan, len(result.Scans))
	for i, s := range result.Scans {
		scans[i] = toGenPackScan(s)
	}
	resp := genoms.SubstituteLineItemResult{Scans: scans}
	c.JSON(http.StatusCreated, genoms.SubstituteLineItemResultResponse{Data: &resp})
}

func (h *Handler) RefundLineItem(c *gin.Context, id genoms.Id) {
	var req genoms.RefundLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	claims := middleware.ClaimsFromGin(c)
	userID, role := "", ""
	if claims != nil {
		userID, role = claims.UserID, string(claims.Role)
	}
	result, err := h.refundLineItem.Execute(c.Request.Context(), appoms.RefundLineItemInput{
		SessionID: id, LineItemID: req.LineItemId, Quantity: req.Quantity, Reason: strOrEmpty(req.Reason),
		ScannedBy: userID, RequesterUserID: userID, RequesterRole: role,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := genoms.RefundLineItemResult{Scan: toGenPackScan(result.Scan)}
	c.JSON(http.StatusCreated, genoms.RefundLineItemResultResponse{Data: &resp})
}

func (h *Handler) ResolvePickScanCode(c *gin.Context, id genoms.Id) {
	var req genoms.ResolvePickScanCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	result, err := h.resolvePickScanCode.Execute(c.Request.Context(), id, req.Code)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := genoms.ResolvePickScanCodeResult{SkuId: result.SKUID, ProductName: result.ProductName, WeightGrams: result.WeightGrams}
	c.JSON(http.StatusOK, genoms.ResolvePickScanCodeResultResponse{Data: &resp})
}

func (h *Handler) DeletePickScan(c *gin.Context, id genoms.Id, scanId string) {
	err := h.deletePickScan.Execute(c.Request.Context(), id, scanId)
	if err := mapError(c, err); err != nil {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) PausePicking(c *gin.Context, id genoms.Id) {
	session, err := h.pausePicking.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPickingSession(*session)
	c.JSON(http.StatusOK, genoms.PickingSessionResponse{Data: &resp})
}

func (h *Handler) CompletePicking(c *gin.Context, id genoms.Id) {
	claims := middleware.ClaimsFromGin(c)
	completedBy := ""
	if claims != nil {
		completedBy = claims.UserID
	}
	var req genoms.CompletePickingJSONRequestBody
	// The request body is optional (no shortfalls to resolve yet) --
	// ShouldBindJSON on an empty body would error, so only attempt it when
	// one was actually sent.
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
			return
		}
	}
	resolutions := make(map[string]appoms.StockShortfallResolution)
	if req.Resolutions != nil {
		for scanID, r := range *req.Resolutions {
			resolutions[scanID] = appoms.StockShortfallResolution(r)
		}
	}

	result, err := h.completePicking.Execute(c.Request.Context(), appoms.CompletePickingInput{
		SessionID: id, CompletedBy: completedBy, Resolutions: resolutions,
	})
	if err := mapError(c, err); err != nil {
		return
	}

	resp := genoms.CompletePickingResult{Completed: result.Completed}
	if result.Completed {
		session := toGenPickingSession(result.Session)
		resp.Session = &session
		missing := make([]genoms.MissingPickItem, len(result.MissingItems))
		for i, m := range result.MissingItems {
			missing[i] = genoms.MissingPickItem{LineItemId: m.LineItemID, Name: m.Name, ExpectedQty: m.ExpectedQty, PickedQty: m.PickedQty}
		}
		resp.MissingItems = &missing
	} else {
		shortfalls := make([]genoms.StockShortfall, len(result.StockShortfalls))
		for i, s := range result.StockShortfalls {
			shortfalls[i] = genoms.StockShortfall{ScanId: s.ScanID, SkuId: s.SKUID, Name: s.Name, Needed: s.Needed, Short: s.Short}
		}
		resp.StockShortfalls = &shortfalls
	}
	c.JSON(http.StatusOK, genoms.CompletePickingResultResponse{Data: &resp})
}

func toGenPickingSession(s omsDomain.PickingSession) genoms.PickingSession {
	return genoms.PickingSession{
		Id: s.ID, SalesOrderId: s.SalesOrderID, Status: genoms.PickingSessionStatus(s.Status),
		StartedAt: s.StartedAt, PausedAt: s.PausedAt, CompletedAt: s.CompletedAt, CompletedBy: s.CompletedBy,
		LockedBy: s.LockedBy, LockedAt: s.LockedAt, LockedSince: s.LockedSince,
	}
}

func toGenPickScan(s omsDomain.PickScan) genoms.PickScan {
	returnToStock := s.ReturnToStock
	return genoms.PickScan{
		Id: s.ID, SessionId: s.SessionID, LineItemId: s.LineItemID, SkuId: s.SKUID, ActionType: genoms.PickScanActionType(s.ActionType),
		Quantity: s.Quantity, OriginalQuantityCovered: s.OriginalQtyCovered, ReturnToStock: &returnToStock,
		PriceMinorUnits: s.PriceMinorUnits, Reason: &s.Reason,
		ScannedAt: s.ScannedAt, ScannedBy: s.ScannedBy,
	}
}
