package oms_test

import (
	"context"
	"testing"
	"time"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReceivePOSSalesOrder_createsTheOrderAndDecrementsStockPerLine(t *testing.T) {
	db := newMockGormDBN(t, 3)
	salesOrderRepo := &stubSalesOrderRepo{}
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, recordMovement)
	occurredAt := time.Now()

	order, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: occurredAt,
		TotalMinorUnits: 2000,
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{
			{SKUID: "sku-1", Name: "Widget A", SKUCode: "WID-A", Quantity: 2, PriceMinorUnits: 500, SubtotalMinorUnits: 1000, TotalMinorUnits: 1000},
			{SKUID: "sku-2", Name: "Widget B", SKUCode: "WID-B", Quantity: 1, PriceMinorUnits: 1000, SubtotalMinorUnits: 1000, TotalMinorUnits: 1000},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "sales-order-1", order.ID)
	assert.Equal(t, int64(2000), order.TotalMinorUnits)
	assert.Equal(t, omsDomain.SalesOrderStatusCompleted, order.Status, "defaults to completed -- a POS sale already happened")
	require.Len(t, salesOrderRepo.created, 1)
	require.Len(t, lineItemRepo.created, 2)
	assert.Equal(t, "Widget A", lineItemRepo.created[0].Name)
	assert.Equal(t, "SKU-sku-1", lineItemRepo.created[0].SKUCode, "sku_code/barcode come from the matched SKU record, not the POS payload")
	assert.Equal(t, "BARCODE-sku-1", lineItemRepo.created[0].Barcode)
	assert.Equal(t, int64(500), lineItemRepo.created[0].PriceMinorUnits)
	require.Len(t, recordMovement.calls, 2)
	assert.Equal(t, "sku-1", recordMovement.calls[0].SKUID)
	assert.Equal(t, invDomain.MovementTypeSale, recordMovement.calls[0].Type)
	assert.Equal(t, int64(-2), recordMovement.calls[0].QuantityDelta, "negative -- stock going out")
	assert.Equal(t, int64(-1), recordMovement.calls[1].QuantityDelta)
	assert.True(t, recordMovement.calls[0].ReconcileShortfall, "a POS sale already physically happened -- insufficient tracked stock must not block it")
}

func TestReceivePOSSalesOrder_matchesLineItemsToSKUsByID(t *testing.T) {
	db := newMockGormDBN(t, 2)
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{
		"sku-1": {ID: "sku-1", SKUCode: "5601049213991", Barcode: "5601049213991"},
	}}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, skuRepo, &stubRecordMovement{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{
			// POS omitted sku_code entirely -- the exact bug reported: line item
			// synced with a blank sku_code even though sku_id correctly matches
			// a real SKU.
			{SKUID: "sku-1", Quantity: 1, PriceMinorUnits: 182},
		},
	})

	require.NoError(t, err)
	require.Len(t, lineItemRepo.created, 1)
	assert.Equal(t, "5601049213991", lineItemRepo.created[0].SKUCode)
	assert.Equal(t, "5601049213991", lineItemRepo.created[0].Barcode)
}

func TestReceivePOSSalesOrder_propagatesASKUNotFoundError(t *testing.T) {
	db := newMockGormDB(t)
	skuRepo := &stubSKURepo{findErr: pmsDomain.ErrSKUNotFound}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, skuRepo, &stubRecordMovement{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{SKUID: "missing-sku", Quantity: 1, PriceMinorUnits: 500}},
	})

	assert.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}

func TestReceivePOSSalesOrder_isIdempotentOnExternalOrderID(t *testing.T) {
	db := newMockGormDB(t)
	existing := &omsDomain.SalesOrder{ID: "sales-order-existing", ExternalOrderID: "pos-order-1"}
	salesOrderRepo := &stubSalesOrderRepo{findByExternalIDResult: existing}
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, recordMovement)

	order, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{SKUID: "sku-1", Quantity: 2, PriceMinorUnits: 500}},
	})

	require.NoError(t, err)
	assert.Equal(t, "sales-order-existing", order.ID)
	assert.Empty(t, salesOrderRepo.created, "retry must not create a second order")
	assert.Empty(t, lineItemRepo.created, "retry must not create duplicate line items")
	assert.Empty(t, recordMovement.calls, "retry must not double-decrement stock")
}

func TestReceivePOSSalesOrder_rejectsEmptyLineItems(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, &stubRecordMovement{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{ExternalOrderID: "pos-order-1"})

	assert.ErrorIs(t, err, omsDomain.ErrEmptyLineItems)
}

func TestReceivePOSSalesOrder_propagatesAStockMovementError(t *testing.T) {
	db := newMockGormDB(t)
	recordMovement := &stubRecordMovement{err: invDomain.ErrInsufficientStock}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, recordMovement)

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{SKUID: "sku-1", Quantity: 100, PriceMinorUnits: 500}},
	})

	assert.ErrorIs(t, err, invDomain.ErrInsufficientStock)
}

func TestReceivePOSSalesOrder_propagatesACreateError(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{createErr: assert.AnError}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, &stubRecordMovement{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{SKUID: "sku-1", Quantity: 1, PriceMinorUnits: 500}},
	})

	require.Error(t, err)
}

func TestReceivePOSSalesOrder_preservesExplicitStatus(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, &stubRecordMovement{})

	order, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		Status:    omsDomain.SalesOrderStatusRefunded,
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{SKUID: "sku-1", Quantity: 1, PriceMinorUnits: 500}},
	})

	require.NoError(t, err)
	assert.Equal(t, omsDomain.SalesOrderStatusRefunded, order.Status)
}
