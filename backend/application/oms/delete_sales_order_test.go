package oms_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/domain/integrations"
	"testing"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
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
	outbox := &eventrecordtest.Outbox{}
	uc := appoms.NewDeleteSalesOrderUseCase(db, salesOrderRepo, &stubOrigins{}, lineItemRepo, linkRepo, reverseMovement, outbox)

	err := uc.Execute(context.Background(), "sales-order-1")

	require.NoError(t, err)
	require.Len(t, reverseMovement.calls, 1)
	assert.Equal(t, int64(3), reverseMovement.calls[0].QuantityDelta, "positive -- stock coming back")
	assert.Equal(t, "sales-order-1", salesOrderRepo.gotDeleteID)

	assert.Contains(t, outbox.Subjects(), "oms.sales_order.deleted")
	assert.Equal(t, true, outbox.Events[0].Data()["stock_restored"])

}

func TestDeleteSalesOrder_doesNotRestockWhenStockWasNeverDecremented(t *testing.T) {
	db := newMockGormDBN(t, 2)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(nil)}
	reverseMovement := &stubReverseMovement{}
	uc := appoms.NewDeleteSalesOrderUseCase(db, salesOrderRepo, &stubOrigins{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, reverseMovement, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "sales-order-1")

	require.NoError(t, err)
	assert.Empty(t, reverseMovement.calls)
	assert.Equal(t, "sales-order-1", salesOrderRepo.gotDeleteID)
}

func TestDeleteSalesOrder_returnsNotFoundForAMissingOrder(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewDeleteSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubOrigins{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubReverseMovement{}, &eventrecordtest.Outbox{})

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
	uc := appoms.NewDeleteSalesOrderUseCase(db, salesOrderRepo, &stubOrigins{}, lineItemRepo, linkRepo, reverseMovement, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "sales-order-1")

	assert.ErrorIs(t, err, assert.AnError)
	assert.Empty(t, salesOrderRepo.gotDeleteID, "must not delete if the restock failed")
}

// COM-15: the deleted event still says where the order came from -- read
// from its origin before the row goes with the order.
func TestDeleteSalesOrder_theEventCarriesTheOrdersSource(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appoms.NewDeleteSalesOrderUseCase(newMockGormDBN(t, 2), &stubSalesOrderRepo{findByIDResult: existingOrder(nil)},
		&stubOrigins{source: integrations.OriginBol}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubReverseMovement{}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "sales-order-1"))

	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "bol", e.Data()["source"])
}

func TestDeleteSalesOrder_anOriginReadFailureKeepsTheOrder(t *testing.T) {
	repo := &stubSalesOrderRepo{findByIDResult: existingOrder(nil)}
	uc := appoms.NewDeleteSalesOrderUseCase(newMockGormDBN(t, 2), repo, &stubOrigins{ofErr: errors.New("db down")},
		&stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubReverseMovement{}, &eventrecordtest.Outbox{})

	require.Error(t, uc.Execute(context.Background(), "sales-order-1"))
	assert.Empty(t, repo.gotDeleteID)
}
