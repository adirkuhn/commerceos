package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryStockDecrement_resolvesAPendingLineItemAndClearsNeedsStockReview(t *testing.T) {
	db := newMockGormDBN(t, 4) // header tx (find order + list line items) + link movement + setLineItemStockStatus + setOrderNeedsStockReview
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
		o.Status = omsDomain.SalesOrderStatusProcessing
		o.NeedsStockReview = true
	})}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SKUID: strPtr("sku-1"), Quantity: 2, StockStatus: omsDomain.StockStatusPending},
	}}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewRetryStockDecrementUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, recordMovement)

	result, err := uc.Execute(context.Background(), "sales-order-1")

	require.NoError(t, err)
	assert.Equal(t, 1, result.Resolved)
	assert.Equal(t, 0, result.StillPending)
	require.Len(t, recordMovement.calls, 1)
	assert.Equal(t, int64(-2), recordMovement.calls[0].QuantityDelta)
	require.Len(t, lineItemRepo.updateCalls, 1)
	assert.Equal(t, string(omsDomain.StockStatusDecremented), lineItemRepo.updateCalls[0].Fields["stock_status"])
	require.NotEmpty(t, salesOrderRepo.updateCalls)
	assert.Equal(t, false, salesOrderRepo.updateCalls[len(salesOrderRepo.updateCalls)-1].Fields["needs_stock_review"])
}

func TestRetryStockDecrement_stillInsufficientKeepsPendingAndReportsPartialResult(t *testing.T) {
	db := newMockGormDBN(t, 2) // header tx + setOrderNeedsStockReview (no link/setLineItemStockStatus -- movement never succeeded)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
		o.Status = omsDomain.SalesOrderStatusProcessing
	})}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SKUID: strPtr("sku-1"), Quantity: 2, StockStatus: omsDomain.StockStatusPending},
	}}
	recordMovement := &stubRecordMovement{err: invDomain.ErrInsufficientStock}
	uc := appoms.NewRetryStockDecrementUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, recordMovement)

	result, err := uc.Execute(context.Background(), "sales-order-1")

	require.NoError(t, err, "still-insufficient is a partial-success outcome, not a use-case error")
	assert.Equal(t, 0, result.Resolved)
	assert.Equal(t, 1, result.StillPending)
	assert.Empty(t, lineItemRepo.updateCalls, "still pending -- stock_status must not change")
	require.NotEmpty(t, salesOrderRepo.updateCalls)
	assert.Equal(t, true, salesOrderRepo.updateCalls[len(salesOrderRepo.updateCalls)-1].Fields["needs_stock_review"])
}

func TestRetryStockDecrement_skipsAlreadyDecrementedAndUnlinkedLineItems(t *testing.T) {
	db := newMockGormDBN(t, 2) // header tx + setOrderNeedsStockReview -- no line item is eligible to retry
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
		o.Status = omsDomain.SalesOrderStatusProcessing
	})}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SKUID: strPtr("sku-1"), Quantity: 1, StockStatus: omsDomain.StockStatusDecremented},
		{ID: "li-2", SKUID: nil, Quantity: 1, StockStatus: omsDomain.StockStatusPending},
	}}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewRetryStockDecrementUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, recordMovement)

	result, err := uc.Execute(context.Background(), "sales-order-1")

	require.NoError(t, err)
	assert.Equal(t, 0, result.Resolved)
	assert.Equal(t, 0, result.StillPending)
	assert.Empty(t, recordMovement.calls, "li-1 already decremented, li-2 has no sku_id yet -- neither is retryable")
}

func TestRetryStockDecrement_rejectsAnOrderWhoseStatusDoesNotReduceStock(t *testing.T) {
	db := newMockGormDB(t) // single tx -- rejected before the movement loop
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
		o.Status = omsDomain.SalesOrderStatusCancelled
	})}
	uc := appoms.NewRetryStockDecrementUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubRecordMovement{})

	_, err := uc.Execute(context.Background(), "sales-order-1")

	assert.ErrorIs(t, err, omsDomain.ErrOrderNotStockReducing)
}
