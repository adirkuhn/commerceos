package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForceAcceptStock_reconcilesEveryPendingLinkedLineItemAndResolvesTheOrder(t *testing.T) {
	db := newMockGormDBN(t, 5) // header tx + 2 link movements + setOrderNeedsStockReview + freezeCostAtSale
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
		o.Status = omsDomain.SalesOrderStatusProcessing
		o.NeedsStockReview = true
	})}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SKUID: strPtr("sku-1"), Quantity: 2, StockStatus: omsDomain.StockStatusPending},
		{ID: "li-2", SKUID: strPtr("sku-2"), Quantity: 1, StockStatus: omsDomain.StockStatusDecremented},
	}}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewForceAcceptStockUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, recordMovement)

	result, err := uc.Execute(context.Background(), "sales-order-1")

	require.NoError(t, err)
	assert.Equal(t, 1, result.Resolved, "only li-1 was pending -- li-2 was already decremented")
	require.Len(t, recordMovement.calls, 1)
	assert.Equal(t, "sku-1", recordMovement.calls[0].SKUID)
	assert.Equal(t, int64(2), recordMovement.calls[0].Quantity)
	assert.True(t, recordMovement.calls[0].ReconcileShortfall, "force-accept bypasses the insufficiency check entirely")
	require.Len(t, lineItemRepo.updateCalls, 1)
	assert.Equal(t, "li-1", lineItemRepo.updateCalls[0].ID)
	assert.Equal(t, string(omsDomain.StockStatusDecremented), lineItemRepo.updateCalls[0].Fields["stock_status"])
	require.NotEmpty(t, salesOrderRepo.updateCalls)
	last := salesOrderRepo.updateCalls[len(salesOrderRepo.updateCalls)-1]
	assert.Equal(t, false, last.Fields["needs_stock_review"], "force-accept always resolves the order -- ReconcileShortfall never leaves anything pending")
}

func TestForceAcceptStock_skipsUnlinkedLineItems(t *testing.T) {
	db := newMockGormDBN(t, 2) // header tx + setOrderNeedsStockReview -- nothing to reconcile
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
		o.Status = omsDomain.SalesOrderStatusProcessing
	})}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SKUID: nil, Quantity: 1, StockStatus: omsDomain.StockStatusPending},
	}}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewForceAcceptStockUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, recordMovement)

	result, err := uc.Execute(context.Background(), "sales-order-1")

	require.NoError(t, err)
	assert.Equal(t, 0, result.Resolved, "no sku_id -- force-accept can't reconcile stock for a line with nothing to draw from, that's LinkLineItemSKUUseCase's job")
	assert.Empty(t, recordMovement.calls)
}

func TestForceAcceptStock_returnsNotFoundForAMissingOrder(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewForceAcceptStockUseCase(db, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, &stubRecordMovement{})

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}

func TestForceAcceptStock_propagatesAGenuineMovementError(t *testing.T) {
	db := newMockGormDBN(t, 1) // header tx only -- aborts before setOrderNeedsStockReview
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
		o.Status = omsDomain.SalesOrderStatusProcessing
	})}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SKUID: strPtr("sku-1"), Quantity: 1, StockStatus: omsDomain.StockStatusPending},
	}}
	recordMovement := &stubRecordMovement{err: assert.AnError}
	uc := appoms.NewForceAcceptStockUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, recordMovement)

	_, err := uc.Execute(context.Background(), "sales-order-1")

	assert.ErrorIs(t, err, assert.AnError)
}
