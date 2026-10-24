package oms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddSalesOrderLineItem_createsLineItemAndBumpsOrderTotal(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", TotalMinorUnits: 1000}}
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1", SKUCode: "SKU-1", Barcode: "BC-1"}}}
	outbox := &eventrecordtest.Outbox{}
	uc := appoms.NewAddSalesOrderLineItemUseCase(db, salesOrderRepo, lineItemRepo, skuRepo, &stubLinePricer{includesTax: true}, outbox)

	li, err := uc.Execute(context.Background(), appoms.AddSalesOrderLineItemInput{
		SalesOrderID: "order-1", SKUID: "sku-1", Name: "Freebie Widget", Quantity: 2, PriceMinorUnits: 500,
	})

	require.NoError(t, err)
	require.Len(t, lineItemRepo.created, 1)
	assert.Equal(t, "SKU-1", li.SKUCode)
	assert.Equal(t, int64(2), li.Quantity)
	assert.Equal(t, int64(1000), li.TotalMinorUnits) // 500 * 2
	require.Len(t, salesOrderRepo.updateCalls, 1)
	assert.Equal(t, "order-1", salesOrderRepo.updateCalls[0].ID)
	assert.Equal(t, int64(2000), salesOrderRepo.updateCalls[0].Fields["total_minor_units"]) // 1000 + 1000

	assert.Contains(t, outbox.Subjects(), "oms.sales_order_line_item.added")
	assert.Contains(t, outbox.Subjects(), "oms.sales_order.updated")

}

func TestAddSalesOrderLineItem_zeroPriceIsAValidFreebie(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", TotalMinorUnits: 1000}}
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1"}}}
	uc := appoms.NewAddSalesOrderLineItemUseCase(db, salesOrderRepo, lineItemRepo, skuRepo, &stubLinePricer{includesTax: true}, &eventrecordtest.Outbox{})

	li, err := uc.Execute(context.Background(), appoms.AddSalesOrderLineItemInput{
		SalesOrderID: "order-1", SKUID: "sku-1", Name: "Freebie", Quantity: 1, PriceMinorUnits: 0,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(0), li.TotalMinorUnits)
	assert.Equal(t, int64(1000), salesOrderRepo.updateCalls[0].Fields["total_minor_units"]) // unchanged
}

func TestAddSalesOrderLineItem_propagatesOrderNotFound(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDErr: omsDomain.ErrSalesOrderNotFound}
	uc := appoms.NewAddSalesOrderLineItemUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubLinePricer{includesTax: true}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.AddSalesOrderLineItemInput{SalesOrderID: "missing", SKUID: "sku-1", Quantity: 1, PriceMinorUnits: 0})

	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}

func TestAddSalesOrderLineItem_propagatesSKUNotFound(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	skuRepo := &stubSKURepo{findErr: pmsDomain.ErrSKUNotFound}
	uc := appoms.NewAddSalesOrderLineItemUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, skuRepo, &stubLinePricer{includesTax: true}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.AddSalesOrderLineItemInput{SalesOrderID: "order-1", SKUID: "missing", Quantity: 1, PriceMinorUnits: 0})

	assert.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
}

func TestAddSalesOrderLineItem_rejectsNonPositiveQuantity(t *testing.T) {
	uc := appoms.NewAddSalesOrderLineItemUseCase(nil, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubLinePricer{includesTax: true}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.AddSalesOrderLineItemInput{SalesOrderID: "order-1", SKUID: "sku-1", Quantity: 0, PriceMinorUnits: 0})

	assert.ErrorIs(t, err, omsDomain.ErrInvalidLineItemQuantity)
}

func TestAddSalesOrderLineItem_rejectsNegativePrice(t *testing.T) {
	uc := appoms.NewAddSalesOrderLineItemUseCase(nil, &stubSalesOrderRepo{}, &stubSalesOrderLineItemRepo{}, &stubSKURepo{}, &stubLinePricer{includesTax: true}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.AddSalesOrderLineItemInput{SalesOrderID: "order-1", SKUID: "sku-1", Quantity: 1, PriceMinorUnits: -1})

	assert.ErrorIs(t, err, omsDomain.ErrInvalidPrice)
}

type stubLinePricer struct {
	rate        int64
	includesTax bool
	err         error
}

func (s *stubLinePricer) PriceLine(_ context.Context, _ *string, quantity, unit int64) (omsDomain.LineAmounts, bool, error) {
	return omsDomain.PriceLine(quantity, unit, s.rate, s.includesTax), s.includesTax, s.err
}

func TestAddSalesOrderLineItem_splitsTheVATOutOfATaxInclusivePrice(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", TotalMinorUnits: 1000, TaxTotalMinorUnits: 100}}
	lineItemRepo := &stubSalesOrderLineItemRepo{}
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1"}}}
	uc := appoms.NewAddSalesOrderLineItemUseCase(db, salesOrderRepo, lineItemRepo, skuRepo, &stubLinePricer{rate: 900, includesTax: true}, &eventrecordtest.Outbox{})

	li, err := uc.Execute(context.Background(), appoms.AddSalesOrderLineItemInput{SalesOrderID: "order-1", SKUID: "sku-1", Quantity: 2, PriceMinorUnits: 199})

	require.NoError(t, err)
	assert.True(t, li.TotalIncludesTax)
	assert.Equal(t, int64(398), li.TotalMinorUnits)
	assert.Equal(t, int64(33), li.TotalTaxMinorUnits)
	assert.Equal(t, int64(1398), salesOrderRepo.updateCalls[0].Fields["total_minor_units"])
	assert.Equal(t, int64(133), salesOrderRepo.updateCalls[0].Fields["tax_total_minor_units"])
}

func TestAddSalesOrderLineItem_addsTheVATOnTopOfATaxExclusivePrice(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", TotalMinorUnits: 1000}}
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1"}}}
	uc := appoms.NewAddSalesOrderLineItemUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, skuRepo, &stubLinePricer{rate: 2100}, &eventrecordtest.Outbox{})

	li, err := uc.Execute(context.Background(), appoms.AddSalesOrderLineItemInput{SalesOrderID: "order-1", SKUID: "sku-1", Quantity: 1, PriceMinorUnits: 1000})

	require.NoError(t, err)
	assert.False(t, li.TotalIncludesTax)
	assert.Equal(t, int64(2210), salesOrderRepo.updateCalls[0].Fields["total_minor_units"])
	assert.Equal(t, int64(210), salesOrderRepo.updateCalls[0].Fields["tax_total_minor_units"])
}

func TestAddSalesOrderLineItem_propagatesAPricingError(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}
	skuRepo := &stubSKURepo{byID: map[string]*pmsDomain.SKU{"sku-1": {ID: "sku-1"}}}
	boom := errors.New("boom")
	uc := appoms.NewAddSalesOrderLineItemUseCase(db, salesOrderRepo, &stubSalesOrderLineItemRepo{}, skuRepo, &stubLinePricer{err: boom}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.AddSalesOrderLineItemInput{SalesOrderID: "order-1", SKUID: "sku-1", Quantity: 1, PriceMinorUnits: 1000})

	assert.ErrorIs(t, err, boom)
}
