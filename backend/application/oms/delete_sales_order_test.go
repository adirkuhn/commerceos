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

func TestDeleteSalesOrder_restocksBeforeDeletingWhenStockWasDecremented(t *testing.T) {
	db := newMockGormDBN(t, 2)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
		o.StockDecremented = true
	})}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{SKUID: strPtr("sku-1"), Quantity: 3}}}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewDeleteSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, recordMovement)

	err := uc.Execute(context.Background(), "sales-order-1")

	require.NoError(t, err)
	require.Len(t, recordMovement.calls, 1)
	assert.Equal(t, int64(3), recordMovement.calls[0].QuantityDelta, "positive -- stock coming back")
	assert.Equal(t, "sales-order-1", salesOrderRepo.gotDeleteID)
}

func TestDeleteSalesOrder_doesNotRestockWhenStockWasNeverDecremented(t *testing.T) {
	db := newMockGormDBN(t, 2)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(nil)}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewDeleteSalesOrderUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, recordMovement)

	err := uc.Execute(context.Background(), "sales-order-1")

	require.NoError(t, err)
	assert.Empty(t, recordMovement.calls)
	assert.Equal(t, "sales-order-1", salesOrderRepo.gotDeleteID)
}

func TestDeleteSalesOrder_returnsNotFoundForAMissingOrder(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewDeleteSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubRecordMovement{})

	err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}

func TestDeleteSalesOrder_propagatesAStockMovementError(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: existingOrder(func(o *omsDomain.SalesOrder) {
		o.StockDecremented = true
	})}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{SKUID: strPtr("sku-1"), Quantity: 3}}}
	recordMovement := &stubRecordMovement{err: invDomain.ErrInsufficientStock}
	uc := appoms.NewDeleteSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, recordMovement)

	err := uc.Execute(context.Background(), "sales-order-1")

	assert.ErrorIs(t, err, invDomain.ErrInsufficientStock)
	assert.Empty(t, salesOrderRepo.gotDeleteID, "must not delete if the restock failed")
}
