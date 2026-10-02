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

type linkExistingWeightPackagesUC interface {
	Execute(ctx context.Context, in apppms.LinkExistingWeightPackagesInput) (*apppms.LinkExistingWeightPackagesResult, error)
}

type Handler struct {
	pack packWeightPortionsUC
	link linkExistingWeightPackagesUC
}

func NewHandler(pack packWeightPortionsUC, link linkExistingWeightPackagesUC) *Handler {
	return &Handler{pack: pack, link: link}
}

var _ genweightportions.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB, outboxRepo outbox.Repository) {
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	uc := apppms.NewPackWeightPortionsUseCase(db, apppms.PackWeightPortionsDeps{
		SKUs: skuRepo, Variants: variantRepo, Products: productRepo, Batches: batchRepo,
		CreateVariant: apppms.NewCreateVariantUseCase(db, productRepo, variantRepo, outboxRepo),
		EnableVariants: apppms.NewUpdateProductUseCase(db, productRepo, variantRepo,
			repositories.NewProductChannelListingRepo(db), outboxRepo, repositories.NewProductHistoryRepo(db)),
		CreateSKU:   apppms.NewCreateSKUUseCase(db, variantRepo, skuRepo, repositories.NewSKUPriceHistoryRepo(db), outboxRepo),
		CreateBatch: invapp.NewCreateBatchUseCase(db, batchRepo),
		Portion:     invapp.NewPortionFromBulkUseCase(db, skuRepo, movementRepo, outboxRepo),
		Stock:       invapp.NewGetScopedStockUseCase(db, movementRepo),
	})
	link := apppms.NewLinkExistingWeightPackagesUseCase(db, skuRepo, variantRepo,
		apppms.NewUpdateSKUUseCase(db, variantRepo, skuRepo, outboxRepo, repositories.NewSKUPriceHistoryRepo(db)))
	genweightportions.RegisterHandlers(rg, NewHandler(uc, link))
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

func (h *Handler) LinkExistingWeightPackages(c *gin.Context, id string) {
	var req genweightportions.LinkExistingWeightPackagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err.Error())
		return
	}
	result, err := h.link.Execute(c.Request.Context(), apppms.LinkExistingWeightPackagesInput{
		BulkSKUID: id, DryRun: req.DryRun != nil && *req.DryRun,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	out := genweightportions.LinkExistingWeightPackagesResult{
		Linked:  make([]genweightportions.LinkedWeightPackage, len(result.Linked)),
		Skipped: make([]genweightportions.SkippedWeightPackage, len(result.Skipped)),
		Errors:  result.Errors,
	}
	if out.Errors == nil {
		out.Errors = []string{}
	}
	for i, l := range result.Linked {
		out.Linked[i] = genweightportions.LinkedWeightPackage{
			SkuId: l.SKU.ID, SkuCode: l.SKU.SKUCode, VariantName: l.VariantName, WeightGrams: int(l.WeightGrams),
			PriceMinorUnits: int(l.PriceMinorUnits), PriceChanged: l.PriceChanged,
		}
	}
	for i, sk := range result.Skipped {
		out.Skipped[i] = genweightportions.SkippedWeightPackage{
			SkuId: sk.SKU.ID, SkuCode: sk.SKU.SKUCode, VariantName: sk.VariantName,
			Reason: genweightportions.SkippedWeightPackageReason(sk.Reason),
		}
	}
	middleware.OK(c, out)
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
	case errors.Is(err, pmsDomain.ErrSKUNotFound), errors.Is(err, invDomain.ErrBatchNotFound),
		errors.Is(err, pmsDomain.ErrVariantNotFound), errors.Is(err, pmsDomain.ErrProductNotFound):
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
