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

type RemovePurchaseOrderLineItemUseCase struct {
	db           *gorm.DB
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
	outbox       eventrecord.Enqueuer
}

func NewRemovePurchaseOrderLineItemUseCase(db *gorm.DB, lineItemRepo procurementDomain.PurchaseOrderLineItemRepository, outbox eventrecord.Enqueuer) *RemovePurchaseOrderLineItemUseCase {
	return &RemovePurchaseOrderLineItemUseCase{db: db, lineItemRepo: lineItemRepo, outbox: outbox}
}

func (uc *RemovePurchaseOrderLineItemUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.lineItemRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.lineItemRepo.Delete(ctx, id); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.PurchaseOrderLineItemRemoved, id, eventrecord.Data{
			"purchase_order_id": existing.PurchaseOrderID, "sku_id": existing.SKUID,
		})
	})
	if err != nil {
		return fmt.Errorf("removePurchaseOrderLineItem: %w", err)
	}
	return nil
}
