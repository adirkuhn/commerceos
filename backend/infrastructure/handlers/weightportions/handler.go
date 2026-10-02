// Package weightportions wires the admin-only (JWT) Weight Packages
// API (see api/weightportions's doc comment) to its use case.
package weightportions

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genweightportions "github.com/adirkuhn/commerceos/api/weightportions"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type packWeightPortionsUC interface {
	Execute(ctx context.Context, in apppms.PackWeightPortionsInput) (*apppms.PackWeightPortionsResult, error)
}

type Handler struct {
	pack packWeightPortionsUC
}

func NewHandler(pack packWeightPortionsUC) *Handler {
	return &Handler{pack: pack}
}

var _ genweightportions.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB, outboxRepo outbox.Repository) {
	skuRepo := repositories.NewSKURepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	uc := apppms.NewPackWeightPortionsUseCase(
		db, skuRepo, batchRepo,
		apppms.NewCreateSKUUseCase(db, repositories.NewVariantRepo(db), skuRepo, repositories.NewSKUPriceHistoryRepo(db), outboxRepo),
		invapp.NewCreateBatchUseCase(db, batchRepo),
		invapp.NewPortionFromBulkUseCase(db, skuRepo, movementRepo, outboxRepo),
		invapp.NewGetScopedStockUseCase(db, movementRepo),
	)
	genweightportions.RegisterHandlers(rg, NewHandler(uc))
}

func (h *Handler) PackWeightPortions(c *gin.Context, id string) {
	var req genweightportions.PackWeightPortionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err.Error())
		return
	}
	lines := make([]apppms.WeightPackageLine, len(req.Lines))
	for i, l := range req.Lines {
		lines[i] = apppms.WeightPackageLine{WeightGrams: int64(l.WeightGrams), Quantity: int64(l.Quantity)}
	}
	result, err := h.pack.Execute(c.Request.Context(), apppms.PackWeightPortionsInput{
		BulkSKUID: id, WarehouseID: req.WarehouseId, BulkBatchID: req.BulkBatchId,
		Lines: lines, SKUCodePrefix: stringValue(req.SkuCodePrefix), Reason: stringValue(req.Reason),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	packed := make([]genweightportions.PackedWeightLine, len(result.Lines))
	for i, l := range result.Lines {
		packed[i] = toGenPackedLine(l)
	}
	middleware.OK(c, struct {
		Lines []genweightportions.PackedWeightLine `json:"lines"`
	}{Lines: packed})
}

func toGenPackedLine(l apppms.PackedWeightLine) genweightportions.PackedWeightLine {
	return genweightportions.PackedWeightLine{
		SkuId: l.SKU.ID, SkuCode: l.SKU.SKUCode, Barcode: l.SKU.Barcode,
		PriceMinorUnits: int(l.SKU.PriceMinorUnits), CurrencyCode: l.SKU.CurrencyCode,
		WeightGrams: int(l.WeightGrams), Quantity: int(l.Quantity), Created: l.Created,
	}
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, pmsDomain.ErrSKUNotFound), errors.Is(err, invDomain.ErrBatchNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, pmsDomain.ErrNoWeightsProvided), errors.Is(err, pmsDomain.ErrBulkSKUNotWeightPool),
		errors.Is(err, invDomain.ErrInvalidQuantity), errors.Is(err, invDomain.ErrBatchDoesNotBelongToSKU),
		errors.Is(err, invDomain.ErrInsufficientStock), errors.Is(err, pmsDomain.ErrDuplicateSKUCode):
		middleware.Fail(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err.Error())
	default:
		log.Printf("weightportions: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
