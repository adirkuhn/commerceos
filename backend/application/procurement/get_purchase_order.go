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
	// Documents is every file ever uploaded against this PO (create-time
	// upload plus any later "Line Items > Upload Document" attachments),
	// newest last -- FileURL is already a directly fetchable URL (see
	// pms.FileStorage.Save), so the frontend can link straight to it with
	// no extra endpoint.
	Documents []procurementDomain.PurchaseOrderDocument
}

type GetPurchaseOrderUseCase struct {
	db           *gorm.DB
	poRepo       procurementDomain.PurchaseOrderRepository
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
	documentRepo procurementDomain.PurchaseOrderDocumentRepository
}

func NewGetPurchaseOrderUseCase(
	db *gorm.DB,
	poRepo procurementDomain.PurchaseOrderRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	documentRepo procurementDomain.PurchaseOrderDocumentRepository,
) *GetPurchaseOrderUseCase {
	return &GetPurchaseOrderUseCase{db: db, poRepo: poRepo, lineItemRepo: lineItemRepo, documentRepo: documentRepo}
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

		documents, err := uc.documentRepo.ListByPurchaseOrder(ctx, id)
		if err != nil {
			return err
		}
		detail.Documents = documents
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getPurchaseOrder: %w", err)
	}
	return &detail, nil
}
