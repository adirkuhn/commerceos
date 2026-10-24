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

func TestReceivePOSSalesOrder_createsTheOrderAndDecrementsStockPerLine(t *testing.T) {
	db := newMockGormDBN(t, 5) // order create, 2x(link+freezeCostAtSale)
	salesOrderRepo := &stubSalesOrderRepo{}
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	recordMovement := &stubRecordMovement{}
	outbox := &eventrecordtest.Outbox{}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, &stubSalesOrderPaymentRepo{}, &stubSKURepo{}, recordMovement, outbox)
	occurredAt := time.Now()

	order, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: occurredAt,
		TotalMinorUnits: 2000,
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{
			{SKUID: strPtr("sku-1"), Name: "Widget A", SKUCode: "WID-A", Quantity: 2, PriceMinorUnits: 500, SubtotalMinorUnits: 1000, TotalMinorUnits: 1000},
			{SKUID: strPtr("sku-2"), Name: "Widget B", SKUCode: "WID-B", Quantity: 1, PriceMinorUnits: 1000, SubtotalMinorUnits: 1000, TotalMinorUnits: 1000},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "sales-order-1", order.ID)
	assert.Equal(t, int64(2000), order.TotalMinorUnits)
	assert.Equal(t, omsDomain.SalesOrderStatusCompleted, order.Status, "defaults to completed -- a POS sale already happened")
	assert.True(t, order.Fulfilled, "a completed order starts fulfilled too")
	require.Len(t, salesOrderRepo.created, 1)
	require.Len(t, lineItemRepo.created, 2)
	assert.Equal(t, "Widget A", lineItemRepo.created[0].Name)
	assert.Equal(t, "SKU-sku-1", lineItemRepo.created[0].SKUCode, "sku_code/barcode come from the matched SKU record, not the POS payload")
	assert.Equal(t, "BARCODE-sku-1", lineItemRepo.created[0].Barcode)
	assert.Equal(t, int64(500), lineItemRepo.created[0].PriceMinorUnits)
	require.Len(t, recordMovement.calls, 2)
	assert.Equal(t, "sku-1", recordMovement.calls[0].SKUID)
	assert.Equal(t, invDomain.MovementTypeSale, recordMovement.calls[0].Type)
	assert.Equal(t, int64(2), recordMovement.calls[0].Quantity)
	assert.Equal(t, int64(1), recordMovement.calls[1].Quantity)
	assert.True(t, recordMovement.calls[0].ReconcileShortfall, "a POS sale already physically happened -- insufficient tracked stock must not block it")

	assert.Contains(t, outbox.Subjects(), "oms.sales_order.created")
	assert.Equal(t, "pos", outbox.Events[0].Data()["source"])

}

func TestReceivePOSSalesOrder_matchesLineItemsToSKUsByID(t *testing.T) {
	db := newMockGormDBN(t, 3) // order create, link, freezeCostAtSale
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	taxClassID := "tax-class-standard"
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{
		"sku-1": {ID: "sku-1", SKUCode: "5601049213991", Barcode: "5601049213991", TaxClassID: &taxClassID},
	}}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, &stubSalesOrderPaymentRepo{}, skuRepo, &stubRecordMovement{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{
			// POS omitted sku_code entirely -- the exact bug reported: line item
			// synced with a blank sku_code even though sku_id correctly matches
			// a real SKU.
			{SKUID: strPtr("sku-1"), Quantity: 1, PriceMinorUnits: 182},
		},
	})

	require.NoError(t, err)
	require.Len(t, lineItemRepo.created, 1)
	assert.Equal(t, "5601049213991", lineItemRepo.created[0].SKUCode)
	assert.Equal(t, "5601049213991", lineItemRepo.created[0].Barcode)
	require.NotNil(t, lineItemRepo.created[0].TaxClassID)
	assert.Equal(t, "tax-class-standard", *lineItemRepo.created[0].TaxClassID)
	assert.True(t, lineItemRepo.created[0].TotalIncludesTax, "POS sends line totals with VAT")
}

func TestReceivePOSSalesOrder_propagatesASKUNotFoundError(t *testing.T) {
	db := newMockGormDB(t)
	skuRepo := &stubSKURepo{findErr: pmsDomain.ErrSKUNotFound}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSalesOrderPaymentRepo{}, skuRepo, &stubRecordMovement{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{SKUID: strPtr("missing-sku"), Quantity: 1, PriceMinorUnits: 500}},
	})

	assert.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}

// TestReceivePOSSalesOrder_acceptsAnAdhocLineItemWithNoSKU proves a POS
// "open item"/custom-amount line (no sku_id) is saved as part of the
// order, not dropped or rejected -- the exact bug reported: an adhoc item
// used to abort the whole order's sync (skuRepo.FindByID("") ->
// ErrSKUNotFound -> transaction rollback).
func TestReceivePOSSalesOrder_acceptsAnAdhocLineItemWithNoSKU(t *testing.T) {
	db := newMockGormDBN(t, 2)
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, &stubSalesOrderPaymentRepo{}, &stubSKURepo{}, &stubRecordMovement{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		TotalMinorUnits: 500,
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{
			{Name: "Gift wrapping", Quantity: 1, PriceMinorUnits: 500, SubtotalMinorUnits: 500, TotalMinorUnits: 500},
		},
	})

	require.NoError(t, err)
	require.Len(t, lineItemRepo.created, 1)
	assert.Nil(t, lineItemRepo.created[0].SKUID)
	assert.Equal(t, "Gift wrapping", lineItemRepo.created[0].Name)
	assert.Equal(t, int64(500), lineItemRepo.created[0].TotalMinorUnits)
}

// TestReceivePOSSalesOrder_adhocLineItemDoesNotRecordAStockMovement proves
// there's nothing to decrement stock for on a line with no SKU -- the
// nil-SKUID guard in the stock-movement loop, exercised for real now that
// an adhoc line can actually reach it.
func TestReceivePOSSalesOrder_adhocLineItemDoesNotRecordAStockMovement(t *testing.T) {
	db := newMockGormDBN(t, 1)
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSalesOrderPaymentRepo{}, &stubSKURepo{}, recordMovement, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{Name: "Gift wrapping", Quantity: 1, PriceMinorUnits: 500}},
	})

	require.NoError(t, err)
	assert.Empty(t, recordMovement.calls, "no SKU on the line -- nothing to decrement stock for")
}

// TestReceivePOSSalesOrder_rejectsAdhocLineItemWithNoName proves Name is
// required when there's no SKU -- otherwise the line would carry no
// identifying data at all.
func TestReceivePOSSalesOrder_rejectsAdhocLineItemWithNoName(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSalesOrderPaymentRepo{}, &stubSKURepo{}, &stubRecordMovement{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{Quantity: 1, PriceMinorUnits: 500}},
	})

	assert.ErrorIs(t, err, omsDomain.ErrAdhocLineItemRequiresName)
}

// TestReceivePOSSalesOrder_mixesAdhocAndCatalogLineItemsInOneOrder proves
// a single POS ticket can carry both kinds of line at once.
func TestReceivePOSSalesOrder_mixesAdhocAndCatalogLineItemsInOneOrder(t *testing.T) {
	db := newMockGormDBN(t, 3) // order create, link, freezeCostAtSale
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, &stubSalesOrderPaymentRepo{}, &stubSKURepo{}, recordMovement, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{
			{SKUID: strPtr("sku-1"), Quantity: 1, PriceMinorUnits: 500},
			{Name: "Gift wrapping", Quantity: 1, PriceMinorUnits: 200},
		},
	})

	require.NoError(t, err)
	require.Len(t, lineItemRepo.created, 2)
	assert.NotNil(t, lineItemRepo.created[0].SKUID)
	assert.Nil(t, lineItemRepo.created[1].SKUID)
	require.Len(t, recordMovement.calls, 1, "only the catalog line moves stock")
}

func TestReceivePOSSalesOrder_isIdempotentOnExternalOrderID(t *testing.T) {
	db := newMockGormDB(t)
	existing := &omsDomain.SalesOrder{ID: "sales-order-existing", ExternalOrderID: "pos-order-1"}
	salesOrderRepo := &stubSalesOrderRepo{findByExternalIDResult: existing}
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	recordMovement := &stubRecordMovement{}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, &stubSalesOrderLineItemStockMovementRepo{}, &stubSalesOrderPaymentRepo{}, &stubSKURepo{}, recordMovement, &eventrecordtest.Outbox{})

	order, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{SKUID: strPtr("sku-1"), Quantity: 2, PriceMinorUnits: 500}},
	})

	require.NoError(t, err)
	assert.Equal(t, "sales-order-existing", order.ID)
	assert.Empty(t, salesOrderRepo.created, "retry must not create a second order")
	assert.Empty(t, lineItemRepo.created, "retry must not create duplicate line items")
	assert.Empty(t, recordMovement.calls, "retry must not double-decrement stock")
}

func TestReceivePOSSalesOrder_rejectsEmptyLineItems(t *testing.T) {
	db := newMockGormDB(t)
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSalesOrderPaymentRepo{}, &stubSKURepo{}, &stubRecordMovement{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{ExternalOrderID: "pos-order-1"})

	assert.ErrorIs(t, err, omsDomain.ErrEmptyLineItems)
}

func TestReceivePOSSalesOrder_propagatesAStockMovementError(t *testing.T) {
	db := newMockGormDB(t)
	recordMovement := &stubRecordMovement{err: invDomain.ErrInsufficientStock}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSalesOrderPaymentRepo{}, &stubSKURepo{}, recordMovement, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{SKUID: strPtr("sku-1"), Quantity: 100, PriceMinorUnits: 500}},
	})

	assert.ErrorIs(t, err, invDomain.ErrInsufficientStock)
}

func TestReceivePOSSalesOrder_propagatesACreateError(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{createErr: assert.AnError}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSalesOrderPaymentRepo{}, &stubSKURepo{}, &stubRecordMovement{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{SKUID: strPtr("sku-1"), Quantity: 1, PriceMinorUnits: 500}},
	})

	require.Error(t, err)
}

func TestReceivePOSSalesOrder_preservesExplicitStatus(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{}
	uc := appoms.NewReceivePOSSalesOrderUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSalesOrderLineItemStockMovementRepo{}, &stubSalesOrderPaymentRepo{}, &stubSKURepo{}, &stubRecordMovement{}, &eventrecordtest.Outbox{})

	order, err := uc.Execute(context.Background(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: "pos-order-1", WarehouseID: "warehouse-1", CurrencyCode: "USD", OccurredAt: time.Now(),
		Status:    omsDomain.SalesOrderStatusRefunded,
		LineItems: []appoms.ReceivePOSSalesOrderLineItemInput{{SKUID: strPtr("sku-1"), Quantity: 1, PriceMinorUnits: 500}},
	})

	require.NoError(t, err)
	assert.Equal(t, omsDomain.SalesOrderStatusRefunded, order.Status)
	assert.False(t, order.Fulfilled, "a non-completed explicit status must not start fulfilled")
}
