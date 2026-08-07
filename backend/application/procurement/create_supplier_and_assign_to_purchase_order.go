package procurement

import (
	"context"
	"errors"
	"fmt"
	"time"

	suppliersapp "github.com/adirkuhn/commerceos/application/suppliers"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateSupplierAndAssignToPurchaseOrderInput struct {
	PurchaseOrderID string
	Name            string
}

// Narrow interfaces at point of use (ISP) so this composition is unit
// testable without a real DB -- *suppliersapp.CreateSupplierUseCase and
// *repositories.SupplierRepo satisfy these structurally, no adapter needed
// in production wiring.
type createSupplierForAssignUC interface {
	Execute(ctx context.Context, in suppliersapp.CreateSupplierInput) (*suppliersDomain.Supplier, error)
}

type findSupplierByNameForAssignUC interface {
	FindByName(ctx context.Context, name string) (*suppliersDomain.Supplier, error)
}

// CreateSupplierAndAssignToPurchaseOrderUseCase is the review UI's "this
// supplier doesn't exist yet, create it" action for a PO uploaded without a
// supplier selected (see CreatePurchaseOrderFromDocumentUseCase) -- creates
// a minimal active supplier, assigns it to the PO, and reverts the PO out
// of needs_review if every line item is also already linked. Mirrors
// CreateProductAndLinkLineItemUseCase's shape for the equivalent
// missing-product case.
type CreateSupplierAndAssignToPurchaseOrderUseCase struct {
	db                 *gorm.DB
	findSupplierByName findSupplierByNameForAssignUC
	createSupplier     createSupplierForAssignUC
	poRepo             procurementDomain.PurchaseOrderRepository
	lineItemRepo       procurementDomain.PurchaseOrderLineItemRepository
}

func NewCreateSupplierAndAssignToPurchaseOrderUseCase(
	db *gorm.DB,
	findSupplierByName findSupplierByNameForAssignUC,
	createSupplier createSupplierForAssignUC,
	poRepo procurementDomain.PurchaseOrderRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
) *CreateSupplierAndAssignToPurchaseOrderUseCase {
	return &CreateSupplierAndAssignToPurchaseOrderUseCase{
		db: db, findSupplierByName: findSupplierByName, createSupplier: createSupplier, poRepo: poRepo, lineItemRepo: lineItemRepo,
	}
}

func (uc *CreateSupplierAndAssignToPurchaseOrderUseCase) Execute(ctx context.Context, in CreateSupplierAndAssignToPurchaseOrderInput) (*procurementDomain.PurchaseOrder, error) {
	// A name typed here (or suggested by AI parsing) may already match an
	// existing supplier -- reuse it instead of minting a duplicate row.
	var supplier *suppliersDomain.Supplier
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		found, err := uc.findSupplierByName.FindByName(ctx, in.Name)
		if err != nil && !errors.Is(err, suppliersDomain.ErrSupplierNotFound) {
			return err
		}
		supplier = found
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("createSupplierAndAssignToPurchaseOrder: %w", err)
	}

	if supplier == nil {
		supplier, err = uc.createSupplier.Execute(ctx, suppliersapp.CreateSupplierInput{
			Name: in.Name, Status: suppliersDomain.SupplierStatusActive,
		})
		if err != nil {
			return nil, fmt.Errorf("createSupplierAndAssignToPurchaseOrder: %w", err)
		}
	}

	var po procurementDomain.PurchaseOrder
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.poRepo.FindByID(ctx, in.PurchaseOrderID)
		if err != nil {
			return err
		}
		existing.SupplierID = &supplier.ID
		existing.UpdatedAt = time.Now()
		if err := uc.poRepo.Update(ctx, existing); err != nil {
			return err
		}
		po = *existing
		if err := revertPurchaseOrderFromReviewIfFullyResolved(ctx, uc.poRepo, uc.lineItemRepo, &po); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("createSupplierAndAssignToPurchaseOrder: %w", err)
	}
	return &po, nil
}
