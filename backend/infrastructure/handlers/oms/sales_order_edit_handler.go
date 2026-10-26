package oms

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

type editSalesOrderUC interface {
	Execute(ctx context.Context, in appoms.EditSalesOrderInput) (*appoms.CreateSalesOrderResult, error)
}

// WithEditSalesOrder wires PUT /sales-orders/{id}.
func (h *Handler) WithEditSalesOrder(uc editSalesOrderUC) *Handler {
	h.editSalesOrder = uc
	return h
}

func (h *Handler) EditSalesOrder(c *gin.Context, id genoms.Id) {
	var req genoms.EditSalesOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	result, err := h.editSalesOrder.Execute(c.Request.Context(), appoms.EditSalesOrderInput{
		ID: id, Number: strOrEmpty(req.Number), WarehouseID: req.WarehouseId, CustomerID: req.CustomerId,
		Billing: fromGenAddress(req.Billing), Shipping: fromGenAddress(req.Shipping), CustomerNote: strOrEmpty(req.CustomerNote),
		ShippingTotalMinorUnits: int64OrZero(req.ShippingTotalMinorUnits), FeeLabel: strOrEmpty(req.FeeLabel),
		FeeTotalMinorUnits: int64OrZero(req.FeeTotalMinorUnits), TaxTotalMinorUnits: int64OrZero(req.TaxTotalMinorUnits),
		TotalMinorUnits: int64OrZero(req.TotalMinorUnits), OccurredAt: req.OccurredAt, LineItems: fromGenCreateLineItems(req.LineItems),
	})
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
