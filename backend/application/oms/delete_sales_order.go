package oms

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	"github.com/adirkuhn/commerceos/domain/events"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// DeleteSalesOrderUseCase removes a sales order and its line items,
// restocking first if stock had actually been decremented for it (see
// SalesOrder.StockDecremented's doc) -- deleting a completed sale must not
// silently leave its stock impact applied with nothing to show for it.
//
// Restock (if needed) happens before the delete transaction: if it fails,
// the order is left untouched rather than deleted with its stock impact
// unresolved.
type DeleteSalesOrderUseCase struct {
	outbox          eventrecord.Enqueuer
	db              *gorm.DB
	salesOrderRepo  omsDomain.SalesOrderRepository
	lineItemRepo    omsDomain.SalesOrderLineItemRepository
	linkRepo        omsDomain.SalesOrderLineItemStockMovementRepository
	reverseMovement movementReverser
}

func NewDeleteSalesOrderUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, reverseMovement movementReverser, outbox eventrecord.Enqueuer) *DeleteSalesOrderUseCase {
	return &DeleteSalesOrderUseCase{outbox: outbox, db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, linkRepo: linkRepo, reverseMovement: reverseMovement}
}

func (uc *DeleteSalesOrderUseCase) Execute(ctx context.Context, id string) error {
	var order *omsDomain.SalesOrder
	var lineItems []omsDomain.SalesOrderLineItem
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.salesOrderRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		order = existing
		if existing.StockDecremented {
			lineItems, err = uc.lineItemRepo.ListBySalesOrder(ctx, id)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("deleteSalesOrder: %w", err)
	}

	for _, li := range lineItems {
		if li.SKUID == nil || li.StockStatus == omsDomain.StockStatusPending {
			continue
		}
		links, err := listLineItemMovements(ctx, uc.db, uc.linkRepo, li.ID)
		if err != nil {
			return fmt.Errorf("deleteSalesOrder: %w", err)
		}
		for _, l := range links {
			if l.QuantityDelta >= 0 {
				continue
			}
			if _, err := uc.reverseMovement.Execute(ctx, invapp.RecordStockMovementInput{
				SKUID: *li.SKUID, WarehouseID: order.WarehouseID, BatchID: l.BatchID, Type: invDomain.MovementTypeAdjustment,
				QuantityDelta: -l.QuantityDelta, Reason: fmt.Sprintf("sales order %s deleted", id),
				OccurredAt: time.Now(),
			}); err != nil {
				return fmt.Errorf("deleteSalesOrder: %w", err)
			}
		}
	}

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.lineItemRepo.DeleteBySalesOrder(ctx, id); err != nil {
			return err
		}
		if err := uc.salesOrderRepo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.SalesOrderDeleted, id, eventrecord.Data{
			"source": string(order.Source), "number": order.Number, "status": string(order.Status),
			"stock_restored": order.StockDecremented,
		})
	})
	if err != nil {
		return fmt.Errorf("deleteSalesOrder: %w", err)
	}
	return nil
}
