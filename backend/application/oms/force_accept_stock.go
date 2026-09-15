package oms

import (
	"context"
	"fmt"
	"time"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ForceAcceptStockResult struct {
	Order    omsDomain.SalesOrder
	Resolved int
}

// ForceAcceptStockUseCase is the explicit staff override for a pending line
// item when the ledger is simply wrong -- staff know the item physically
// exists (it's on the shelf, it was scanned, whatever), CommerceOS's
// tracked stock just doesn't reflect it. Unlike RetryStockDecrementUseCase
// (which re-attempts the same strict check and can still leave items
// pending), this always succeeds for every SKU-linked pending line item: it
// goes through AllocateSaleStockUseCase with ReconcileShortfall, which
// writes a reconciliation Adjustment for whatever's short before completing
// the sale, the same escape hatch POS sales already use for "this already
// physically happened, don't block it." A SKUID-less line item is skipped
// entirely -- there's no SKU to draw stock from at all, that's
// LinkLineItemSKUUseCase's job on the "Unmatched SKUs" page, not this one.
type ForceAcceptStockUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	linkRepo       omsDomain.SalesOrderLineItemStockMovementRepository
	skuRepo        skuLookupRepo
	recordMovement recordSaleMovementUC
}

func NewForceAcceptStockUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, skuRepo skuLookupRepo, recordMovement recordSaleMovementUC) *ForceAcceptStockUseCase {
	return &ForceAcceptStockUseCase{db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, linkRepo: linkRepo, skuRepo: skuRepo, recordMovement: recordMovement}
}

func (uc *ForceAcceptStockUseCase) Execute(ctx context.Context, salesOrderID string) (*ForceAcceptStockResult, error) {
	var order omsDomain.SalesOrder
	var lineItems []omsDomain.SalesOrderLineItem
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		o, err := uc.salesOrderRepo.FindByID(ctx, salesOrderID)
		if err != nil {
			return err
		}
		order = *o
		lineItems, err = uc.lineItemRepo.ListBySalesOrder(ctx, salesOrderID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("forceAcceptStock: %w", err)
	}

	resolved := 0
	for _, li := range lineItems {
		if li.SKUID == nil || li.StockStatus != omsDomain.StockStatusPending {
			continue
		}
		movements, err := uc.recordMovement.Execute(ctx, invapp.AllocateSaleStockInput{
			SKUID: *li.SKUID, WarehouseID: order.WarehouseID, Type: invDomain.MovementTypeSale,
			Quantity: li.Quantity, Reason: fmt.Sprintf("sales order %s stock force-accepted", salesOrderID), OccurredAt: time.Now(),
			ReconcileShortfall: true,
		})
		if err != nil {
			return nil, fmt.Errorf("forceAcceptStock: %w", err)
		}
		for _, movement := range movements {
			if err := linkLineItemMovement(ctx, uc.db, uc.linkRepo, li.ID, movement); err != nil {
				return nil, fmt.Errorf("forceAcceptStock: %w", err)
			}
		}
		if err := setLineItemStockStatus(ctx, uc.db, uc.lineItemRepo, li.ID, omsDomain.StockStatusDecremented); err != nil {
			return nil, fmt.Errorf("forceAcceptStock: %w", err)
		}
		if err := freezeCostAtSale(ctx, uc.db, uc.lineItemRepo, uc.skuRepo, li.ID, *li.SKUID, li.Quantity); err != nil {
			return nil, fmt.Errorf("forceAcceptStock: %w", err)
		}
		resolved++
	}
	if err := setOrderNeedsStockReview(ctx, uc.db, uc.salesOrderRepo, salesOrderID, false); err != nil {
		return nil, fmt.Errorf("forceAcceptStock: %w", err)
	}
	order.NeedsStockReview = false

	return &ForceAcceptStockResult{Order: order, Resolved: resolved}, nil
}
