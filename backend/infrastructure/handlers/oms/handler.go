package oms

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
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

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listSalesOrders listSalesOrdersUC
	getSalesOrder   getSalesOrderUC
}

func NewHandler(listSalesOrders listSalesOrdersUC, getSalesOrder getSalesOrderUC) *Handler {
	return &Handler{listSalesOrders: listSalesOrders, getSalesOrder: getSalesOrder}
}

var _ genoms.ServerInterface = (*Handler)(nil)

// Register is the production wiring. Callers mount rg under the normal
// human-JWT-authenticated /api/v1 group -- see api/oms/generate.go's doc
// comment for why the write side lives elsewhere.
func Register(rg gin.IRouter, db *gorm.DB) {
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	h := NewHandler(
		appoms.NewListSalesOrdersUseCase(db, salesOrderRepo),
		appoms.NewGetSalesOrderUseCase(db, salesOrderRepo, lineItemRepo),
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
	result, err := h.listSalesOrders.Execute(c.Request.Context(), appoms.ListSalesOrdersInput{Page: page, PerPage: perPage})
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
		PaymentMethod: &o.PaymentMethod, PaymentMethodTitle: &o.PaymentMethodTitle, TransactionId: &o.TransactionID,
		DatePaid: o.DatePaid, DateCompleted: o.DateCompleted, OccurredAt: o.OccurredAt, CreatedAt: o.CreatedAt,
	}
}

func toGenSalesOrderLineItem(li omsDomain.SalesOrderLineItem) genoms.SalesOrderLineItem {
	return genoms.SalesOrderLineItem{
		Id: li.ID, SkuId: li.SKUID, Name: &li.Name, SkuCode: &li.SKUCode, Quantity: li.Quantity,
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
	case errors.Is(err, omsDomain.ErrSalesOrderNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	default:
		log.Printf("oms: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
