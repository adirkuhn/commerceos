package oms

import (
	"context"
	"fmt"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const repairScanBatchSize = 500

type RepairStockReviewFlagsResult struct {
	OrdersScanned   int
	LineItemsFixed  int
	OrdersFlagged   int
	FlaggedOrderIDs []string
}

// RepairStockReviewFlagsUseCase is the one-time fix for WooCommerce orders
// stuck in an inconsistent state by the bug this feature fixes: before
// per-line-item StockStatus/NeedsStockReview/NeedsSKUReview existed,
// UpdateSalesOrderUseCase committed the header transaction (status,
// StockDecremented=true) BEFORE attempting the actual stock movement --
// so an order whose movement then failed with ErrInsufficientStock was left
// claiming StockDecremented=true with no stock_movement ever recorded for
// it, and no way for staff to discover or resolve that short of a manual DB
// query. This use case runs within one tenant's context (see
// cmd/repair-stock-review-flags for the cross-tenant loop) and:
//
//  1. Flags NeedsSKUReview on every WooCommerce order with a line item that
//     still has no sku_id, and marks that line item StockStatusPending if
//     its order currently owes a decrement (ReducesStock()) -- it can never
//     have been decremented (the movement loop always skips a nil SKUID).
//  2. For every WooCommerce order with StockDecremented=true, checks each
//     sku_id-linked line item for a recorded stock movement (via
//     SalesOrderLineItemStockMovementRepository.ExistsForLineItem). A
//     matched-SKU line item on such an order that has NO movement recorded
//     never actually got its decrement applied despite the order claiming
//     otherwise -- marked StockStatusPending, order flagged
//     NeedsStockReview.
//
// Deliberately scoped to StockDecremented=true orders only for (2) -- an
// order imported with SkipStockMovements (the post-first-products-import
// bootstrap) can legitimately have a stock-reducing status with
// StockDecremented=false and no movements by design; scanning those would
// wrongly flag correct data. Idempotent/safe to re-run: an order/line item
// already in the correct state is left untouched.
type RepairStockReviewFlagsUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	linkRepo       omsDomain.SalesOrderLineItemStockMovementRepository
}

func NewRepairStockReviewFlagsUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository) *RepairStockReviewFlagsUseCase {
	return &RepairStockReviewFlagsUseCase{db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, linkRepo: linkRepo}
}

func (uc *RepairStockReviewFlagsUseCase) Execute(ctx context.Context) (*RepairStockReviewFlagsResult, error) {
	result := &RepairStockReviewFlagsResult{}
	source := omsDomain.SalesOrderSourceWooCommerce

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		offset := 0
		for {
			orders, _, err := uc.salesOrderRepo.List(ctx, omsDomain.ListSalesOrdersFilter{
				Limit: repairScanBatchSize, Offset: offset, Source: &source,
			})
			if err != nil {
				return err
			}
			if len(orders) == 0 {
				return nil
			}
			for _, order := range orders {
				result.OrdersScanned++
				lineItemsFixed, orderFlagged, err := uc.repairOne(ctx, order)
				if err != nil {
					return fmt.Errorf("order %s: %w", order.ID, err)
				}
				result.LineItemsFixed += lineItemsFixed
				if orderFlagged {
					result.OrdersFlagged++
					result.FlaggedOrderIDs = append(result.FlaggedOrderIDs, order.ID)
				}
			}
			if len(orders) < repairScanBatchSize {
				return nil
			}
			offset += repairScanBatchSize
		}
	})
	if err != nil {
		return nil, fmt.Errorf("repairStockReviewFlags: %w", err)
	}
	return result, nil
}

func (uc *RepairStockReviewFlagsUseCase) repairOne(ctx context.Context, order omsDomain.SalesOrder) (lineItemsFixed int, orderFlagged bool, err error) {
	lineItems, err := uc.lineItemRepo.ListBySalesOrder(ctx, order.ID)
	if err != nil {
		return 0, false, err
	}

	unmatchedFound := false
	stockIssueFound := false
	reducesStock := order.Status.ReducesStock()

	for _, li := range lineItems {
		if li.SKUID == nil {
			unmatchedFound = true
			if reducesStock && li.StockStatus != omsDomain.StockStatusPending {
				if err := uc.lineItemRepo.Update(ctx, li.ID, map[string]any{"stock_status": string(omsDomain.StockStatusPending)}); err != nil {
					return 0, false, err
				}
				stockIssueFound = true
				lineItemsFixed++
			}
			continue
		}
		if !order.StockDecremented || li.StockStatus == omsDomain.StockStatusPending {
			continue
		}
		exists, err := uc.linkRepo.ExistsForLineItem(ctx, li.ID)
		if err != nil {
			return 0, false, err
		}
		if !exists {
			if err := uc.lineItemRepo.Update(ctx, li.ID, map[string]any{"stock_status": string(omsDomain.StockStatusPending)}); err != nil {
				return 0, false, err
			}
			stockIssueFound = true
			lineItemsFixed++
		}
	}

	if unmatchedFound && !order.NeedsSKUReview {
		if err := uc.salesOrderRepo.Update(ctx, order.ID, map[string]any{"needs_sku_review": true}); err != nil {
			return 0, false, err
		}
		orderFlagged = true
	}
	if stockIssueFound && !order.NeedsStockReview {
		if err := uc.salesOrderRepo.Update(ctx, order.ID, map[string]any{"needs_stock_review": true}); err != nil {
			return 0, false, err
		}
		orderFlagged = true
	}
	return lineItemsFixed, orderFlagged, nil
}
