package procurement

import (
	"context"
	"fmt"
	"time"

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
	db     *gorm.DB
	poRepo procurementDomain.PurchaseOrderRepository
}

func NewUpdatePurchaseOrderUseCase(db *gorm.DB, poRepo procurementDomain.PurchaseOrderRepository) *UpdatePurchaseOrderUseCase {
	return &UpdatePurchaseOrderUseCase{db: db, poRepo: poRepo}
}

func (uc *UpdatePurchaseOrderUseCase) Execute(ctx context.Context, in UpdatePurchaseOrderInput) (*procurementDomain.PurchaseOrder, error) {
	var po procurementDomain.PurchaseOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.poRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
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
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updatePurchaseOrder: %w", err)
	}
	return &po, nil
}
