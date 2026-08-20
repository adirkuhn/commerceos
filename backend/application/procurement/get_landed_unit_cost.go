package procurement

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

// GetLandedUnitCostUseCase computes a PO line item's landed unit cost (see
// AllocateLandedUnitCost) for a caller that isn't already holding every line
// item on the order in memory -- the desktop receiving review flow, which
// calls this once per line right before booking stock via inventory's
// generic receive-stock endpoint, passing the result back as
// unit_cost_minor_units/purchase_order_line_item_id on that call. Read-only,
// no transaction needed beyond the lookup itself.
type GetLandedUnitCostUseCase struct {
	db           *gorm.DB
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
}

func NewGetLandedUnitCostUseCase(db *gorm.DB, lineItemRepo procurementDomain.PurchaseOrderLineItemRepository) *GetLandedUnitCostUseCase {
	return &GetLandedUnitCostUseCase{db: db, lineItemRepo: lineItemRepo}
}

func (uc *GetLandedUnitCostUseCase) Execute(ctx context.Context, lineItemID string) (int64, error) {
	var cost int64
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		target, err := uc.lineItemRepo.FindByID(ctx, lineItemID)
		if err != nil {
			return err
		}
		lineItems, err := uc.lineItemRepo.ListByPurchaseOrder(ctx, target.PurchaseOrderID)
		if err != nil {
			return err
		}
		cost, err = AllocateLandedUnitCost(lineItems, lineItemID)
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("getLandedUnitCost: %w", err)
	}
	return cost, nil
}
