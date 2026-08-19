package oms_test

import (
	"context"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubRepairSalesOrderRepo layers List (paginated, source-filtered) on top
// of stubSalesOrderRepo's existing fields -- kept separate so the simpler
// update_sales_order_test.go stub isn't burdened with pagination logic it
// doesn't need.
type stubRepairSalesOrderRepo struct {
	stubSalesOrderRepo
	all []omsDomain.SalesOrder
}

func (s *stubRepairSalesOrderRepo) List(_ context.Context, filter omsDomain.ListSalesOrdersFilter) ([]omsDomain.SalesOrder, int64, error) {
	s.gotListFilter = filter
	start := filter.Offset
	if start > len(s.all) {
		return nil, int64(len(s.all)), nil
	}
	end := start + filter.Limit
	if end > len(s.all) {
		end = len(s.all)
	}
	return s.all[start:end], int64(len(s.all)), nil
}

func TestRepairStockReviewFlags_flagsAnOrderWithAMissingMovementDespiteStockDecrementedTrue(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubRepairSalesOrderRepo{all: []omsDomain.SalesOrder{
		{ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, Status: omsDomain.SalesOrderStatusProcessing, StockDecremented: true},
	}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SalesOrderID: "so-1", SKUID: strPtr("sku-1"), StockStatus: omsDomain.StockStatusDecremented},
	}}
	linkRepo := &stubSalesOrderLineItemStockMovementRepo{existsResult: false}
	uc := appoms.NewRepairStockReviewFlagsUseCase(db, salesOrderRepo, lineItemRepo, linkRepo)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, result.OrdersScanned)
	assert.Equal(t, 1, result.LineItemsFixed)
	assert.Equal(t, 1, result.OrdersFlagged)
	assert.Equal(t, []string{"so-1"}, result.FlaggedOrderIDs)
	require.Len(t, lineItemRepo.updateCalls, 1)
	assert.Equal(t, string(omsDomain.StockStatusPending), lineItemRepo.updateCalls[0].Fields["stock_status"])
	require.NotEmpty(t, salesOrderRepo.updateCalls)
	assert.Equal(t, true, salesOrderRepo.updateCalls[0].Fields["needs_stock_review"])
}

func TestRepairStockReviewFlags_leavesAnOrderWithARecordedMovementUntouched(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubRepairSalesOrderRepo{all: []omsDomain.SalesOrder{
		{ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, Status: omsDomain.SalesOrderStatusProcessing, StockDecremented: true},
	}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SalesOrderID: "so-1", SKUID: strPtr("sku-1"), StockStatus: omsDomain.StockStatusDecremented},
	}}
	linkRepo := &stubSalesOrderLineItemStockMovementRepo{existsResult: true}
	uc := appoms.NewRepairStockReviewFlagsUseCase(db, salesOrderRepo, lineItemRepo, linkRepo)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 0, result.LineItemsFixed)
	assert.Equal(t, 0, result.OrdersFlagged)
	assert.Empty(t, lineItemRepo.updateCalls)
	assert.Empty(t, salesOrderRepo.updateCalls)
}

func TestRepairStockReviewFlags_ignoresASkipStockMovementsBootstrapOrder(t *testing.T) {
	db := newMockGormDB(t)
	// StockDecremented=false even though status reduces stock -- the
	// SkipStockMovements bootstrap signature. Must never be flagged even
	// though its line item also has no recorded movement, by design.
	salesOrderRepo := &stubRepairSalesOrderRepo{all: []omsDomain.SalesOrder{
		{ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, Status: omsDomain.SalesOrderStatusProcessing, StockDecremented: false},
	}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SalesOrderID: "so-1", SKUID: strPtr("sku-1"), StockStatus: omsDomain.StockStatusDecremented},
	}}
	linkRepo := &stubSalesOrderLineItemStockMovementRepo{existsResult: false}
	uc := appoms.NewRepairStockReviewFlagsUseCase(db, salesOrderRepo, lineItemRepo, linkRepo)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 0, result.LineItemsFixed)
	assert.Equal(t, 0, result.OrdersFlagged)
	assert.Empty(t, lineItemRepo.updateCalls)
}

func TestRepairStockReviewFlags_flagsAnUnmatchedSKULineItemAndMarksItPending(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubRepairSalesOrderRepo{all: []omsDomain.SalesOrder{
		{ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, Status: omsDomain.SalesOrderStatusProcessing, StockDecremented: true},
	}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SalesOrderID: "so-1", SKUID: nil, StockStatus: omsDomain.StockStatusDecremented},
	}}
	linkRepo := &stubSalesOrderLineItemStockMovementRepo{}
	uc := appoms.NewRepairStockReviewFlagsUseCase(db, salesOrderRepo, lineItemRepo, linkRepo)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, result.LineItemsFixed)
	require.Len(t, lineItemRepo.updateCalls, 1)
	assert.Equal(t, string(omsDomain.StockStatusPending), lineItemRepo.updateCalls[0].Fields["stock_status"])
	require.Len(t, salesOrderRepo.updateCalls, 2, "both needs_sku_review and needs_stock_review must be flagged")
}

func TestRepairStockReviewFlags_isIdempotentOnASecondRun(t *testing.T) {
	db := newMockGormDB(t)
	salesOrderRepo := &stubRepairSalesOrderRepo{all: []omsDomain.SalesOrder{
		{ID: "so-1", Source: omsDomain.SalesOrderSourceWooCommerce, Status: omsDomain.SalesOrderStatusProcessing,
			StockDecremented: true, NeedsStockReview: true},
	}}
	lineItemRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: []omsDomain.SalesOrderLineItem{
		{ID: "li-1", SalesOrderID: "so-1", SKUID: strPtr("sku-1"), StockStatus: omsDomain.StockStatusPending},
	}}
	linkRepo := &stubSalesOrderLineItemStockMovementRepo{existsResult: false}
	uc := appoms.NewRepairStockReviewFlagsUseCase(db, salesOrderRepo, lineItemRepo, linkRepo)

	result, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 0, result.LineItemsFixed, "already pending -- must not be re-touched")
	assert.Equal(t, 0, result.OrdersFlagged, "already flagged -- must not re-flag")
	assert.Empty(t, lineItemRepo.updateCalls)
	assert.Empty(t, salesOrderRepo.updateCalls)
}
