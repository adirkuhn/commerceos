package oms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubOrderNumbers struct {
	next  string
	err   error
	calls int
}

func (s *stubOrderNumbers) Next(context.Context) (string, error) {
	s.calls++
	return s.next, s.err
}

func numberedOrderInput(number string) appoms.CreateSalesOrderInput {
	return appoms.CreateSalesOrderInput{
		Number: number, WarehouseID: "warehouse-1", CurrencyCode: "EUR", OccurredAt: time.Now(),
		LineItems: []appoms.CreateSalesOrderLineItemInput{{SKUID: "sku-1", Quantity: 1, PriceMinorUnits: 500}},
	}
}

func TestCreateSalesOrder_numbersAnOrderSentWithoutOne(t *testing.T) {
	salesOrderRepo := &stubSalesOrderRepo{}
	numbers := &stubOrderNumbers{next: "CO-1001"}
	uc := appoms.NewCreateSalesOrderUseCase(newMockGormDBN(t, 1), salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, &stubRecordMovement{}, &eventrecordtest.Outbox{}).
		WithNumbers(numbers)

	order, err := uc.Execute(context.Background(), numberedOrderInput(""))

	require.NoError(t, err)
	assert.Equal(t, "CO-1001", order.SalesOrder.Number)
	require.Len(t, salesOrderRepo.created, 1)
	assert.Equal(t, "CO-1001", salesOrderRepo.created[0].Number)
}

func TestCreateSalesOrder_keepsATypedNumber(t *testing.T) {
	salesOrderRepo := &stubSalesOrderRepo{}
	numbers := &stubOrderNumbers{next: "CO-1001"}
	uc := appoms.NewCreateSalesOrderUseCase(newMockGormDBN(t, 1), salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, &stubRecordMovement{}, &eventrecordtest.Outbox{}).
		WithNumbers(numbers)

	order, err := uc.Execute(context.Background(), numberedOrderInput("2026-17"))

	require.NoError(t, err)
	assert.Equal(t, "2026-17", order.SalesOrder.Number)
	assert.Zero(t, numbers.calls)
}

func TestCreateSalesOrder_failsWhenNoNumberCanBeHandedOut(t *testing.T) {
	uc := appoms.NewCreateSalesOrderUseCase(newMockGormDBN(t, 1), &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, &stubRecordMovement{}, &eventrecordtest.Outbox{}).
		WithNumbers(&stubOrderNumbers{err: errors.New("counter down")})

	_, err := uc.Execute(context.Background(), numberedOrderInput(""))

	assert.ErrorContains(t, err, "counter down")
}
