package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeleteSalesOrder_restocksBeforeDeletingWhenStockWasDecremented proves
// deleting a decremented order reverses its line items' ORIGINAL decrement
// movement(s) (via the link table's denormalized BatchID/QuantityDelta),
// not a fresh general-ledger credit -- see restockLineItems's doc for why.
func TestDeleteSalesOrder_restocksBeforeDeletingWhenStockWasDecremented(t *testing.T) {
	db := newMockGormDBN(t, 3) // header + listLineItemMovements + delete tx
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
		o.StockDecremented = true
	})}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SKUID: strPtr("sku-1"), Quantity: 3, StockStatus: omsDomain.StockStatusDecremented},
	}}
	linkRepo := &stubSalesOrderLineItemStockMovementRepo{byLineItem: map[string][]omsDomain.SalesOrderLineItemStockMovement{
		"li-1": {{SalesOrderLineItemID: "li-1", QuantityDelta: -3}},
	}}
	reverseMovement := &stubReverseMovement{}
	uc := appoms.NewDeleteSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, linkRepo, reverseMovement)

	err := uc.Execute(context.Background(), "sales-order-1")

	require.NoError(t, err)
	require.Len(t, reverseMovement.calls, 1)
	assert.Equal(t, int64(3), reverseMovement.calls[0].QuantityDelta, "positive -- stock coming back")
	assert.Equal(t, "sales-order-1", salesOrderRepo.gotDeleteID)
}

func TestDeleteSalesOrder_doesNotRestockWhenStockWasNeverDecremented(t *testing.T) {
	db := newMockGormDBN(t, 2)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(nil)}
	reverseMovement := &stubReverseMovement{}
	uc := appoms.NewDeleteSalesOrderUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, reverseMovement)

	err := uc.Execute(context.Background(), "sales-order-1")

	require.NoError(t, err)
	assert.Empty(t, reverseMovement.calls)
	assert.Equal(t, "sales-order-1", salesOrderRepo.gotDeleteID)
}

func TestDeleteSalesOrder_returnsNotFoundForAMissingOrder(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewDeleteSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubReverseMovement{})

	err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}

func TestDeleteSalesOrder_propagatesAStockMovementError(t *testing.T) {
	db := newMockGormDBN(t, 2) // header + listLineItemMovements -- the error aborts before the delete tx
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
		o.StockDecremented = true
	})}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SKUID: strPtr("sku-1"), Quantity: 3, StockStatus: omsDomain.StockStatusDecremented},
	}}
	linkRepo := &stubSalesOrderLineItemStockMovementRepo{byLineItem: map[string][]omsDomain.SalesOrderLineItemStockMovement{
		"li-1": {{SalesOrderLineItemID: "li-1", QuantityDelta: -3}},
	}}
	reverseMovement := &stubReverseMovement{err: assert.AnError}
	uc := appoms.NewDeleteSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, linkRepo, reverseMovement)

	err := uc.Execute(context.Background(), "sales-order-1")

	assert.ErrorIs(t, err, assert.AnError)
	assert.Empty(t, salesOrderRepo.gotDeleteID, "must not delete if the restock failed")
}
