package oms

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// decrementLineItems is import_woo_orders.go's createOne movement loop and
// UpdateSalesOrderUseCase's decrementing branch sharing the exact same
// per-line-item pending logic: a SKUID-less line (never matched a catalog
// SKU) or one whose decrement hits ErrInsufficientStock is marked
// StockStatusPending and the order's NeedsStockReview/NeedsSKUReview
// flagged, instead of hard-failing the whole order the way both call sites
// used to. Staff resolve a pending item via LinkLineItemSKUUseCase (SKU-less
// case) then RetryStockDecrementUseCase (either case) from the "Stock
// Issues"/"Unmatched SKUs" admin pages -- see those use cases' docs.
func decrementLineItems(ctx context.Context, db *gorm.DB, lineItemRepo omsDomain.SalesOrderLineItemRepository, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, salesOrderRepo omsDomain.SalesOrderRepository, skuRepo skuLookupRepo, recordMovement recordSaleMovementUC, outbox eventrecord.Enqueuer, orderID, warehouseID string, lineItems []omsDomain.SalesOrderLineItem) error {
	pending, err := allocateLineItems(ctx, db, lineItemRepo, linkRepo, skuRepo, recordMovement, outbox, warehouseID, lineItems,
		fmt.Sprintf("sales order %s status change", orderID), time.Now())
	if err != nil {
		return err
	}
	return setOrderNeedsStockReview(ctx, db, salesOrderRepo, outbox, orderID, len(pending) > 0, OrderChangeReasonStockShortfall)
}

// allocateLineItems takes each line item's stock (FEFO, see
// AllocateSaleStockUseCase), linking the movements and freezing its cost.
// A SKU-less line, or one short of stock, is left out and its position in
// lineItems returned -- the short ones already marked StockStatusPending --
// for the caller to flag the order; any other error stops it.
func allocateLineItems(ctx context.Context, db *gorm.DB, lineItemRepo omsDomain.SalesOrderLineItemRepository, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, skuRepo skuLookupRepo, recordMovement recordSaleMovementUC, outbox eventrecord.Enqueuer, warehouseID string, lineItems []omsDomain.SalesOrderLineItem, reason string, occurredAt time.Time) (pending []int, err error) {
	for i, li := range lineItems {
		if li.SKUID == nil {
			// Never matched a catalog SKU -- NeedsSKUReview already
			// reflects this from creation time; nothing to decrement
			// against until it's linked.
			pending = append(pending, i)
			continue
		}
		movements, err := recordMovement.Execute(ctx, invapp.AllocateSaleStockInput{
			SKUID: *li.SKUID, WarehouseID: warehouseID, Type: invDomain.MovementTypeSale,
			Quantity: li.Quantity, Reason: reason, OccurredAt: occurredAt,
		})
		if err != nil {
			if !errors.Is(err, invDomain.ErrInsufficientStock) {
				return nil, err
			}
			pending = append(pending, i)
			if err := setLineItemStockStatus(ctx, db, lineItemRepo, outbox, li, omsDomain.StockStatusPending, OrderChangeReasonStockShortfall); err != nil {
				return nil, err
			}
			continue
		}
		for _, movement := range movements {
			if err := linkLineItemMovement(ctx, db, linkRepo, li.ID, movement); err != nil {
				return nil, err
			}
		}
		if err := freezeCostAtSale(ctx, db, lineItemRepo, skuRepo, li.ID, *li.SKUID, li.Quantity); err != nil {
			return nil, err
		}
	}
	return pending, nil
}

// restockLineItems only restocks a line item whose StockStatus is already
// Decremented -- a Pending one (SKUID-less, or a decrement that never
// succeeded) never actually took stock, so crediting it back would over-add.
// Those are instead reset straight to Decremented: the order transitioning
// to a Restocks() status (cancelled/failed) means the decrement is no longer
// owed at all, pending or not.
//
// Reverses each of the line item's ORIGINAL decrement movements batch for
// batch (via the link table's denormalized BatchID/QuantityDelta -- see
// SalesOrderLineItemStockMovement's doc), not a fresh general-ledger credit:
// AllocateSaleStockUseCase can split one line item's decrement across
// several real batches (FEFO/FIFO), so crediting it all back to the general
// ledger instead would leave those batches permanently short. Written as
// Type Adjustment, not Sale -- Sale's sign validation only ever allows a
// negative delta (removing stock), so a positive "giving it back" movement
// has to be the other type explicitly allowed either direction.
func restockLineItems(ctx context.Context, db *gorm.DB, lineItemRepo omsDomain.SalesOrderLineItemRepository, salesOrderRepo omsDomain.SalesOrderRepository, reverseMovement movementReverser, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, outbox eventrecord.Enqueuer, orderID, warehouseID string, lineItems []omsDomain.SalesOrderLineItem) error {
	for _, li := range lineItems {
		if li.SKUID == nil {
			continue
		}
		if li.StockStatus == omsDomain.StockStatusPending {
			if err := setLineItemStockStatus(ctx, db, lineItemRepo, outbox, li, omsDomain.StockStatusDecremented, OrderChangeReasonStockRestored); err != nil {
				return err
			}
			continue
		}
		links, err := listLineItemMovements(ctx, db, linkRepo, li.ID)
		if err != nil {
			return err
		}
		for _, l := range links {
			if l.QuantityDelta >= 0 {
				// Only reverse actual stock-out movements -- a prior
				// reconciliation Adjustment (see ReconcileShortfall) isn't
				// linked to the line item at all, but guard anyway in case
				// a future linked movement is ever non-negative.
				continue
			}
			if _, err := reverseMovement.Execute(ctx, invapp.RecordStockMovementInput{
				SKUID: *li.SKUID, WarehouseID: warehouseID, BatchID: l.BatchID, Type: invDomain.MovementTypeAdjustment,
				QuantityDelta: -l.QuantityDelta, Reason: fmt.Sprintf("sales order %s restocked", orderID), OccurredAt: time.Now(),
			}); err != nil {
				return err
			}
		}
	}
	return setOrderNeedsStockReview(ctx, db, salesOrderRepo, outbox, orderID, false, OrderChangeReasonStockRestored)
}

// listLineItemMovements is linkRepo.ListByLineItem wrapped in its own
// tenant.WithTenant call -- never nest inside another transaction, same
// composition rule as recordSaleMovementUC.
func listLineItemMovements(ctx context.Context, db *gorm.DB, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, lineItemID string) ([]omsDomain.SalesOrderLineItemStockMovement, error) {
	var links []omsDomain.SalesOrderLineItemStockMovement
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		links, err = linkRepo.ListByLineItem(ctx, lineItemID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listLineItemMovements: %w", err)
	}
	return links, nil
}

func setLineItemStockStatus(ctx context.Context, db *gorm.DB, lineItemRepo omsDomain.SalesOrderLineItemRepository, outbox eventrecord.Enqueuer, li omsDomain.SalesOrderLineItem, status omsDomain.StockStatus, reason string) error {
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := lineItemRepo.Update(ctx, li.ID, map[string]any{"stock_status": string(status)}); err != nil {
			return err
		}
		return recordLineItemStockStatus(ctx, outbox, li, status, reason)
	})
	if err != nil {
		return fmt.Errorf("setLineItemStockStatus: %w", err)
	}
	return nil
}

func setOrderNeedsStockReview(ctx context.Context, db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, outbox eventrecord.Enqueuer, orderID string, needsReview bool, reason string) error {
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		order, err := salesOrderRepo.FindByID(ctx, orderID)
		if err != nil {
			return err
		}
		if err := salesOrderRepo.Update(ctx, orderID, map[string]any{"needs_stock_review": needsReview}); err != nil {
			return err
		}
		return recordSalesOrderUpdated(ctx, outbox, orderID,
			eventrecord.NewChanges().Set("needs_stock_review", order.NeedsStockReview, needsReview), reason)
	})
	if err != nil {
		return fmt.Errorf("setOrderNeedsStockReview: %w", err)
	}
	return nil
}
