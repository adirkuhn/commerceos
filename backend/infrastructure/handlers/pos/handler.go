package pos

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genpos "github.com/adirkuhn/commerceos/api/pos"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ── Use-case interfaces (defined at point of use — ISP) ───────────────────────

type listProductsUC interface {
	Execute(ctx context.Context, in apppms.ListProductsInput) (*apppms.ListProductsResult, error)
}
type listStockLevelsUC interface {
	Execute(ctx context.Context, in appinventory.ListStockLevelsInput) (*appinventory.ListStockLevelsResult, error)
}
type receivePOSSalesOrderUC interface {
	Execute(ctx context.Context, in appoms.ReceivePOSSalesOrderInput) (*omsDomain.SalesOrder, error)
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listProducts         listProductsUC
	listStockLevels      listStockLevelsUC
	receivePOSSalesOrder receivePOSSalesOrderUC
}

func NewHandler(listProducts listProductsUC, listStockLevels listStockLevelsUC, receivePOSSalesOrder receivePOSSalesOrderUC) *Handler {
	return &Handler{listProducts: listProducts, listStockLevels: listStockLevels, receivePOSSalesOrder: receivePOSSalesOrder}
}

var _ genpos.ServerInterface = (*Handler)(nil)

// Register is the production wiring. Callers must mount rg on a router
// group using middleware.POSLicense, never middleware.JWT -- see
// api/pos/generate.go's doc comment.
func Register(rg gin.IRouter, db *gorm.DB) {
	productRepo := repositories.NewProductRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	recordMovement := appinventory.NewRecordStockMovementUseCase(db, movementRepo, outboxRepo)
	h := NewHandler(
		apppms.NewListProductsUseCase(db, productRepo),
		appinventory.NewListStockLevelsUseCase(db, movementRepo, batchRepo),
		appoms.NewReceivePOSSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, recordMovement),
	)
	genpos.RegisterHandlers(rg, h)
}

func (h *Handler) ListPOSProducts(c *gin.Context, params genpos.ListPOSProductsParams) {
	page, perPage := 1, 20
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	result, err := h.listProducts.Execute(c.Request.Context(), apppms.ListProductsInput{
		Page: page, PerPage: perPage, UpdatedAfter: params.UpdatedAfter,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	products := make([]genpos.POSProduct, len(result.Products))
	for i, p := range result.Products {
		products[i] = genpos.POSProduct{
			Id: p.ID, Name: p.Name, Slug: p.Slug, Status: string(p.Status),
			HasVariants: p.HasVariants, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
		}
	}
	resp := genpos.POSProductList{Products: products, Total: result.Total}
	c.JSON(http.StatusOK, genpos.POSProductListResponse{Data: &resp})
}

func (h *Handler) ListPOSStockLevels(c *gin.Context, params genpos.ListPOSStockLevelsParams) {
	page, perPage := 1, 50
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	result, err := h.listStockLevels.Execute(c.Request.Context(), appinventory.ListStockLevelsInput{
		ChangedSince: params.ChangedSince, Page: page, PerPage: perPage,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	levels := make([]genpos.POSStockLevel, len(result.Levels))
	for i, l := range result.Levels {
		levels[i] = genpos.POSStockLevel{SkuId: l.SKUID, Total: l.Total, Syncable: l.Syncable}
	}
	resp := genpos.POSStockLevelList{Levels: levels, Total: result.Total}
	c.JSON(http.StatusOK, genpos.POSStockLevelListResponse{Data: &resp})
}

func (h *Handler) CreatePOSSalesOrder(c *gin.Context) {
	var req genpos.CreatePOSSalesOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	lineItems := make([]appoms.ReceivePOSSalesOrderLineItemInput, len(req.LineItems))
	for i, li := range req.LineItems {
		lineItems[i] = appoms.ReceivePOSSalesOrderLineItemInput{
			SKUID: li.SkuId, Quantity: li.Quantity, UnitPriceMinorUnits: li.UnitPriceMinorUnits,
		}
	}
	order, err := h.receivePOSSalesOrder.Execute(c.Request.Context(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: req.ExternalOrderId, WarehouseID: req.WarehouseId, CurrencyCode: req.CurrencyCode,
		OccurredAt: req.OccurredAt, LineItems: lineItems,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := genpos.POSSalesOrder{
		Id: order.ID, ExternalOrderId: order.ExternalOrderID, WarehouseId: order.WarehouseID,
		CurrencyCode: order.CurrencyCode, TotalMinorUnits: order.TotalMinorUnits,
		OccurredAt: order.OccurredAt, CreatedAt: order.CreatedAt,
	}
	c.JSON(http.StatusOK, genpos.POSSalesOrderResponse{Data: &resp})
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, omsDomain.ErrEmptyLineItems), errors.Is(err, invDomain.ErrInsufficientStock),
		errors.Is(err, invDomain.ErrInvalidQuantity):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		log.Printf("pos: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
