package procurement

import (
	"context"
	"fmt"

	pmsapp "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

type CreateProductAndLinkLineItemInput struct {
	LineItemID      string
	ProductName     string
	SKUCode         string
	Barcode         string
	PriceMinorUnits int64
	CurrencyCode    string
}

// Narrow interfaces at point of use (ISP) so this use case can be unit
// tested without a real DB -- *pmsapp.CreateProductUseCase etc. and
// *LinkLineItemToSKUUseCase all satisfy these structurally, no adapter
// needed in production wiring.
type createProductForLinkingUC interface {
	Execute(ctx context.Context, in pmsapp.CreateProductInput) (*pmsDomain.Product, error)
}
type createVariantForLinkingUC interface {
	Execute(ctx context.Context, in pmsapp.CreateVariantInput) (*pmsDomain.Variant, error)
}
type createSKUForLinkingUC interface {
	Execute(ctx context.Context, in pmsapp.CreateSKUInput) (*pmsDomain.SKU, error)
}
type linkLineItemForCreateUC interface {
	Execute(ctx context.Context, in LinkLineItemToSKUInput) (*procurementDomain.PurchaseOrderLineItem, error)
}

// CreateProductAndLinkLineItemUseCase is the review UI's "product doesn't
// exist, create it" action: creates a minimal product/variant/SKU (mirrors
// the frontend's own "create product with variant/SKU in one submit" flow --
// see Products.tsx -- as three sequential, independently-transactional use
// case calls rather than one nested transaction; tenant.WithTenant always
// opens a fresh top-level transaction against the base *gorm.DB, so calling
// it from inside another already-open WithTenant block would run two
// unrelated transactions concurrently instead of nesting), then links the
// line item to the new SKU via LinkLineItemToSKUUseCase (which also writes
// the SupplierProductAlias and reverts the PO out of needs_review).
type CreateProductAndLinkLineItemUseCase struct {
	createProductUC createProductForLinkingUC
	createVariantUC createVariantForLinkingUC
	createSKUUC     createSKUForLinkingUC
	linkUC          linkLineItemForCreateUC
}

func NewCreateProductAndLinkLineItemUseCase(
	createProductUC createProductForLinkingUC,
	createVariantUC createVariantForLinkingUC,
	createSKUUC createSKUForLinkingUC,
	linkUC linkLineItemForCreateUC,
) *CreateProductAndLinkLineItemUseCase {
	return &CreateProductAndLinkLineItemUseCase{
		createProductUC: createProductUC, createVariantUC: createVariantUC, createSKUUC: createSKUUC, linkUC: linkUC,
	}
}

func (uc *CreateProductAndLinkLineItemUseCase) Execute(ctx context.Context, in CreateProductAndLinkLineItemInput) (*procurementDomain.PurchaseOrderLineItem, error) {
	product, err := uc.createProductUC.Execute(ctx, pmsapp.CreateProductInput{
		Name: in.ProductName, Status: pmsDomain.ProductStatusActive,
	})
	if err != nil {
		return nil, fmt.Errorf("createProductAndLinkLineItem: %w", err)
	}

	variant, err := uc.createVariantUC.Execute(ctx, pmsapp.CreateVariantInput{
		ProductID: product.ID, Name: "Default",
	})
	if err != nil {
		return nil, fmt.Errorf("createProductAndLinkLineItem: %w", err)
	}

	sku, err := uc.createSKUUC.Execute(ctx, pmsapp.CreateSKUInput{
		VariantID: variant.ID, SKUCode: in.SKUCode, Barcode: in.Barcode,
		PriceMinorUnits: in.PriceMinorUnits, CurrencyCode: in.CurrencyCode, Status: pmsDomain.SKUStatusActive,
	})
	if err != nil {
		return nil, fmt.Errorf("createProductAndLinkLineItem: %w", err)
	}

	li, err := uc.linkUC.Execute(ctx, LinkLineItemToSKUInput{LineItemID: in.LineItemID, SKUID: sku.ID})
	if err != nil {
		return nil, fmt.Errorf("createProductAndLinkLineItem: %w", err)
	}
	return li, nil
}
