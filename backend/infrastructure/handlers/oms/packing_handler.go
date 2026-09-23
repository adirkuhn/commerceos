// Packing handler methods live in their own file, separate from
// handler.go's picking/sales-order methods, purely to keep that already-
// large file from growing further (see its own size vs. CLAUDE.md's
// soft-400/hard-800-line guidance) -- Handler is still one type, these are
// just more of its methods.
package oms

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

type startOrResumePackingUC interface {
	Execute(ctx context.Context, salesOrderID, userID string) (*omsDomain.PackingSession, error)
}
type getPackingSessionUC interface {
	Execute(ctx context.Context, id string) (*appoms.PackingSessionDetail, error)
}
type touchPackingSessionUC interface {
	Execute(ctx context.Context, sessionID, userID string) error
}
type resetPackingSessionUC interface {
	Execute(ctx context.Context, sessionID string) (*omsDomain.PackingSession, error)
}
type scanPackItemUC interface {
	Execute(ctx context.Context, in appoms.ScanPackItemInput) (*appoms.ScanPackItemResult, error)
}
type deletePackScanUC interface {
	Execute(ctx context.Context, sessionID, scanID string) error
}
type completePackingUC interface {
	Execute(ctx context.Context, in appoms.CompletePackingInput) (*appoms.CompletePackingResult, error)
}

func (h *Handler) StartOrResumePacking(c *gin.Context, id genoms.Id) {
	claims := middleware.ClaimsFromGin(c)
	userID := ""
	if claims != nil {
		userID = claims.UserID
	}
	session, err := h.startOrResumePacking.Execute(c.Request.Context(), id, userID)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPackingSession(*session)
	c.JSON(http.StatusOK, genoms.PackingSessionResponse{Data: &resp})
}

func (h *Handler) TouchPackingSession(c *gin.Context, id genoms.Id) {
	claims := middleware.ClaimsFromGin(c)
	userID := ""
	if claims != nil {
		userID = claims.UserID
	}
	err := h.touchPackingSession.Execute(c.Request.Context(), id, userID)
	if err := mapError(c, err); err != nil {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ResetPackingSession(c *gin.Context, id genoms.Id) {
	session, err := h.resetPackingSession.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPackingSession(*session)
	c.JSON(http.StatusOK, genoms.PackingSessionResponse{Data: &resp})
}

func (h *Handler) GetPackingSession(c *gin.Context, id genoms.Id) {
	detail, err := h.getPackingSession.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	lineItems := make([]genoms.SalesOrderLineItem, len(detail.LineItems))
	for i, li := range detail.LineItems {
		lineItems[i] = toGenSalesOrderLineItem(li)
	}
	scans := make([]genoms.PackScan, len(detail.Scans))
	for i, s := range detail.Scans {
		scans[i] = toGenPackScan(s)
	}
	resp := genoms.PackingSessionDetail{Session: toGenPackingSession(detail.Session), LineItems: lineItems, Scans: scans}
	c.JSON(http.StatusOK, genoms.PackingSessionDetailResponse{Data: &resp})
}

func (h *Handler) ScanPackItem(c *gin.Context, id genoms.Id) {
	var req genoms.ScanPackItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	claims := middleware.ClaimsFromGin(c)
	scannedBy := ""
	if claims != nil {
		scannedBy = claims.UserID
	}
	result, err := h.scanPackItem.Execute(c.Request.Context(), appoms.ScanPackItemInput{
		SessionID: id, Code: strOrEmpty(req.Code), SKUID: strOrEmpty(req.SkuId),
		Quantity: int64OrZero(req.Quantity), ScannedBy: scannedBy,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := genoms.ScanPackItemResult{Scan: toGenPackScan(result.Scan)}
	c.JSON(http.StatusCreated, genoms.ScanPackItemResultResponse{Data: &resp})
}

func (h *Handler) DeletePackScan(c *gin.Context, id genoms.Id, scanId string) {
	err := h.deletePackScan.Execute(c.Request.Context(), id, scanId)
	if err := mapError(c, err); err != nil {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) CompletePacking(c *gin.Context, id genoms.Id) {
	claims := middleware.ClaimsFromGin(c)
	completedBy := ""
	if claims != nil {
		completedBy = claims.UserID
	}
	result, err := h.completePacking.Execute(c.Request.Context(), appoms.CompletePackingInput{
		SessionID: id, CompletedBy: completedBy,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	missing := make([]genoms.MissingPackItem, len(result.MissingItems))
	for i, m := range result.MissingItems {
		missing[i] = genoms.MissingPackItem{LineItemId: m.LineItemID, Name: m.Name, ExpectedQty: m.ExpectedQty, PackedQty: m.PackedQty}
	}
	resp := genoms.CompletePackingResult{Session: toGenPackingSession(result.Session), MissingItems: &missing}
	c.JSON(http.StatusOK, genoms.CompletePackingResultResponse{Data: &resp})
}

func toGenPackingSession(s omsDomain.PackingSession) genoms.PackingSession {
	return genoms.PackingSession{
		Id: s.ID, SalesOrderId: s.SalesOrderID, Status: genoms.PackingSessionStatus(s.Status),
		StartedAt: s.StartedAt, CompletedAt: s.CompletedAt, CompletedBy: s.CompletedBy,
		LockedBy: s.LockedBy, LockedAt: s.LockedAt,
	}
}

func toGenPackScan(s omsDomain.PackScan) genoms.PackScan {
	return genoms.PackScan{
		Id: s.ID, SessionId: s.SessionID, LineItemId: s.LineItemID, SkuId: s.SKUID,
		Quantity: s.Quantity, ScannedAt: s.ScannedAt, ScannedBy: s.ScannedBy,
	}
}
