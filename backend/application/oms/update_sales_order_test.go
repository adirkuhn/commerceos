package oms_test

import (
	"context"
	"testing"
	"time"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func existingOrder(overrides func(*omsDomain.SalesOrder)) *omsDomain.SalesOrder {
	o := &omsDomain.SalesOrder{
		ID: "sales-order-1", WarehouseID: "warehouse-1", Status: omsDomain.SalesOrderStatusPending,
		StockDecremented: false, OccurredAt: time.Now(),
	}
	if overrides != nil {
		overrides(o)
	}
	return o
}

func TestUpdateSalesOrder_transitioningToProcessingDecrementsStock(t *testing.T) {
	db := newMockGormDBN(t, 2)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(nil)}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{SKUID: strPtr("sku-1"), Quantity: 3},
	}}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, recordMovement)

	status := omsDomain.SalesOrderStatusProcessing
	_, err := uc.Execute(context.Background(), "sales-order-1", appoms.UpdateSalesOrderInput{Status: &status})

	require.NoError(t, err)
	require.Len(t, recordMovement.calls, 1)
	assert.Equal(t, "sku-1", recordMovement.calls[0].SKUID)
	assert.Equal(t, int64(-3), recordMovement.calls[0].QuantityDelta, "stock now being taken")
	assert.Equal(t, true, salesOrderRepo.gotUpdateFields["stock_decremented"])
	assert.Equal(t, string(omsDomain.SalesOrderStatusProcessing), salesOrderRepo.gotUpdateFields["status"])
}

func TestUpdateSalesOrder_cancellingARestockedOrderReturnsStock(t *testing.T) {
	db := newMockGormDBN(t, 2)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
		o.Status = omsDomain.SalesOrderStatusProcessing
		o.StockDecremented = true
	})}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{SKUID: strPtr("sku-1"), Quantity: 3},
	}}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, recordMovement)

	status := omsDomain.SalesOrderStatusCancelled
	_, err := uc.Execute(context.Background(), "sales-order-1", appoms.UpdateSalesOrderInput{Status: &status})

	require.NoError(t, err)
	require.Len(t, recordMovement.calls, 1)
	assert.Equal(t, int64(3), recordMovement.calls[0].QuantityDelta, "positive -- stock coming back")
	assert.Equal(t, false, salesOrderRepo.gotUpdateFields["stock_decremented"])
}

func TestUpdateSalesOrder_refundingDoesNotAutoRestock(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
		o.Status = omsDomain.SalesOrderStatusCompleted
		o.StockDecremented = true
	})}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, recordMovement)

	status := omsDomain.SalesOrderStatusRefunded
	_, err := uc.Execute(context.Background(), "sales-order-1", appoms.UpdateSalesOrderInput{Status: &status})

	require.NoError(t, err)
	assert.Empty(t, recordMovement.calls, "refund disposition is a separate staff decision, not automatic")
}

func TestUpdateSalesOrder_processingToCompletedIsAStatusChangeWithNoStockMovement(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
		o.Status = omsDomain.SalesOrderStatusProcessing
		o.StockDecremented = true
	})}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, recordMovement)

	status := omsDomain.SalesOrderStatusCompleted
	_, err := uc.Execute(context.Background(), "sales-order-1", appoms.UpdateSalesOrderInput{Status: &status})

	require.NoError(t, err)
	assert.Empty(t, recordMovement.calls, "already decremented -- both statuses are stock-reducing")
}

func TestUpdateSalesOrder_updatesHeaderFieldsWithoutTouchingStock(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(nil)}
	uc := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubRecordMovement{})

	note := "call before delivery"
	_, err := uc.Execute(context.Background(), "sales-order-1", appoms.UpdateSalesOrderInput{CustomerNote: &note})

	require.NoError(t, err)
	assert.Equal(t, "call before delivery", salesOrderRepo.gotUpdateFields["customer_note"])
	assert.NotContains(t, salesOrderRepo.gotUpdateFields, "status")
}

func TestUpdateSalesOrder_updatesShippingMethod(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(nil)}
	uc := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubRecordMovement{})

	method := "free_shipping"
	title := "Free shipping"
	updated, err := uc.Execute(context.Background(), "sales-order-1", appoms.UpdateSalesOrderInput{
		ShippingMethod: &method, ShippingMethodTitle: &title,
	})

	require.NoError(t, err)
	assert.Equal(t, "free_shipping", salesOrderRepo.gotUpdateFields["shipping_method"])
	assert.Equal(t, "Free shipping", salesOrderRepo.gotUpdateFields["shipping_method_title"])
	assert.Equal(t, "free_shipping", updated.ShippingMethod)
	assert.Equal(t, "Free shipping", updated.ShippingMethodTitle)
}

func TestUpdateSalesOrder_returnsNotFoundForAMissingOrder(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{}
	uc := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubRecordMovement{})

	note := "x"
	_, err := uc.Execute(context.Background(), "missing", appoms.UpdateSalesOrderInput{CustomerNote: &note})

	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}

func TestUpdateSalesOrder_propagatesAStockMovementError(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(nil)}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{SKUID: strPtr("sku-1"), Quantity: 3}}}
	recordMovement := &stubRecordMovement{err: invDomain.ErrInsufficientStock}
	uc := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, recordMovement)

	status := omsDomain.SalesOrderStatusProcessing
	_, err := uc.Execute(context.Background(), "sales-order-1", appoms.UpdateSalesOrderInput{Status: &status})

	assert.ErrorIs(t, err, invDomain.ErrInsufficientStock)
}
