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

func TestReceivePOSSalesOrder_createsTheOrderAndDecrementsStockPerLine(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{}
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, recordMovement)
	occurredAt := time.Now()

	order, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: occurredAt,
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{
			{SKUID: "sku-1", Quantity: 2, UnitPriceMinorUnits: 500},
			{SKUID: "sku-2", Quantity: 1, UnitPriceMinorUnits: 1000},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "sales-order-1", order.ID)
	assert.Equal(t, int64(2000), order.TotalMinorUnits, "2*500 + 1*1000")
	require.Len(t, salesOrderRepo.created, 1)
	require.Len(t, lineItemRepo.created, 2)
	require.Len(t, recordMovement.calls, 2)
	assert.Equal(t, "sku-1", recordMovement.calls[0].SKUID)
	assert.Equal(t, invDomain.MovementTypeSale, recordMovement.calls[0].Type)
	assert.Equal(t, int64(-2), recordMovement.calls[0].QuantityDelta, "negative -- stock going out")
	assert.Equal(t, int64(-1), recordMovement.calls[1].QuantityDelta)
}

func TestReceivePOSSalesOrder_isIdempotentOnExternalOrderID(t *testing.T) {
	db := newMockGormDB(t)
	existing := &omsDomain.SalesOrder{ID: "sales-order-existing", ExternalOrderID: "pos-order-1"}
	salesOrderRepo := &stubSalesOrderRepo{findByExternalIDResult: existing}
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, recordMovement)

	order, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{SKUID: "sku-1", Quantity: 2, UnitPriceMinorUnits: 500}},
	})

	require.NoError(t, err)
	assert.Equal(t, "sales-order-existing", order.ID)
	assert.Empty(t, salesOrderRepo.created, "retry must not create a second order")
	assert.Empty(t, lineItemRepo.created, "retry must not create duplicate line items")
	assert.Empty(t, recordMovement.calls, "retry must not double-decrement stock")
}

func TestReceivePOSSalesOrder_rejectsEmptyLineItems(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubRecordMovement{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{ExternalOrderID: "pos-order-1"})

	assert.ErrorIs(t, err, omsDomain.ErrEmptyLineItems)
}

func TestReceivePOSSalesOrder_propagatesAStockMovementError(t *testing.T) {
	db := newMockGormDB(t)
	recordMovement := &stubRecordMovement{err: invDomain.ErrInsufficientStock}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, recordMovement)

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{SKUID: "sku-1", Quantity: 100, UnitPriceMinorUnits: 500}},
	})

	assert.ErrorIs(t, err, invDomain.ErrInsufficientStock)
}

func TestReceivePOSSalesOrder_propagatesACreateError(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{createErr: assert.AnError}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubRecordMovement{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{SKUID: "sku-1", Quantity: 1, UnitPriceMinorUnits: 500}},
	})

	require.Error(t, err)
}
