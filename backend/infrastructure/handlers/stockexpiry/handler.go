// Package stockexpiry wires the close-to-expiry API (see api/stockexpiry's
// doc comment) to its use cases.
package stockexpiry

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/gorm"

	genstockexpiry "github.com/adirkuhn/commerceos/api/stockexpiry"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type listUC interface {
	Execute(ctx context.Context, in appinventory.ListCloseToExpiryInput) (*appinventory.ListCloseToExpiryResult, error)
}

type acknowledgementUC interface {
	Execute(ctx context.Context, in appinventory.BatchExpiryAcknowledgementInput) error
}

type Handler struct {
	list        listUC
	acknowledge acknowledgementUC
	undo        acknowledgementUC
}

func NewHandler(list listUC, acknowledge, undo acknowledgementUC) *Handler {
	return &Handler{list: list, acknowledge: acknowledge, undo: undo}
}

var _ genstockexpiry.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB, outboxRepo outbox.Repository) {
	settingsRepo := repositories.NewTenantSettingsRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	acks := repositories.NewExpiryAcknowledgementRepo(db)
	genstockexpiry.RegisterHandlers(rg, NewHandler(
		appinventory.NewListCloseToExpiryBatchesUseCase(db, settingsRepo, repositories.NewCloseToExpiryRepo(db), time.Now),
		appinventory.NewAcknowledgeBatchExpiryUseCase(db, settingsRepo, batchRepo, acks, outboxRepo, time.Now),
		appinventory.NewUnacknowledgeBatchExpiryUseCase(db, batchRepo, acks, outboxRepo),
	))
}

func (h *Handler) ListCloseToExpiryBatches(c *gin.Context, params genstockexpiry.ListCloseToExpiryBatchesParams) {
	in := appinventory.ListCloseToExpiryInput{Page: intValue(params.Page), PerPage: intValue(params.PerPage)}
	if params.Window != nil {
		in.WindowDays = int(*params.Window)
	}
	if params.Acknowledged != nil {
		in.Acknowledged = *params.Acknowledged
	}
	res, err := h.list.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genstockexpiry.CloseToExpiryListResponse{Data: toGenList(res)})
}

func (h *Handler) AcknowledgeBatchExpiry(c *gin.Context, id openapi_types.UUID) {
	var req genstockexpiry.AcknowledgeBatchExpiryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	err := h.acknowledge.Execute(c.Request.Context(), appinventory.BatchExpiryAcknowledgementInput{
		BatchID: id.String(), WindowDays: int(req.WindowDays), By: callerID(c),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) UnacknowledgeBatchExpiry(c *gin.Context, id openapi_types.UUID, windowDays genstockexpiry.UnacknowledgeBatchExpiryParamsWindowDays) {
	err := h.undo.Execute(c.Request.Context(), appinventory.BatchExpiryAcknowledgementInput{
		BatchID: id.String(), WindowDays: int(windowDays), By: callerID(c),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	c.Status(http.StatusNoContent)
}

func toGenList(res *appinventory.ListCloseToExpiryResult) genstockexpiry.CloseToExpiryList {
	out := genstockexpiry.CloseToExpiryList{
		Today: openapi_types.Date{Time: res.Today}, Window: toGenWindow(res.Window),
		Windows: make([]genstockexpiry.CloseToExpiryWindow, len(res.Windows)),
		Items:   make([]genstockexpiry.CloseToExpiryBatch, len(res.Items)),
		Page:    res.Page, PerPage: res.PerPage, Total: res.Total,
	}
	for i, w := range res.Windows {
		out.Windows[i] = toGenWindow(w)
	}
	for i, b := range res.Items {
		out.Items[i] = toGenBatch(b, res.Today)
	}
	return out
}

func toGenWindow(w appinventory.CloseToExpiryWindow) genstockexpiry.CloseToExpiryWindow {
	return genstockexpiry.CloseToExpiryWindow{
		WindowDays: w.Band.WindowDays, MinDaysLeft: w.Band.MinDaysLeft,
		From: openapi_types.Date{Time: w.From}, To: openapi_types.Date{Time: w.To},
		OpenCount: w.Counts.Open, AcknowledgedCount: w.Counts.Acknowledged,
	}
}

func toGenBatch(b invDomain.CloseToExpiryBatch, today time.Time) genstockexpiry.CloseToExpiryBatch {
	out := genstockexpiry.CloseToExpiryBatch{
		BatchId: b.BatchID, LotNumber: b.LotNumber, ExpiryDate: openapi_types.Date{Time: b.ExpiryDate},
		DaysLeft: invDomain.DaysUntil(b.ExpiryDate, today), Quantity: int(b.Quantity),
		WarehouseId: b.WarehouseID, WarehouseName: b.WarehouseName,
		SkuId: b.SKUID, SkuCode: b.SKUCode, ProductId: b.ProductID, ProductName: b.ProductName, ImageUrl: b.ImageURL,
		PriceMinorUnits: int(b.PriceMinorUnits), CurrencyCode: b.CurrencyCode, AcknowledgedAt: b.AcknowledgedAt,
	}
	if sale := b.ActiveSalePrice(today); sale != nil {
		v := int(*sale)
		out.SalePriceMinorUnits = &v
	}
	if b.Clearance != nil {
		out.ClearancePromotion = &genstockexpiry.ClearancePromotion{Id: b.Clearance.PromotionID, Name: b.Clearance.PromotionName}
	}
	return out
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, invDomain.ErrBatchNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, invDomain.ErrBatchNotInExpiryWindow), errors.Is(err, invDomain.ErrUnknownExpiryWindow):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	default:
		log.Printf("stockexpiry: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}

func callerID(c *gin.Context) string {
	if claims := middleware.ClaimsFromGin(c); claims != nil {
		return claims.UserID
	}
	return ""
}

func intValue(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
