package procurement

import (
	"context"
	"fmt"
	"time"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type LinkLineItemToSKUInput struct {
	LineItemID string
	SKUID      string
}

// LinkLineItemToSKUUseCase is the review UI's "link to the correct product"
// action: sets the line item's SKUID, writes a SupplierProductAlias keyed on
// the item's raw text so the same mismatch auto-links next time (per the
// plan's "keep this in mind for the next PO" requirement), and reverts the
// parent PO out of needs_review once every line item is linked.
type LinkLineItemToSKUUseCase struct {
	db           *gorm.DB
	poRepo       procurementDomain.PurchaseOrderRepository
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
	aliasRepo    procurementDomain.SupplierProductAliasRepository
}

func NewLinkLineItemToSKUUseCase(
	db *gorm.DB,
	poRepo procurementDomain.PurchaseOrderRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	aliasRepo procurementDomain.SupplierProductAliasRepository,
) *LinkLineItemToSKUUseCase {
	return &LinkLineItemToSKUUseCase{db: db, poRepo: poRepo, lineItemRepo: lineItemRepo, aliasRepo: aliasRepo}
}

func (uc *LinkLineItemToSKUUseCase) Execute(ctx context.Context, in LinkLineItemToSKUInput) (*procurementDomain.PurchaseOrderLineItem, error) {
	var li procurementDomain.PurchaseOrderLineItem
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.lineItemRepo.FindByID(ctx, in.LineItemID)
		if err != nil {
			return err
		}
		po, err := uc.poRepo.FindByID(ctx, existing.PurchaseOrderID)
		if err != nil {
			return err
		}

		now := time.Now()
		existing.SKUID = &in.SKUID
		// Explicit human confirmation -- overrides any unreviewed alias
		// guess this item may have carried in with.
		existing.SKUAutoLinkedFromAlias = false
		existing.UpdatedAt = now
		if err := uc.lineItemRepo.Update(ctx, existing); err != nil {
			return err
		}
		li = *existing

		// Can't attribute a learned correction to an unknown supplier -- if
		// the PO itself is still missing one, skip writing the alias (the
		// line item is still linked either way).
		rawKey := NormalizeRawKey(existing.RawSKU, existing.RawDescription)
		if rawKey != "" && po.SupplierID != nil {
			alias := procurementDomain.SupplierProductAlias{
				TenantID: tenant.IDFromContext(ctx), SupplierID: *po.SupplierID, RawKey: rawKey, SKUID: in.SKUID,
				CreatedAt: now, UpdatedAt: now,
			}
			if err := uc.aliasRepo.Upsert(ctx, &alias); err != nil {
				return err
			}
		}

		return revertPurchaseOrderFromReviewIfFullyResolved(ctx, uc.poRepo, uc.lineItemRepo, po)
	})
	if err != nil {
		return nil, fmt.Errorf("linkLineItemToSKU: %w", err)
	}
	return &li, nil
}

// revertPurchaseOrderFromReviewIfFullyResolved reverts a needs_review PO
// back to draft once it has a supplier AND every line item has a SKUID --
// shared by LinkLineItemToSKUUseCase, CreateProductAndLinkLineItemUseCase,
// and CreateSupplierAndAssignToPurchaseOrderUseCase, the three review-UI
// actions that can resolve the last open issue on a PO.
func revertPurchaseOrderFromReviewIfFullyResolved(
	ctx context.Context,
	poRepo procurementDomain.PurchaseOrderRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	po *procurementDomain.PurchaseOrder,
) error {
	if po.Status != procurementDomain.PurchaseOrderStatusNeedsReview {
		return nil
	}
	if po.SupplierID == nil {
		return nil
	}
	items, err := lineItemRepo.ListByPurchaseOrder(ctx, po.ID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.SKUID == nil {
			return nil
		}
	}
	po.Status = procurementDomain.PurchaseOrderStatusDraft
	po.UpdatedAt = time.Now()
	return poRepo.Update(ctx, po)
}
