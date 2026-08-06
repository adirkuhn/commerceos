package procurement

import (
	"context"
	"fmt"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type PurchaseOrderDetail struct {
	PurchaseOrder procurementDomain.PurchaseOrder
	LineItems     []procurementDomain.PurchaseOrderLineItem
}

type GetPurchaseOrderUseCase struct {
	db           *gorm.DB
	poRepo       procurementDomain.PurchaseOrderRepository
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
}

func NewGetPurchaseOrderUseCase(
	db *gorm.DB,
	poRepo procurementDomain.PurchaseOrderRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
) *GetPurchaseOrderUseCase {
	return &GetPurchaseOrderUseCase{db: db, poRepo: poRepo, lineItemRepo: lineItemRepo}
}

func (uc *GetPurchaseOrderUseCase) Execute(ctx context.Context, id string) (*PurchaseOrderDetail, error) {
	var detail PurchaseOrderDetail
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		po, err := uc.poRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		detail.PurchaseOrder = *po

		lineItems, err := uc.lineItemRepo.ListByPurchaseOrder(ctx, id)
		if err != nil {
			return err
		}
		detail.LineItems = lineItems
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getPurchaseOrder: %w", err)
	}
	return &detail, nil
}
