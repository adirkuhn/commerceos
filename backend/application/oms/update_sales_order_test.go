package oms_test

import (
	"context"
	"errors"
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
	db := newMockGormDBN(t, 3) // header + link movement + setOrderNeedsStockReview
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
	assert.Equal(t, true, salesOrderRepo.gotTrySwapTo, "swap flips stock_decremented true, via TrySetStockDecremented not the header fields map")
	require.NotEmpty(t, salesOrderRepo.updateCalls)
	headerFields := salesOrderRepo.updateCalls[0].Fields
	assert.NotContains(t, headerFields, "stock_decremented", "must never be set through the unconditional fields map -- that's the race the fix closes")
	assert.Equal(t, string(omsDomain.SalesOrderStatusProcessing), headerFields["status"])
}

// TestUpdateSalesOrder_losingTheDecrementRaceIsANoOp proves that when
// TrySetStockDecremented reports the swap didn't happen (another concurrent
// call already flipped StockDecremented first -- see the race this closes),
// this call skips the movement loop entirely rather than double-decrementing.
func TestUpdateSalesOrder_losingTheDecrementRaceIsANoOp(t *testing.T) {
	db := newMockGormDB(t) // header tx only -- no movement/link/review tx should ever open
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(nil), trySwapResultSet: true, trySwapResult: false}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{SKUID: strPtr("sku-1"), Quantity: 3},
	}}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, recordMovement)

	status := omsDomain.SalesOrderStatusProcessing
	_, err := uc.Execute(context.Background(), "sales-order-1", appoms.UpdateSalesOrderInput{Status: &status})

	require.NoError(t, err)
	assert.Equal(t, 1, salesOrderRepo.trySwapCalls)
	assert.Empty(t, recordMovement.calls, "lost the race -- the winner already decremented, this call must not repeat it")
}

func TestUpdateSalesOrder_cancellingARestockedOrderReturnsStock(t *testing.T) {
	db := newMockGormDBN(t, 3) // header + link movement + setOrderNeedsStockReview
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
	assert.Equal(t, false, salesOrderRepo.gotTrySwapTo)
	require.NotEmpty(t, salesOrderRepo.updateCalls)
	assert.NotContains(t, salesOrderRepo.updateCalls[0].Fields, "stock_decremented")
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

// TestUpdateSalesOrder_decrementingPropagatesANonInsufficientStockError
// proves a genuinely unexpected recordMovement error (not
// ErrInsufficientStock) still aborts the call with that error -- only
// insufficient stock gets the pending/review treatment.
func TestUpdateSalesOrder_decrementingPropagatesANonInsufficientStockError(t *testing.T) {
	db := newMockGormDBN(t, 1) // header tx only -- the error aborts before any pending/review tx
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(nil)}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: strPtr("sku-1"), Quantity: 1}}}
	recordMovement := &stubRecordMovement{err: errors.New("db unreachable")}
	uc := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, recordMovement)

	status := omsDomain.SalesOrderStatusProcessing
	_, err := uc.Execute(context.Background(), "sales-order-1", appoms.UpdateSalesOrderInput{Status: &status})

	require.Error(t, err)
	assert.NotErrorIs(t, err, invDomain.ErrInsufficientStock)
	assert.Empty(t, lineItemRepo.updateCalls, "not insufficient stock -- must not be marked pending")
}

// TestUpdateSalesOrder_decrementingCountsAnUnlinkedLineItemAsStillPending
// proves an already-Pending, SKUID-less line item (see
// ImportWooOrdersUseCase) keeps the order flagged NeedsStockReview even
// when every linked line item's own decrement succeeds -- it's still
// waiting on LinkLineItemSKUUseCase, not resolved by this call.
func TestUpdateSalesOrder_decrementingCountsAnUnlinkedLineItemAsStillPending(t *testing.T) {
	db := newMockGormDBN(t, 3) // header + link movement (for the linked item) + setOrderNeedsStockReview
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(nil)}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SKUID: strPtr("sku-1"), Quantity: 1},
		{ID: "li-2", SKUID: nil, Quantity: 1, StockStatus: omsDomain.StockStatusPending},
	}}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, recordMovement)

	status := omsDomain.SalesOrderStatusProcessing
	_, err := uc.Execute(context.Background(), "sales-order-1", appoms.UpdateSalesOrderInput{Status: &status})

	require.NoError(t, err)
	require.Len(t, recordMovement.calls, 1, "the unlinked item is never attempted")
	require.NotEmpty(t, salesOrderRepo.updateCalls)
	last := salesOrderRepo.updateCalls[len(salesOrderRepo.updateCalls)-1]
	assert.Equal(t, true, last.Fields["needs_stock_review"], "li-2 is still unresolved")
}

// TestUpdateSalesOrder_restockingSkipsUnlinkedAndResetsPendingLineItems
// proves the restock direction never restocks a line item that was never
// actually decremented (StockStatusPending, or SKUID nil) -- crediting
// stock back for a decrement that never happened would over-add. A
// Pending+linked item is reset straight to Decremented (no longer owed,
// order is cancelled); a SKUID-less item is skipped entirely.
func TestUpdateSalesOrder_restockingSkipsUnlinkedAndResetsPendingLineItems(t *testing.T) {
	db := newMockGormDBN(t, 4) // header + real restock movement+link (2 tx) + setLineItemStockStatus (pending reset) + setOrderNeedsStockReview
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
		o.Status = omsDomain.SalesOrderStatusProcessing
		o.StockDecremented = true
	})}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-decremented", SKUID: strPtr("sku-1"), Quantity: 2, StockStatus: omsDomain.StockStatusDecremented},
		{ID: "li-pending", SKUID: strPtr("sku-2"), Quantity: 1, StockStatus: omsDomain.StockStatusPending},
		{ID: "li-unlinked", SKUID: nil, Quantity: 1, StockStatus: omsDomain.StockStatusPending},
	}}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, recordMovement)

	status := omsDomain.SalesOrderStatusCancelled
	_, err := uc.Execute(context.Background(), "sales-order-1", appoms.UpdateSalesOrderInput{Status: &status})

	require.NoError(t, err)
	require.Len(t, recordMovement.calls, 1, "only the genuinely-decremented item is restocked")
	assert.Equal(t, "sku-1", recordMovement.calls[0].SKUID)
	assert.Equal(t, int64(2), recordMovement.calls[0].QuantityDelta)
	require.Len(t, lineItemRepo.updateCalls, 1, "only li-pending is reset -- li-unlinked is skipped entirely")
	assert.Equal(t, "li-pending", lineItemRepo.updateCalls[0].ID)
	assert.Equal(t, string(omsDomain.StockStatusDecremented), lineItemRepo.updateCalls[0].Fields["stock_status"])
	require.NotEmpty(t, salesOrderRepo.updateCalls)
	last := salesOrderRepo.updateCalls[len(salesOrderRepo.updateCalls)-1]
	assert.Equal(t, false, last.Fields["needs_stock_review"])
}

// TestUpdateSalesOrder_insufficientStockMarksLineItemPendingInsteadOfFailing
// proves the fix for the original bug report (updateSalesOrder:
// recordStockMovement: insufficient stock for this batch/ledger hard-failing
// the whole status change): ErrInsufficientStock on one line item's
// decrement no longer aborts the use case -- it's marked StockStatusPending
// and the order's NeedsStockReview flagged, so status changes/other line
// items on the same order aren't blocked by it. Staff resolve it later via
// RetryStockDecrementUseCase from the "Stock Issues" admin page.
func TestUpdateSalesOrder_insufficientStockMarksLineItemPendingInsteadOfFailing(t *testing.T) {
	db := newMockGormDBN(t, 3) // header + setLineItemStockStatus + setOrderNeedsStockReview
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(nil)}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: strPtr("sku-1"), Quantity: 3}}}
	recordMovement := &stubRecordMovement{err: invDomain.ErrInsufficientStock}
	uc := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, recordMovement)

	status := omsDomain.SalesOrderStatusProcessing
	order, err := uc.Execute(context.Background(), "sales-order-1", appoms.UpdateSalesOrderInput{Status: &status})

	require.NoError(t, err)
	require.NotNil(t, order)
	require.Len(t, lineItemRepo.updateCalls, 1)
	assert.Equal(t, "li-1", lineItemRepo.updateCalls[0].ID)
	assert.Equal(t, string(omsDomain.StockStatusPending), lineItemRepo.updateCalls[0].Fields["stock_status"])
	require.NotEmpty(t, salesOrderRepo.updateCalls)
	lastFields := salesOrderRepo.updateCalls[len(salesOrderRepo.updateCalls)-1].Fields
	assert.Equal(t, true, lastFields["needs_stock_review"])
}
