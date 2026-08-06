package procurement

import (
	"context"
	"fmt"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListPurchaseOrdersUseCase struct {
	db     *gorm.DB
	poRepo procurementDomain.PurchaseOrderRepository
}

func NewListPurchaseOrdersUseCase(db *gorm.DB, poRepo procurementDomain.PurchaseOrderRepository) *ListPurchaseOrdersUseCase {
	return &ListPurchaseOrdersUseCase{db: db, poRepo: poRepo}
}

func (uc *ListPurchaseOrdersUseCase) Execute(ctx context.Context) ([]procurementDomain.PurchaseOrder, error) {
	var purchaseOrders []procurementDomain.PurchaseOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		purchaseOrders, err = uc.poRepo.List(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listPurchaseOrders: %w", err)
	}
	return purchaseOrders, nil
}
