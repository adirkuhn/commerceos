package oms

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

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
func decrementLineItems(ctx context.Context, db *gorm.DB, lineItemRepo omsDomain.SalesOrderLineItemRepository, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, salesOrderRepo omsDomain.SalesOrderRepository, skuRepo skuLookupRepo, recordMovement recordSaleMovementUC, orderID, warehouseID string, lineItems []omsDomain.SalesOrderLineItem) error {
	stillPending := false
	for _, li := range lineItems {
		if li.SKUID == nil {
			// Never matched a catalog SKU -- NeedsSKUReview already
			// reflects this from creation time; nothing to decrement
			// against until it's linked.
			stillPending = true
			continue
		}
		movements, err := recordMovement.Execute(ctx, invapp.AllocateSaleStockInput{
			SKUID: *li.SKUID, WarehouseID: warehouseID, Type: invDomain.MovementTypeSale,
			Quantity: li.Quantity, Reason: fmt.Sprintf("sales order %s status change", orderID), OccurredAt: time.Now(),
		})
		if err != nil {
			if !errors.Is(err, invDomain.ErrInsufficientStock) {
				return err
			}
			stillPending = true
			if err := setLineItemStockStatus(ctx, db, lineItemRepo, li.ID, omsDomain.StockStatusPending); err != nil {
				return err
			}
			continue
		}
		for _, movement := range movements {
			if err := linkLineItemMovement(ctx, db, linkRepo, li.ID, movement); err != nil {
				return err
			}
		}
		if err := freezeCostAtSale(ctx, db, lineItemRepo, skuRepo, li.ID, *li.SKUID, li.Quantity); err != nil {
			return err
		}
	}
	return setOrderNeedsStockReview(ctx, db, salesOrderRepo, orderID, stillPending)
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
func restockLineItems(ctx context.Context, db *gorm.DB, lineItemRepo omsDomain.SalesOrderLineItemRepository, salesOrderRepo omsDomain.SalesOrderRepository, reverseMovement movementReverser, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, orderID, warehouseID string, lineItems []omsDomain.SalesOrderLineItem) error {
	for _, li := range lineItems {
		if li.SKUID == nil {
			continue
		}
		if li.StockStatus == omsDomain.StockStatusPending {
			if err := setLineItemStockStatus(ctx, db, lineItemRepo, li.ID, omsDomain.StockStatusDecremented); err != nil {
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
	return setOrderNeedsStockReview(ctx, db, salesOrderRepo, orderID, false)
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

func setLineItemStockStatus(ctx context.Context, db *gorm.DB, lineItemRepo omsDomain.SalesOrderLineItemRepository, lineItemID string, status omsDomain.StockStatus) error {
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return lineItemRepo.Update(ctx, lineItemID, map[string]any{"stock_status": string(status)})
	})
	if err != nil {
		return fmt.Errorf("setLineItemStockStatus: %w", err)
	}
	return nil
}

func setOrderNeedsStockReview(ctx context.Context, db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, orderID string, needsReview bool) error {
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return salesOrderRepo.Update(ctx, orderID, map[string]any{"needs_stock_review": needsReview})
	})
	if err != nil {
		return fmt.Errorf("setOrderNeedsStockReview: %w", err)
	}
	return nil
}
