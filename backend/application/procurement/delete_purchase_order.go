package procurement

import (
	"context"
	"fmt"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeletePurchaseOrderUseCase struct {
	db     *gorm.DB
	poRepo procurementDomain.PurchaseOrderRepository
}

func NewDeletePurchaseOrderUseCase(db *gorm.DB, poRepo procurementDomain.PurchaseOrderRepository) *DeletePurchaseOrderUseCase {
	return &DeletePurchaseOrderUseCase{db: db, poRepo: poRepo}
}

func (uc *DeletePurchaseOrderUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.poRepo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.poRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deletePurchaseOrder: %w", err)
	}
	return nil
}
