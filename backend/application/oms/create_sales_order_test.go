package oms_test

import (
	"context"
	"testing"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSalesOrder_completedOrderDecrementsStockPerLine(t *testing.T) {
	// n=3: order creation, linkLineItemMovement, freezeCostAtSale -- one
	// each of the latter two per SKU-linked line item (this test has one).
	db := newMockGormDBN(t, 3)
	salesOrderRepo := &stubSalesOrderRepo{}
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	recordMovement := &stubRecordMovement{}
	outbox := &eventrecordtest.Outbox{}
	uc := appoms.NewCreateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, recordMovement, outbox)
	occurredAt := time.Now()

	order, err := uc.Execute(context.Background(), appoms.CreateSalesOrderInput{
		WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: occurredAt,
		Status: omsDomain.SalesOrderStatusCompleted, TotalMinorUnits: 2000,
		ShippingMethod: "flat_rate", ShippingMethodTitle: "Flat rate",
		LineItems: []appoms.CreateSalesOrderLineItemInput{
			{SKUID: "sku-1", Name: "Widget A", SKUCode: "WID-A", Quantity: 2, PriceMinorUnits: 500, SubtotalMinorUnits: 1000, TotalMinorUnits: 1000},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "sales-order-1", order.SalesOrder.ID)
	assert.Equal(t, omsDomain.SalesOrderSourceManual, order.SalesOrder.Source)
	assert.NotEmpty(t, order.SalesOrder.ExternalOrderID, "manual orders still need a unique external_order_id for the DB constraint")
	assert.Equal(t, "flat_rate", order.SalesOrder.ShippingMethod)
	assert.Equal(t, "Flat rate", order.SalesOrder.ShippingMethodTitle)
	assert.True(t, order.SalesOrder.Fulfilled, "created with an explicit Completed status -- starts fulfilled")
	require.Len(t, salesOrderRepo.created, 1)
	require.Len(t, lineItemRepo.created, 1)
	require.Len(t, recordMovement.calls, 1)
	assert.Equal(t, "sku-1", recordMovement.calls[0].SKUID)
	assert.Equal(t, invDomain.MovementTypeSale, recordMovement.calls[0].Type)
	assert.Equal(t, int64(2), recordMovement.calls[0].Quantity)

	assert.Contains(t, outbox.Subjects(), "oms.sales_order.created")

}

func TestCreateSalesOrder_snapshotsTheSKUsTaxClassOntoTheLineItem(t *testing.T) {
	db := newMockGormDBN(t, 1)
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	taxClassID := "tax-class-standard"
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{
		"sku-1": {ID: "sku-1", SKUCode: "WID-A", TaxClassID: &taxClassID},
	}}
	uc := appoms.NewCreateSalesOrderUseCase(db, &stubSalesOrderRepo{}, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, skuRepo, &stubRecordMovement{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.CreateSalesOrderInput{
		WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		Status:    omsDomain.SalesOrderStatusPending,
		LineItems: []appoms.CreateSalesOrderLineItemInput{{SKUID: "sku-1", Quantity: 1, PriceMinorUnits: 500}},
	})

	require.NoError(t, err)
	require.Len(t, lineItemRepo.created, 1)
	require.NotNil(t, lineItemRepo.created[0].TaxClassID)
	assert.Equal(t, "tax-class-standard", *lineItemRepo.created[0].TaxClassID)
}

func TestCreateSalesOrder_missingSKUDoesNotBlockOrderCreation(t *testing.T) {
	db := newMockGormDBN(t, 1)
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	skuRepo := &stubSKURepo{findErr: pmsDomain.ErrSKUNotFound}
	uc := appoms.NewCreateSalesOrderUseCase(db, &stubSalesOrderRepo{}, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, skuRepo, &stubRecordMovement{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.CreateSalesOrderInput{
		WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		Status:    omsDomain.SalesOrderStatusPending,
		LineItems: []appoms.CreateSalesOrderLineItemInput{{SKUID: "unknown-sku", Quantity: 1, PriceMinorUnits: 500}},
	})

	require.NoError(t, err, "unlike POS/Woo, manual order creation has never validated SKUID -- a lookup failure must not become a new way to fail")
	require.Len(t, lineItemRepo.created, 1)
	assert.Nil(t, lineItemRepo.created[0].TaxClassID)
}

func TestCreateSalesOrder_pendingOrderDoesNotDecrementStock(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{}
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewCreateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, recordMovement, &eventrecordtest.Outbox{})

	order, err := uc.Execute(context.Background(), appoms.CreateSalesOrderInput{
		WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		Status: omsDomain.SalesOrderStatusPending,
		LineItems: []appoms.CreateSalesOrderLineItemInput{
			{SKUID: "sku-1", Quantity: 2, PriceMinorUnits: 500},
		},
	})

	require.NoError(t, err)
	assert.False(t, order.SalesOrder.StockDecremented)
	assert.Empty(t, recordMovement.calls, "pending order has not been paid for yet -- no stock impact")
}

func TestCreateSalesOrder_persistsSalesChannelIDWhenGiven(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{}
	uc := appoms.NewCreateSalesOrderUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, &stubRecordMovement{}, &eventrecordtest.Outbox{})
	channelID := "channel-1"

	order, err := uc.Execute(context.Background(), appoms.CreateSalesOrderInput{
		WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(), SalesChannelID: &channelID,
		LineItems: []appoms.CreateSalesOrderLineItemInput{{SKUID: "sku-1", Quantity: 1, PriceMinorUnits: 500}},
	})

	require.NoError(t, err)
	require.NotNil(t, order.SalesOrder.SalesChannelID)
	assert.Equal(t, channelID, *order.SalesOrder.SalesChannelID)
}

func TestCreateSalesOrder_defaultsStatusToPending(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{}
	uc := appoms.NewCreateSalesOrderUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, &stubRecordMovement{}, &eventrecordtest.Outbox{})

	order, err := uc.Execute(context.Background(), appoms.CreateSalesOrderInput{
		WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.CreateSalesOrderLineItemInput{{SKUID: "sku-1", Quantity: 1, PriceMinorUnits: 500}},
	})

	require.NoError(t, err)
	assert.Equal(t, omsDomain.SalesOrderStatusPending, order.SalesOrder.Status, "unlike a POS sale, a manually-entered order has not necessarily happened/been paid for yet")
	assert.False(t, order.SalesOrder.Fulfilled)
}

func TestCreateSalesOrder_rejectsEmptyLineItems(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewCreateSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, &stubRecordMovement{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.CreateSalesOrderInput{WarehouseID: "warehouse-1"})

	assert.ErrorIs(t, err, omsDomain.ErrEmptyLineItems)
}

// Short stock must not fail an order that already exists: the short line
// is left pending and the order goes to Stock issues, same as a status
// change into processing does (decrementLineItems).
func TestCreateSalesOrder_shortStockSendsTheOrderToStockIssues(t *testing.T) {
	// order creation, then for the covered line link + freeze cost, then
	// the short line's status and the order's review flag.
	db := newMockGormDBN(t, 5)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "sales-order-1"}}
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	recordMovement := &stubRecordMovement{errBySKU: map[string]error{"sku-short": invDomain.ErrInsufficientStock}}
	outbox := &eventrecordtest.Outbox{}
	uc := appoms.NewCreateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, recordMovement, outbox)

	result, err := uc.Execute(context.Background(), appoms.CreateSalesOrderInput{
		WarehouseID: "warehouse-1", CurrencyCode: "EUR", OccurredAt: time.Now(),
		Status: omsDomain.SalesOrderStatusProcessing,
		LineItems: []appoms.CreateSalesOrderLineItemInput{
			{SKUID: "sku-ok", Quantity: 1, PriceMinorUnits: 500},
			{SKUID: "sku-short", Quantity: 100, PriceMinorUnits: 500},
		},
	})

	require.NoError(t, err)
	assert.True(t, result.SalesOrder.NeedsStockReview)
	require.Len(t, result.LineItems, 2)
	assert.NotEqual(t, omsDomain.StockStatusPending, result.LineItems[0].StockStatus)
	assert.Equal(t, omsDomain.StockStatusPending, result.LineItems[1].StockStatus)
	assert.Equal(t, map[string]any{"needs_stock_review": true}, salesOrderRepo.gotUpdateFields)
}

func TestCreateSalesOrder_propagatesAStockMovementError(t *testing.T) {
	db := newMockGormDB(t)
	recordMovement := &stubRecordMovement{err: assert.AnError}
	uc := appoms.NewCreateSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, recordMovement, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.CreateSalesOrderInput{
		WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		Status:    omsDomain.SalesOrderStatusCompleted,
		LineItems: []appoms.CreateSalesOrderLineItemInput{{SKUID: "sku-1", Quantity: 100, PriceMinorUnits: 500}},
	})

	assert.ErrorIs(t, err, assert.AnError)
}

func TestCreateSalesOrder_propagatesACreateError(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{createErr: assert.AnError}
	uc := appoms.NewCreateSalesOrderUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSKURepo{}, &stubRecordMovement{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.CreateSalesOrderInput{
		WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.CreateSalesOrderLineItemInput{{SKUID: "sku-1", Quantity: 1, PriceMinorUnits: 500}},
	})

	require.Error(t, err)
}
