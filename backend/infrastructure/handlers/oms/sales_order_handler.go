package oms

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

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
	var statuses []omsDomain.SalesOrderStatus
	if params.Status != nil {
		statuses = make([]omsDomain.SalesOrderStatus, len(*params.Status))
		for i, s := range *params.Status {
			statuses[i] = omsDomain.SalesOrderStatus(s)
		}
	}
	var sources []omsDomain.SalesOrderSource
	if params.Source != nil {
		sources = make([]omsDomain.SalesOrderSource, len(*params.Source))
		for i, s := range *params.Source {
			sources[i] = omsDomain.SalesOrderSource(s)
		}
	}
	result, err := h.listSalesOrders.Execute(c.Request.Context(), appoms.ListSalesOrdersInput{
		Page: page, PerPage: perPage, SortBy: sortBy, SortDir: sortDir,
		NeedsStockReview: params.NeedsStockReview, NeedsSKUReview: params.NeedsSkuReview,
		Fulfilled: params.Fulfilled, Packed: params.Packed, AwaitingPicking: params.AwaitingPicking != nil && *params.AwaitingPicking, Status: statuses, Source: sources, CustomerID: params.CustomerId, QuoteID: quoteIDParam(params.QuoteId),
		ShipDue: params.ShipDue != nil && *params.ShipDue,
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
	resp := genoms.SalesOrderDetail{SalesOrder: toGenSalesOrder(result.SalesOrder), LineItems: lineItems, Payments: toGenSalesOrderPayments(result.SalesOrder.Payments)}
	c.JSON(http.StatusOK, genoms.SalesOrderDetailResponse{Data: &resp})
}

func toGenSalesOrderPayments(payments omsDomain.SalesOrderPayments) []genoms.SalesOrderPayment {
	out := make([]genoms.SalesOrderPayment, len(payments))
	for i, p := range payments {
		out[i] = genoms.SalesOrderPayment{Method: p.Method, AmountMinorUnits: p.AmountMinorUnits, TransactionId: p.TransactionID}
	}
	return out
}

func (h *Handler) CreateSalesOrder(c *gin.Context) {
	var req genoms.CreateSalesOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
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
	resp := genoms.SalesOrderDetail{SalesOrder: toGenSalesOrder(result.SalesOrder), LineItems: lineItemsResp, Payments: toGenSalesOrderPayments(result.SalesOrder.Payments)}
	c.JSON(http.StatusCreated, genoms.SalesOrderDetailResponse{Data: &resp})
}

func (h *Handler) UpdateSalesOrder(c *gin.Context, id genoms.Id) {
	var req genoms.UpdateSalesOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
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

func (h *Handler) BulkUpdateSalesOrderStatus(c *gin.Context) {
	var req genoms.BulkUpdateSalesOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	in := appoms.BulkUpdateSalesOrderStatusInput{OrderIDs: req.OrderIds, Fulfilled: req.Fulfilled}
	if req.Status != nil {
		status := omsDomain.SalesOrderStatus(*req.Status)
		in.Status = &status
	}
	result, err := h.bulkUpdateSalesOrderStatus.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	failed := make([]struct {
		Error   *string `json:"error,omitempty"`
		OrderId *string `json:"order_id,omitempty"`
	}, len(result.Failed))
	for i := range result.Failed {
		failed[i].OrderId = &result.Failed[i].OrderID
		failed[i].Error = &result.Failed[i].Error
	}
	resp := genoms.BulkUpdateSalesOrderStatusResult{Updated: &result.Updated, Failed: &failed}
	c.JSON(http.StatusOK, genoms.BulkUpdateSalesOrderStatusResponse{Data: &resp})
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
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
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

func (h *Handler) AddSalesOrderLineItem(c *gin.Context, id genoms.Id) {
	var req genoms.AddSalesOrderLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	li, err := h.addSalesOrderLineItem.Execute(c.Request.Context(), appoms.AddSalesOrderLineItemInput{
		SalesOrderID: id, SKUID: req.SkuId, Name: strOrEmpty(req.Name), Quantity: req.Quantity, PriceMinorUnits: req.PriceMinorUnits,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenSalesOrderLineItem(*li)
	c.JSON(http.StatusCreated, genoms.SalesOrderLineItemResponse{Data: &resp})
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
	paymentReference := o.PaymentReference()
	return genoms.SalesOrder{
		Id: o.ID, Source: genoms.SalesOrderSource(o.Source), ExternalOrderId: o.ExternalOrderID, Number: &o.Number, OrderKey: &o.OrderKey,
		Status: genoms.SalesOrderStatus(o.Status), WarehouseId: o.WarehouseID, CustomerId: o.CustomerID, QuoteId: o.QuoteID,
		Billing: &billing, Shipping: &shipping, CustomerNote: &o.CustomerNote, CurrencyCode: o.CurrencyCode,
		DiscountTotalMinorUnits: &o.DiscountTotalMinorUnits, ShippingTotalMinorUnits: &o.ShippingTotalMinorUnits,
		TaxTotalMinorUnits: &o.TaxTotalMinorUnits, TotalMinorUnits: o.TotalMinorUnits,
		PaymentMethod: &o.PaymentMethod, PaymentMethodTitle: &o.PaymentMethodTitle,
		ShippingMethod: &o.ShippingMethod, ShippingMethodTitle: &o.ShippingMethodTitle, TransactionId: &o.TransactionID,
		PaymentProvider: &o.PaymentProvider, PaymentExternalId: &o.PaymentExternalID,
		PaymentStatus: genoms.PaymentStatus(o.PaymentStatus()), PaymentReference: &paymentReference,
		DatePaid: o.DatePaid, DateCompleted: o.DateCompleted, ShipByDate: toGenDate(o.ShipByDate), OccurredAt: o.OccurredAt, CreatedAt: o.CreatedAt,
		NeedsStockReview: &o.NeedsStockReview, NeedsSkuReview: &o.NeedsSKUReview, Fulfilled: &o.Fulfilled,
		Packed: &o.Packed,
	}
}

func toGenSalesOrderLineItem(li omsDomain.SalesOrderLineItem) genoms.SalesOrderLineItem {
	stockStatus := genoms.StockStatus(li.StockStatus)
	var soldExpiryDate *openapi_types.Date
	if li.SoldExpiryDate != nil {
		soldExpiryDate = &openapi_types.Date{Time: *li.SoldExpiryDate}
	}
	return genoms.SalesOrderLineItem{
		Id: li.ID, SkuId: li.SKUID, Name: &li.Name, SkuCode: &li.SKUCode, Barcode: &li.Barcode, Quantity: li.Quantity,
		StockStatus:     &stockStatus,
		PriceMinorUnits: &li.PriceMinorUnits, SubtotalMinorUnits: &li.SubtotalMinorUnits,
		SubtotalTaxMinorUnits: &li.SubtotalTaxMinorUnits, TotalMinorUnits: &li.TotalMinorUnits,
		TotalTaxMinorUnits: &li.TotalTaxMinorUnits, ProductId: li.ProductID,
		SoldExpiryDate:       soldExpiryDate,
		CostAtSaleMinorUnits: li.CostAtSaleMinorUnits, CogsMinorUnits: li.CogsMinorUnits,
	}
}

func (h *Handler) ListSalesOrderActivities(c *gin.Context, id genoms.Id) {
	activities, err := h.listSalesOrderActivities.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genoms.SalesOrderActivity, len(activities))
	for i, a := range activities {
		var metadata map[string]interface{}
		if err := json.Unmarshal(a.Metadata, &metadata); err != nil && len(a.Metadata) > 0 {
			log.Printf("omsHandler: activity %s: corrupt metadata, omitting: %v", a.ID, err)
		}
		out[i] = genoms.SalesOrderActivity{
			Id: a.ID, SalesOrderId: a.SalesOrderID, UserId: a.UserID,
			ActivityType: a.ActivityType, Summary: a.Summary, Metadata: metadata, CreatedAt: a.CreatedAt,
		}
	}
	resp := genoms.SalesOrderActivityList{Activities: out}
	c.JSON(http.StatusOK, genoms.SalesOrderActivityListResponse{Data: &resp})
}

// toGenDate maps an optional calendar date to the API's `format: date`.
func toGenDate(d *time.Time) *openapi_types.Date {
	if d == nil {
		return nil
	}
	return &openapi_types.Date{Time: *d}
}

func quoteIDParam(id *openapi_types.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}
