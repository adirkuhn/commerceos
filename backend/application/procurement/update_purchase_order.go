package procurement

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdatePurchaseOrderInput struct {
	ID           string
	SupplierID   *string
	WarehouseID  *string
	Status       procurementDomain.PurchaseOrderStatus
	CurrencyCode string
	ExpectedDate *time.Time
	Notes        string
}

type UpdatePurchaseOrderUseCase struct {
	db           *gorm.DB
	poRepo       procurementDomain.PurchaseOrderRepository
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
	aliasRepo    procurementDomain.SupplierProductAliasRepository
	outbox       eventrecord.Enqueuer
}

func NewUpdatePurchaseOrderUseCase(
	db *gorm.DB,
	poRepo procurementDomain.PurchaseOrderRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	aliasRepo procurementDomain.SupplierProductAliasRepository,
	outbox eventrecord.Enqueuer,
) *UpdatePurchaseOrderUseCase {
	return &UpdatePurchaseOrderUseCase{db: db, poRepo: poRepo, lineItemRepo: lineItemRepo, aliasRepo: aliasRepo, outbox: outbox}
}

func (uc *UpdatePurchaseOrderUseCase) Execute(ctx context.Context, in UpdatePurchaseOrderInput) (*procurementDomain.PurchaseOrder, error) {
	var po procurementDomain.PurchaseOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.poRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		// Picking a different supplier invalidates any not-yet-reviewed
		// alias guess made under the old one -- redo those against the new
		// supplier's own learned aliases rather than leaving a stale SKU.
		before := *existing
		supplierChanged := in.SupplierID != nil && (existing.SupplierID == nil || *existing.SupplierID != *in.SupplierID)
		existing.SupplierID = in.SupplierID
		existing.WarehouseID = in.WarehouseID
		existing.Status = in.Status
		existing.CurrencyCode = in.CurrencyCode
		existing.ExpectedDate = in.ExpectedDate
		existing.Notes = in.Notes
		existing.UpdatedAt = time.Now()
		if err := uc.poRepo.Update(ctx, existing); err != nil {
			return err
		}
		po = *existing
		if err := eventrecord.Record(ctx, uc.outbox, events.PurchaseOrderUpdated, po.ID, purchaseOrderChanges(before, po)); err != nil {
			return err
		}
		if supplierChanged {
			if _, err := resyncAliasMatchedLineItems(ctx, uc.lineItemRepo, uc.aliasRepo, uc.outbox, po.ID, *in.SupplierID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updatePurchaseOrder: %w", err)
	}
	return &po, nil
}
