package procurement

import (
	"context"
	"fmt"
	"time"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdatePurchaseOrderLineItemInput struct {
	ID                 string
	SKUID              *string
	RawSKU             string
	RawDescription     string
	Quantity           int64
	UnitCostMinorUnits int64
}

type UpdatePurchaseOrderLineItemUseCase struct {
	db           *gorm.DB
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
}

func NewUpdatePurchaseOrderLineItemUseCase(db *gorm.DB, lineItemRepo procurementDomain.PurchaseOrderLineItemRepository) *UpdatePurchaseOrderLineItemUseCase {
	return &UpdatePurchaseOrderLineItemUseCase{db: db, lineItemRepo: lineItemRepo}
}

func (uc *UpdatePurchaseOrderLineItemUseCase) Execute(ctx context.Context, in UpdatePurchaseOrderLineItemInput) (*procurementDomain.PurchaseOrderLineItem, error) {
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("updatePurchaseOrderLineItem: %w", procurementDomain.ErrInvalidQuantity)
	}

	var li procurementDomain.PurchaseOrderLineItem
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.lineItemRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		existing.SKUID = in.SKUID
		existing.RawSKU = in.RawSKU
		existing.RawDescription = in.RawDescription
		existing.Quantity = in.Quantity
		existing.UnitCostMinorUnits = in.UnitCostMinorUnits
		existing.UpdatedAt = time.Now()
		if err := uc.lineItemRepo.Update(ctx, existing); err != nil {
			return err
		}
		li = *existing
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updatePurchaseOrderLineItem: %w", err)
	}
	return &li, nil
}
