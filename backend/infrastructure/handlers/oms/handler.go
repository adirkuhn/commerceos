package oms

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ── Use-case interfaces (defined at point of use — ISP) ───────────────────────

type listSalesOrdersUC interface {
	Execute(ctx context.Context, in appoms.ListSalesOrdersInput) (*appoms.ListSalesOrdersResult, error)
}
type getSalesOrderUC interface {
	Execute(ctx context.Context, id string) (*appoms.GetSalesOrderResult, error)
}
type createSalesOrderUC interface {
	Execute(ctx context.Context, in appoms.CreateSalesOrderInput) (*appoms.CreateSalesOrderResult, error)
}
type updateSalesOrderUC interface {
	Execute(ctx context.Context, id string, in appoms.UpdateSalesOrderInput) (*omsDomain.SalesOrder, error)
}
type deleteSalesOrderUC interface {
	Execute(ctx context.Context, id string) error
}
type linkLineItemSKUUC interface {
	Execute(ctx context.Context, in appoms.LinkLineItemSKUInput) (*omsDomain.SalesOrder, error)
}
type retryStockDecrementUC interface {
	Execute(ctx context.Context, salesOrderID string) (*appoms.RetryStockDecrementResult, error)
}
type forceAcceptStockUC interface {
	Execute(ctx context.Context, salesOrderID string) (*appoms.ForceAcceptStockResult, error)
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listSalesOrders     listSalesOrdersUC
	getSalesOrder       getSalesOrderUC
	createSalesOrder    createSalesOrderUC
	updateSalesOrder    updateSalesOrderUC
	deleteSalesOrder    deleteSalesOrderUC
	linkLineItemSKU     linkLineItemSKUUC
	retryStockDecrement retryStockDecrementUC
	forceAcceptStock    forceAcceptStockUC
}

func NewHandler(listSalesOrders listSalesOrdersUC, getSalesOrder getSalesOrderUC, createSalesOrder createSalesOrderUC, updateSalesOrder updateSalesOrderUC, deleteSalesOrder deleteSalesOrderUC, linkLineItemSKU linkLineItemSKUUC, retryStockDecrement retryStockDecrementUC, forceAcceptStock forceAcceptStockUC) *Handler {
	return &Handler{
		listSalesOrders: listSalesOrders, getSalesOrder: getSalesOrder,
		createSalesOrder: createSalesOrder, updateSalesOrder: updateSalesOrder, deleteSalesOrder: deleteSalesOrder,
		linkLineItemSKU: linkLineItemSKU, retryStockDecrement: retryStockDecrement, forceAcceptStock: forceAcceptStock,
	}
}

var _ genoms.ServerInterface = (*Handler)(nil)

// Register is the production wiring. Callers mount rg under the normal
// human-JWT-authenticated /api/v1 group -- this is staff creating/editing
// sales orders from the admin UI. The POS's own push path
// (ReceivePOSSalesOrderUseCase) is separate, under api/pos, authenticated
// by the POS's lm2 license instead -- see api/oms/generate.go's doc comment.
func Register(rg gin.IRouter, db *gorm.DB) {
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	linkRepo := repositories.NewSalesOrderLineItemStockMovementRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)
	skuRepo := repositories.NewSKURepo(db)
	recordMovement := appinventory.NewRecordStockMovementUseCase(db, movementRepo, outboxRepo)
	allocateStock := appinventory.NewAllocateSaleStockUseCase(db, movementRepo, batchRepo, outboxRepo)

	h := NewHandler(
		appoms.NewListSalesOrdersUseCase(db, salesOrderRepo),
		appoms.NewGetSalesOrderUseCase(db, salesOrderRepo, lineItemRepo),
		appoms.NewCreateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, linkRepo, skuRepo, allocateStock),
		appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, linkRepo, allocateStock, recordMovement),
		appoms.NewDeleteSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, linkRepo, recordMovement),
		appoms.NewLinkLineItemSKUUseCase(db, salesOrderRepo, lineItemRepo, skuRepo),
		appoms.NewRetryStockDecrementUseCase(db, salesOrderRepo, lineItemRepo, linkRepo, allocateStock),
		appoms.NewForceAcceptStockUseCase(db, salesOrderRepo, lineItemRepo, linkRepo, allocateStock),
	)
	genoms.RegisterHandlers(rg, h)
}

func (h *Handler) ListSalesOrders(c *gin.Context, params genoms.ListSalesOrdersParams) {
	page, perPage := 1, 20
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	var sortBy omsDomain.SalesOrderSortField
	if params.SortBy != nil {
		sortBy = omsDomain.SalesOrderSortField(*params.SortBy)
	}
	var sortDir omsDomain.SortDirection
	if params.SortDir != nil {
		sortDir = omsDomain.SortDirection(*params.SortDir)
	}
	result, err := h.listSalesOrders.Execute(c.Request.Context(), appoms.ListSalesOrdersInput{
		Page: page, PerPage: perPage, SortBy: sortBy, SortDir: sortDir,
		NeedsStockReview: params.NeedsStockReview, NeedsSKUReview: params.NeedsSkuReview,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	orders := make([]genoms.SalesOrder, len(result.SalesOrders))
	for i, o := range result.SalesOrders {
		orders[i] = toGenSalesOrder(o)
	}
	resp := genoms.SalesOrderList{SalesOrders: orders, Total: result.Total}
	c.JSON(http.StatusOK, genoms.SalesOrderListResponse{Data: &resp})
}

func (h *Handler) GetSalesOrder(c *gin.Context, id genoms.Id) {
	result, err := h.getSalesOrder.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	lineItems := make([]genoms.SalesOrderLineItem, len(result.LineItems))
	for i, li := range result.LineItems {
		lineItems[i] = toGenSalesOrderLineItem(li)
	}
	resp := genoms.SalesOrderDetail{SalesOrder: toGenSalesOrder(result.SalesOrder), LineItems: lineItems}
	c.JSON(http.StatusOK, genoms.SalesOrderDetailResponse{Data: &resp})
}

func (h *Handler) CreateSalesOrder(c *gin.Context) {
	var req genoms.CreateSalesOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	lineItems := make([]appoms.CreateSalesOrderLineItemInput, len(req.LineItems))
	for i, li := range req.LineItems {
		lineItems[i] = appoms.CreateSalesOrderLineItemInput{
			SKUID: li.SkuId, Name: strOrEmpty(li.Name), SKUCode: strOrEmpty(li.SkuCode), Quantity: li.Quantity,
			PriceMinorUnits: li.PriceMinorUnits, SubtotalMinorUnits: int64OrZero(li.SubtotalMinorUnits),
			SubtotalTaxMinorUnits: int64OrZero(li.SubtotalTaxMinorUnits), TotalMinorUnits: int64OrZero(li.TotalMinorUnits),
			TotalTaxMinorUnits: int64OrZero(li.TotalTaxMinorUnits),
		}
	}
	var status omsDomain.SalesOrderStatus
	if req.Status != nil {
		status = omsDomain.SalesOrderStatus(*req.Status)
	}
	result, err := h.createSalesOrder.Execute(c.Request.Context(), appoms.CreateSalesOrderInput{
		Number: strOrEmpty(req.Number), OrderKey: strOrEmpty(req.OrderKey), Status: status,
		WarehouseID: req.WarehouseId, CustomerID: req.CustomerId,
		Billing: fromGenAddress(req.Billing), Shipping: fromGenAddress(req.Shipping), CustomerNote: strOrEmpty(req.CustomerNote),
		CurrencyCode: req.CurrencyCode, DiscountTotalMinorUnits: int64OrZero(req.DiscountTotalMinorUnits),
		ShippingTotalMinorUnits: int64OrZero(req.ShippingTotalMinorUnits), TaxTotalMinorUnits: int64OrZero(req.TaxTotalMinorUnits),
		TotalMinorUnits: int64OrZero(req.TotalMinorUnits), PaymentMethod: strOrEmpty(req.PaymentMethod),
		PaymentMethodTitle: strOrEmpty(req.PaymentMethodTitle),
		ShippingMethod:     strOrEmpty(req.ShippingMethod), ShippingMethodTitle: strOrEmpty(req.ShippingMethodTitle),
		TransactionID: strOrEmpty(req.TransactionId),
		DatePaid:      req.DatePaid, DateCompleted: req.DateCompleted,
		OccurredAt: req.OccurredAt, LineItems: lineItems,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	lineItemsResp := make([]genoms.SalesOrderLineItem, len(result.LineItems))
	for i, li := range result.LineItems {
		lineItemsResp[i] = toGenSalesOrderLineItem(li)
	}
	resp := genoms.SalesOrderDetail{SalesOrder: toGenSalesOrder(result.SalesOrder), LineItems: lineItemsResp}
	c.JSON(http.StatusCreated, genoms.SalesOrderDetailResponse{Data: &resp})
}

func (h *Handler) UpdateSalesOrder(c *gin.Context, id genoms.Id) {
	var req genoms.UpdateSalesOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	in := appoms.UpdateSalesOrderInput{
		Number: req.Number, OrderKey: req.OrderKey, CustomerID: req.CustomerId,
		Billing: fromGenAddressPtr(req.Billing), Shipping: fromGenAddressPtr(req.Shipping),
		CustomerNote: req.CustomerNote, PaymentMethod: req.PaymentMethod, PaymentMethodTitle: req.PaymentMethodTitle,
		ShippingMethod: req.ShippingMethod, ShippingMethodTitle: req.ShippingMethodTitle,
		TransactionID: req.TransactionId, DatePaid: req.DatePaid, DateCompleted: req.DateCompleted,
	}
	if req.Status != nil {
		status := omsDomain.SalesOrderStatus(*req.Status)
		in.Status = &status
	}
	order, err := h.updateSalesOrder.Execute(c.Request.Context(), id, in)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenSalesOrder(*order)
	c.JSON(http.StatusOK, genoms.SalesOrderResponse{Data: &resp})
}

func (h *Handler) DeleteSalesOrder(c *gin.Context, id genoms.Id) {
	err := h.deleteSalesOrder.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) LinkLineItemSKU(c *gin.Context, id genoms.Id, lineItemId string) {
	var req genoms.LinkLineItemSKURequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	order, err := h.linkLineItemSKU.Execute(c.Request.Context(), appoms.LinkLineItemSKUInput{
		SalesOrderID: id, LineItemID: lineItemId, SKUID: req.SkuId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenSalesOrder(*order)
	c.JSON(http.StatusOK, genoms.SalesOrderResponse{Data: &resp})
}

func (h *Handler) RetryStockDecrement(c *gin.Context, id genoms.Id) {
	result, err := h.retryStockDecrement.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	order := toGenSalesOrder(result.Order)
	resp := genoms.RetryStockDecrementResult{SalesOrder: order, Resolved: result.Resolved, StillPending: result.StillPending}
	c.JSON(http.StatusOK, genoms.RetryStockDecrementResponse{Data: &resp})
}

func (h *Handler) ForceAcceptStock(c *gin.Context, id genoms.Id) {
	result, err := h.forceAcceptStock.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	order := toGenSalesOrder(result.Order)
	resp := genoms.ForceAcceptStockResult{SalesOrder: order, Resolved: result.Resolved}
	c.JSON(http.StatusOK, genoms.ForceAcceptStockResponse{Data: &resp})
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func int64OrZero(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func fromGenAddress(a *genoms.Address) omsDomain.Address {
	if a == nil {
		return omsDomain.Address{}
	}
	return omsDomain.Address{
		FirstName: strOrEmpty(a.FirstName), LastName: strOrEmpty(a.LastName), Company: strOrEmpty(a.Company),
		Address1: strOrEmpty(a.Address1), Address2: strOrEmpty(a.Address2), City: strOrEmpty(a.City),
		State: strOrEmpty(a.State), Postcode: strOrEmpty(a.Postcode), Country: strOrEmpty(a.Country),
		Email: strOrEmpty(a.Email), Phone: strOrEmpty(a.Phone),
	}
}

// fromGenAddressPtr is fromGenAddress but preserves "field omitted" (nil)
// vs "field present" for UpdateSalesOrderUseCase's partial-update
// semantics -- fromGenAddress alone can't distinguish those since it
// always returns a concrete zero-valued Address for a nil input.
func fromGenAddressPtr(a *genoms.Address) *omsDomain.Address {
	if a == nil {
		return nil
	}
	addr := fromGenAddress(a)
	return &addr
}

func toGenAddress(a omsDomain.Address) genoms.Address {
	return genoms.Address{
		FirstName: &a.FirstName, LastName: &a.LastName, Company: &a.Company,
		Address1: &a.Address1, Address2: &a.Address2, City: &a.City,
		State: &a.State, Postcode: &a.Postcode, Country: &a.Country,
		Email: &a.Email, Phone: &a.Phone,
	}
}

func toGenSalesOrder(o omsDomain.SalesOrder) genoms.SalesOrder {
	billing := toGenAddress(o.Billing)
	shipping := toGenAddress(o.Shipping)
	return genoms.SalesOrder{
		Id: o.ID, Source: string(o.Source), ExternalOrderId: o.ExternalOrderID, Number: &o.Number, OrderKey: &o.OrderKey,
		Status: genoms.SalesOrderStatus(o.Status), WarehouseId: o.WarehouseID, CustomerId: o.CustomerID,
		Billing: &billing, Shipping: &shipping, CustomerNote: &o.CustomerNote, CurrencyCode: o.CurrencyCode,
		DiscountTotalMinorUnits: &o.DiscountTotalMinorUnits, ShippingTotalMinorUnits: &o.ShippingTotalMinorUnits,
		TaxTotalMinorUnits: &o.TaxTotalMinorUnits, TotalMinorUnits: o.TotalMinorUnits,
		PaymentMethod: &o.PaymentMethod, PaymentMethodTitle: &o.PaymentMethodTitle,
		ShippingMethod: &o.ShippingMethod, ShippingMethodTitle: &o.ShippingMethodTitle, TransactionId: &o.TransactionID,
		DatePaid: o.DatePaid, DateCompleted: o.DateCompleted, OccurredAt: o.OccurredAt, CreatedAt: o.CreatedAt,
		NeedsStockReview: &o.NeedsStockReview, NeedsSkuReview: &o.NeedsSKUReview,
	}
}

func toGenSalesOrderLineItem(li omsDomain.SalesOrderLineItem) genoms.SalesOrderLineItem {
	stockStatus := genoms.StockStatus(li.StockStatus)
	return genoms.SalesOrderLineItem{
		Id: li.ID, SkuId: li.SKUID, Name: &li.Name, SkuCode: &li.SKUCode, Barcode: &li.Barcode, Quantity: li.Quantity,
		StockStatus:     &stockStatus,
		PriceMinorUnits: &li.PriceMinorUnits, SubtotalMinorUnits: &li.SubtotalMinorUnits,
		SubtotalTaxMinorUnits: &li.SubtotalTaxMinorUnits, TotalMinorUnits: &li.TotalMinorUnits,
		TotalTaxMinorUnits: &li.TotalTaxMinorUnits,
	}
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, omsDomain.ErrSalesOrderNotFound), errors.Is(err, omsDomain.ErrSalesOrderLineItemNotFound),
		errors.Is(err, pmsDomain.ErrSKUNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, omsDomain.ErrEmptyLineItems), errors.Is(err, invDomain.ErrInsufficientStock),
		errors.Is(err, invDomain.ErrInvalidQuantity), errors.Is(err, omsDomain.ErrLineItemNotOnOrder),
		errors.Is(err, omsDomain.ErrOrderNotStockReducing), errors.Is(err, omsDomain.ErrDuplicateExternalOrder):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		log.Printf("oms: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
