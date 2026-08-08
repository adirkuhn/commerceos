package procurement

import (
	"context"
	"errors"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
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
	skuRepo      pmsDomain.SKURepository
	variantRepo  pmsDomain.VariantRepository
	productRepo  pmsDomain.ProductRepository
}

func NewGetPurchaseOrderUseCase(
	db *gorm.DB,
	poRepo procurementDomain.PurchaseOrderRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	documentRepo procurementDomain.PurchaseOrderDocumentRepository,
	skuRepo pmsDomain.SKURepository,
	variantRepo pmsDomain.VariantRepository,
	productRepo pmsDomain.ProductRepository,
) *GetPurchaseOrderUseCase {
	return &GetPurchaseOrderUseCase{
		db: db, poRepo: poRepo, lineItemRepo: lineItemRepo, documentRepo: documentRepo,
		skuRepo: skuRepo, variantRepo: variantRepo, productRepo: productRepo,
	}
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
		if err := uc.attachProductDisplayNames(ctx, lineItems); err != nil {
			return err
		}
		detail.LineItems = lineItems

		var total int64
		for _, li := range lineItems {
			total += lineItemSignedAmount(li)
		}
		detail.PurchaseOrder.TotalMinorUnits = total

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

// lineItemSignedAmount is quantity*unit_cost_minor_units, negated for
// discount/credit_note rows (see LineItemType.IsReduction) -- the PO-total
// building block shared by GetPurchaseOrderUseCase and (in SQL form)
// PurchaseOrderLineItemRepo.SumByPurchaseOrders.
func lineItemSignedAmount(li procurementDomain.PurchaseOrderLineItem) int64 {
	amount := li.Quantity * li.UnitCostMinorUnits
	if li.LineType.IsReduction() {
		return -amount
	}
	return amount
}

// attachProductDisplayNames fills in ProductName/SKUCode (display-only, see
// PurchaseOrderLineItem's doc comment) for every linked item, in place. Not
// found is tolerated (leaves the pair nil, item still renders by its raw
// text) rather than failing the whole PO fetch -- a SKU a line item points
// at existing at link time doesn't guarantee it still does by the time this
// runs. Any other repository error still propagates.
func (uc *GetPurchaseOrderUseCase) attachProductDisplayNames(ctx context.Context, items []procurementDomain.PurchaseOrderLineItem) error {
	for i := range items {
		if items[i].SKUID == nil {
			continue
		}
		sku, err := uc.skuRepo.FindByID(ctx, *items[i].SKUID)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrSKUNotFound) {
				continue
			}
			return err
		}
		variant, err := uc.variantRepo.FindByID(ctx, sku.VariantID)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrVariantNotFound) {
				continue
			}
			return err
		}
		product, err := uc.productRepo.FindByID(ctx, variant.ProductID)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrProductNotFound) {
				continue
			}
			return err
		}
		items[i].SKUCode = &sku.SKUCode
		items[i].ProductName = &product.Name
	}
	return nil
}
