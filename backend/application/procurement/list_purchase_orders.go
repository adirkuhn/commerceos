package procurement

import (
	"context"
	"fmt"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListPurchaseOrdersUseCase struct {
	db           *gorm.DB
	poRepo       procurementDomain.PurchaseOrderRepository
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
}

func NewListPurchaseOrdersUseCase(
	db *gorm.DB,
	poRepo procurementDomain.PurchaseOrderRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
) *ListPurchaseOrdersUseCase {
	return &ListPurchaseOrdersUseCase{db: db, poRepo: poRepo, lineItemRepo: lineItemRepo}
}

func (uc *ListPurchaseOrdersUseCase) Execute(ctx context.Context) ([]procurementDomain.PurchaseOrder, error) {
	var purchaseOrders []procurementDomain.PurchaseOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		purchaseOrders, err = uc.poRepo.List(ctx)
		if err != nil {
			return err
		}

		ids := make([]string, len(purchaseOrders))
		for i, po := range purchaseOrders {
			ids[i] = po.ID
		}
		totals, err := uc.lineItemRepo.SumByPurchaseOrders(ctx, ids)
		if err != nil {
			return err
		}
		for i := range purchaseOrders {
			purchaseOrders[i].TotalMinorUnits = totals[purchaseOrders[i].ID]
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listPurchaseOrders: %w", err)
	}
	return purchaseOrders, nil
}
