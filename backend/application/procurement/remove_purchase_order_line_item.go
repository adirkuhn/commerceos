package procurement

import (
	"context"
	"fmt"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type RemovePurchaseOrderLineItemUseCase struct {
	db           *gorm.DB
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
}

func NewRemovePurchaseOrderLineItemUseCase(db *gorm.DB, lineItemRepo procurementDomain.PurchaseOrderLineItemRepository) *RemovePurchaseOrderLineItemUseCase {
	return &RemovePurchaseOrderLineItemUseCase{db: db, lineItemRepo: lineItemRepo}
}

func (uc *RemovePurchaseOrderLineItemUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.lineItemRepo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.lineItemRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("removePurchaseOrderLineItem: %w", err)
	}
	return nil
}
