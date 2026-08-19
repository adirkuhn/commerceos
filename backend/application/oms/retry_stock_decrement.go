package oms

import (
	"context"
	"errors"
	"fmt"
	"time"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type RetryStockDecrementResult struct {
	Order        omsDomain.SalesOrder
	Resolved     int
	StillPending int
}

// RetryStockDecrementUseCase is the explicit staff action that resolves a
// line item's StockStatusPending state after ErrInsufficientStock aborted
// its original decrement (see ImportWooOrdersUseCase/UpdateSalesOrderUseCase)
// -- staff fix the underlying stock (a receiving, a correction) then trigger
// this from the "Stock Issues" admin page. Deliberately manual, not
// triggered automatically by the stock fix itself: the two are separate
// staff judgment calls (see LinkLineItemSKUUseCase's doc for the same
// reasoning on the SKU-linking side).
//
// A still-insufficient item after retrying is not an error for the call as
// a whole -- that's an expected partial-success outcome, reported via
// StillPending rather than aborting whatever did resolve.
type RetryStockDecrementUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	linkRepo       omsDomain.SalesOrderLineItemStockMovementRepository
	recordMovement recordSaleMovementUC
}

func NewRetryStockDecrementUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, recordMovement recordSaleMovementUC) *RetryStockDecrementUseCase {
	return &RetryStockDecrementUseCase{db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, linkRepo: linkRepo, recordMovement: recordMovement}
}

func (uc *RetryStockDecrementUseCase) Execute(ctx context.Context, salesOrderID string) (*RetryStockDecrementResult, error) {
	var order omsDomain.SalesOrder
	var lineItems []omsDomain.SalesOrderLineItem
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		o, err := uc.salesOrderRepo.FindByID(ctx, salesOrderID)
		if err != nil {
			return err
		}
		if !o.Status.ReducesStock() {
			return omsDomain.ErrOrderNotStockReducing
		}
		order = *o
		lineItems, err = uc.lineItemRepo.ListBySalesOrder(ctx, salesOrderID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("retryStockDecrement: %w", err)
	}

	resolved := 0
	stillPending := 0
	for _, li := range lineItems {
		if li.SKUID == nil || li.StockStatus != omsDomain.StockStatusPending {
			continue
		}
		movement, err := uc.recordMovement.Execute(ctx, invapp.RecordStockMovementInput{
			SKUID: *li.SKUID, WarehouseID: order.WarehouseID, Type: invDomain.MovementTypeSale,
			QuantityDelta: -li.Quantity, Reason: fmt.Sprintf("sales order %s stock retry", salesOrderID), OccurredAt: time.Now(),
		})
		if err != nil {
			if !errors.Is(err, invDomain.ErrInsufficientStock) {
				return nil, fmt.Errorf("retryStockDecrement: %w", err)
			}
			stillPending++
			continue
		}
		if err := linkLineItemMovement(ctx, uc.db, uc.linkRepo, li.ID, movement.ID); err != nil {
			return nil, fmt.Errorf("retryStockDecrement: %w", err)
		}
		if err := setLineItemStockStatus(ctx, uc.db, uc.lineItemRepo, li.ID, omsDomain.StockStatusDecremented); err != nil {
			return nil, fmt.Errorf("retryStockDecrement: %w", err)
		}
		resolved++
	}
	if err := setOrderNeedsStockReview(ctx, uc.db, uc.salesOrderRepo, salesOrderID, stillPending > 0); err != nil {
		return nil, fmt.Errorf("retryStockDecrement: %w", err)
	}
	order.NeedsStockReview = stillPending > 0

	return &RetryStockDecrementResult{Order: order, Resolved: resolved, StillPending: stillPending}, nil
}
