package procurement

import (
	"context"
	"fmt"

	pmsapp "github.com/adirkuhn/commerceos/application/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ListReceivedCostLinesForSKUUseCase gives pms's cost-price rollup
// (RecalculateCostPricesUseCase) a cost basis for POs that were marked
// received/partially_received without ever creating an inventory.Batch --
// e.g. via the desktop Edit Purchase Order form's status field, which has no
// stock-booking gate of its own (see UpdatePurchaseOrderUseCase). Returns
// pmsapp.PurchaseCostLine directly (application/procurement already imports
// application/pms elsewhere, e.g. create_product_and_link_line_item.go, so
// this isn't a new dependency) -- this use case's Execute method structurally
// satisfies pms's own unexported purchaseCostLineListerForCostPrice
// interface, no adapter needed. The pms use case is responsible for
// excluding any line item that a real Batch already accounts for (via
// Batch.PurchaseOrderLineItemID), so a properly booked receipt is never
// double-counted between the two sources.
type ListReceivedCostLinesForSKUUseCase struct {
	db           *gorm.DB
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
}

func NewListReceivedCostLinesForSKUUseCase(db *gorm.DB, lineItemRepo procurementDomain.PurchaseOrderLineItemRepository) *ListReceivedCostLinesForSKUUseCase {
	return &ListReceivedCostLinesForSKUUseCase{db: db, lineItemRepo: lineItemRepo}
}

func (uc *ListReceivedCostLinesForSKUUseCase) Execute(ctx context.Context, skuID string) ([]pmsapp.PurchaseCostLine, error) {
	var lines []pmsapp.PurchaseCostLine
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		poIDs, err := uc.lineItemRepo.ListPurchaseOrderIDsReceivedBySKU(ctx, skuID)
		if err != nil {
			return err
		}
		for _, poID := range poIDs {
			items, err := uc.lineItemRepo.ListByPurchaseOrder(ctx, poID)
			if err != nil {
				return err
			}
			for _, li := range items {
				if li.SKUID == nil || *li.SKUID != skuID || !li.LineType.IsProduct() {
					continue
				}
				cost, err := AllocateLandedUnitCost(items, li.ID)
				if err != nil {
					// li came from items itself, so this can't happen -- skip
					// rather than fail the whole SKU on a defensive error.
					continue
				}
				lines = append(lines, pmsapp.PurchaseCostLine{
					PurchaseOrderLineItemID:  li.ID,
					Quantity:                 li.Quantity * li.EffectiveUnitsPerPurchasedItem(),
					LandedUnitCostMinorUnits: cost,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listReceivedCostLinesForSKU: %w", err)
	}
	return lines, nil
}
