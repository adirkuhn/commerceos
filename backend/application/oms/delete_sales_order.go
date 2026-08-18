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

// DeleteSalesOrderUseCase removes a sales order and its line items,
// restocking first if stock had actually been decremented for it (see
// SalesOrder.StockDecremented's doc) -- deleting a completed sale must not
// silently leave its stock impact applied with nothing to show for it.
//
// Restock (if needed) happens before the delete transaction: if it fails,
// the order is left untouched rather than deleted with its stock impact
// unresolved.
type DeleteSalesOrderUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	recordMovement recordSaleMovementUC
}

func NewDeleteSalesOrderUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository, recordMovement recordSaleMovementUC) *DeleteSalesOrderUseCase {
	return &DeleteSalesOrderUseCase{db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, recordMovement: recordMovement}
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
		if li.SKUID == nil {
			continue
		}
		if _, err := uc.recordMovement.Execute(ctx, invapp.RecordStockMovementInput{
			SKUID: *li.SKUID, WarehouseID: order.WarehouseID, Type: invDomain.MovementTypeSale,
			QuantityDelta: li.Quantity, Reason: fmt.Sprintf("sales order %s deleted", id),
			OccurredAt: time.Now(),
		}); err != nil {
			return fmt.Errorf("deleteSalesOrder: %w", err)
		}
	}

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.lineItemRepo.DeleteBySalesOrder(ctx, id); err != nil {
			return err
		}
		return uc.salesOrderRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteSalesOrder: %w", err)
	}
	return nil
}
