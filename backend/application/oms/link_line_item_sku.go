package oms

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type LinkLineItemSKUInput struct {
	SalesOrderID string
	LineItemID   string
	SKUID        string
}

// LinkLineItemSKUUseCase resolves one line item's NeedsSKUReview state -- a
// line whose source SKU string never matched a catalog SKU at import time
// (SKUID nil, see ImportWooOrdersUseCase) is linked to a real SKU by staff
// from the "Unmatched SKUs" admin page. Deliberately does NOT attempt a
// stock decrement itself (that's RetryStockDecrementUseCase, a separate
// explicit staff action -- see this codebase's manual-retry-only design for
// the stock review flow) -- linking and decrementing are two different staff
// judgment calls that shouldn't be silently coupled.
type LinkLineItemSKUUseCase struct {
	outbox         eventrecord.Enqueuer
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	skuRepo        skuLookupRepo
}

func NewLinkLineItemSKUUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository, skuRepo skuLookupRepo, outbox eventrecord.Enqueuer) *LinkLineItemSKUUseCase {
	return &LinkLineItemSKUUseCase{outbox: outbox, db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, skuRepo: skuRepo}
}

func (uc *LinkLineItemSKUUseCase) Execute(ctx context.Context, in LinkLineItemSKUInput) (*omsDomain.SalesOrder, error) {
	var updated omsDomain.SalesOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		li, err := uc.lineItemRepo.FindByID(ctx, in.LineItemID)
		if err != nil {
			return err
		}
		if li.SalesOrderID != in.SalesOrderID {
			return omsDomain.ErrLineItemNotOnOrder
		}
		if _, err := uc.skuRepo.FindByID(ctx, in.SKUID); err != nil {
			return err
		}
		if err := uc.lineItemRepo.Update(ctx, in.LineItemID, map[string]any{"sku_id": in.SKUID}); err != nil {
			return err
		}
		if err := recordLineItemLinked(ctx, uc.outbox, *li, in.SKUID, LineLinkSourceManual); err != nil {
			return err
		}

		lineItems, err := uc.lineItemRepo.ListBySalesOrder(ctx, in.SalesOrderID)
		if err != nil {
			return err
		}
		stillUnlinked := false
		for _, item := range lineItems {
			if item.SKUID == nil {
				stillUnlinked = true
				break
			}
		}
		if err := setNeedsSKUReview(ctx, uc.salesOrderRepo, uc.outbox, in.SalesOrderID, stillUnlinked); err != nil {
			return err
		}

		order, err := uc.salesOrderRepo.FindByID(ctx, in.SalesOrderID)
		if err != nil {
			return err
		}
		updated = *order
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("linkLineItemSKU: %w", err)
	}
	return &updated, nil
}
