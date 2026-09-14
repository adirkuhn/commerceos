// Package weightportions wires the admin-only (JWT) Bulk Weight Portions
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

type bulkAddWeightPortionsUC interface {
	Execute(ctx context.Context, in apppms.BulkAddWeightPortionsInput) (*apppms.BulkAddWeightPortionsResult, error)
}

type Handler struct {
	bulkAdd bulkAddWeightPortionsUC
}

func NewHandler(bulkAdd bulkAddWeightPortionsUC) *Handler {
	return &Handler{bulkAdd: bulkAdd}
}

var _ genweightportions.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB, outboxRepo outbox.Repository) {
	skuRepo := repositories.NewSKURepo(db)
	uc := apppms.NewBulkAddWeightPortionsUseCase(
		db, skuRepo, repositories.NewBatchRepo(db),
		apppms.NewCreateSKUUseCase(db, repositories.NewVariantRepo(db), skuRepo, repositories.NewSKUPriceHistoryRepo(db), outboxRepo),
		invapp.NewCreateBatchUseCase(db, repositories.NewBatchRepo(db)),
		invapp.NewPortionFromBulkUseCase(db, skuRepo, repositories.NewStockMovementRepo(db), outboxRepo),
	)
	h := NewHandler(uc)
	genweightportions.RegisterHandlers(rg, h)
}

func (h *Handler) BulkAddWeightPortions(c *gin.Context, id string) {
	var req genweightportions.BulkAddWeightPortionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err.Error())
		return
	}
	weights := make([]int64, len(req.WeightsGrams))
	for i, w := range req.WeightsGrams {
		weights[i] = int64(w)
	}
	result, err := h.bulkAdd.Execute(c.Request.Context(), apppms.BulkAddWeightPortionsInput{
		BulkSKUID: id, WarehouseID: req.WarehouseId, BulkBatchID: req.BulkBatchId,
		WeightsGrams: weights, SKUCodePrefix: stringValue(req.SkuCodePrefix), Reason: stringValue(req.Reason),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	created := make([]genweightportions.CreatedPortionedSKU, len(result.Created))
	for i, sku := range result.Created {
		created[i] = toGenCreatedSKU(sku)
	}
	middleware.OK(c, struct {
		Created []genweightportions.CreatedPortionedSKU `json:"created"`
	}{Created: created})
}

func toGenCreatedSKU(sku pmsDomain.SKU) genweightportions.CreatedPortionedSKU {
	var weightGrams int
	if sku.WeightGrams != nil {
		weightGrams = int(*sku.WeightGrams)
	}
	return genweightportions.CreatedPortionedSKU{
		Id: sku.ID, SkuCode: sku.SKUCode, Barcode: sku.Barcode,
		PriceMinorUnits: int(sku.PriceMinorUnits), CurrencyCode: sku.CurrencyCode, WeightGrams: weightGrams,
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
