package procurement

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeletePurchaseOrderUseCase struct {
	db           *gorm.DB
	poRepo       procurementDomain.PurchaseOrderRepository
	documentRepo procurementDomain.PurchaseOrderDocumentRepository
	outbox       eventrecord.Enqueuer
}

func NewDeletePurchaseOrderUseCase(
	db *gorm.DB,
	poRepo procurementDomain.PurchaseOrderRepository,
	documentRepo procurementDomain.PurchaseOrderDocumentRepository,
	outbox eventrecord.Enqueuer,
) *DeletePurchaseOrderUseCase {
	return &DeletePurchaseOrderUseCase{db: db, poRepo: poRepo, documentRepo: documentRepo, outbox: outbox}
}

// Execute deletes the PO, then repairs any *other* PO that was pointing at
// it as a duplicate -- the FK (ON DELETE SET NULL) clears
// DuplicateOfPurchaseOrderID automatically, but IsPotentialDuplicate would
// otherwise stay stuck true with a dangling reference. Each dependent is
// re-pointed at another surviving PO that still shares one of its
// documents' content hash, or has its flag cleared entirely if none remain.
func (uc *DeletePurchaseOrderUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.poRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		dependents, err := uc.poRepo.FindDuplicatesOf(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.poRepo.Delete(ctx, id); err != nil {
			return err
		}
		if err := eventrecord.Record(ctx, uc.outbox, events.PurchaseOrderDeleted, id, eventrecord.Data{
			"supplier_id": existing.SupplierID, "status": string(existing.Status),
		}); err != nil {
			return err
		}
		for _, dep := range dependents {
			if err := uc.recomputeDuplicateStatus(ctx, dep.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("deletePurchaseOrder: %w", err)
	}
	return nil
}

func (uc *DeletePurchaseOrderUseCase) recomputeDuplicateStatus(ctx context.Context, purchaseOrderID string) error {
	docs, err := uc.documentRepo.ListByPurchaseOrder(ctx, purchaseOrderID)
	if err != nil {
		return err
	}
	for _, doc := range docs {
		if doc.ContentHash == "" {
			continue
		}
		candidates, err := uc.documentRepo.FindDuplicateCandidates(ctx, doc.ContentHash, purchaseOrderID)
		if err != nil {
			return err
		}
		for _, c := range candidates {
			if c.PurchaseOrderID != purchaseOrderID {
				return uc.poRepo.MarkPotentialDuplicate(ctx, purchaseOrderID, c.PurchaseOrderID)
			}
		}
	}
	return uc.poRepo.ClearPotentialDuplicate(ctx, purchaseOrderID)
}
