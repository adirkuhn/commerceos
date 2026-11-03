package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSalesOrder_returnsOrderAndLineItems(t *testing.T) {
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "so-1"}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{{ID: "li-1", SKUID: strPtr("sku-1")}}}
	uc := appoms.NewGetSalesOrderUseCase(newMockGormDB(t), salesOrderRepo, lineItemRepo, &stubSalesOrderPaymentRepo{})

	result, err := uc.Execute(context.Background(), "so-1")

	require.NoError(t, err)
	assert.Equal(t, "so-1", result.SalesOrder.ID)
	require.Len(t, result.LineItems, 1)
	require.NotNil(t, result.LineItems[0].SKUID)
	assert.Equal(t, "sku-1", *result.LineItems[0].SKUID)
}

func TestGetSalesOrder_propagatesNotFound(t *testing.T) {
	uc := appoms.NewGetSalesOrderUseCase(newMockGormDB(t), &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderPaymentRepo{})

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}

func TestGetSalesOrder_returnsTheOrdersPayments(t *testing.T) {
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "so-1"}}
	payments := omsDomain.SalesOrderPayments{{Method: "Cash", AmountMinorUnits: 500}, {Method: "Revolut", AmountMinorUnits: 1500, TransactionID: "rev-1"}}
	uc := appoms.NewGetSalesOrderUseCase(newMockGormDB(t), salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderPaymentRepo{byOrder: payments})

	result, err := uc.Execute(context.Background(), "so-1")

	require.NoError(t, err)
	assert.Equal(t, payments, result.SalesOrder.Payments)
}

func TestGetSalesOrder_propagatesAPaymentLoadError(t *testing.T) {
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "so-1"}}
	uc := appoms.NewGetSalesOrderUseCase(newMockGormDB(t), salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderPaymentRepo{listErr: assert.AnError})

	_, err := uc.Execute(context.Background(), "so-1")

	assert.ErrorIs(t, err, assert.AnError)
}
