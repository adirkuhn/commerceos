package oms

import (
	"context"
	"fmt"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// freezeCostAtSale writes a line item's cost_at_sale_minor_units/
// cogs_minor_units from the SKU's current moving-average cost -- its own
// tenant.WithTenant call, same composition rule as linkLineItemMovement:
// never nest inside another transaction. Called right after a sale's stock
// movement is recorded, from every place a sales order line actually
// decrements stock (CreateSalesOrderUseCase, UpdateSalesOrderUseCase,
// CompletePickingUseCase, ReceivePOSSalesOrderUseCase,
// ForceAcceptStockUseCase, RetryStockDecrementUseCase).
//
// The cost frozen is what the stock actually cost -- the SKU's inventory
// cost (moving average), not its cost price (the last receipt's
// replacement cost, see pms.SKU.ActualCostMinorUnits).
//
// A no-op (not an error) when the SKU's cost is currently unknown
// (no inventory cost nor cost price, pms.SKUCostPriceSourceUnknown) -- the line's
// cost/COGS are simply left nil, never a fabricated 0 (see
// SalesOrderLineItem.CostAtSaleMinorUnits's doc comment).
//
//events:none cost snapshot taken as part of a sale; the sale itself raises the event
func freezeCostAtSale(ctx context.Context, db *gorm.DB, lineItemRepo omsDomain.SalesOrderLineItemRepository, skuRepo skuLookupRepo, lineItemID, skuID string, quantity int64) error {
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		sku, err := skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		actual := sku.ActualCostMinorUnits()
		if actual == nil {
			return nil
		}
		cost := *actual
		return lineItemRepo.Update(ctx, lineItemID, map[string]any{
			"cost_at_sale_minor_units": cost,
			"cogs_minor_units":         cost * quantity,
		})
	})
	if err != nil {
		return fmt.Errorf("freezeCostAtSale: %w", err)
	}
	return nil
}
