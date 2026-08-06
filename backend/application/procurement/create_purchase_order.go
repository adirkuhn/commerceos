package procurement

import (
	"context"
	"fmt"
	"time"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreatePurchaseOrderInput struct {
	// SupplierID is nil when uploading a document without picking a
	// supplier up front (see CreatePurchaseOrderFromDocumentUseCase) --
	// the PO starts life as needs_review until one is linked.
	SupplierID *string
	// WarehouseID is nil until the first time stock is received against
	// this PO -- see domain/procurement's PurchaseOrder.WarehouseID doc.
	WarehouseID  *string
	CurrencyCode string
	ExpectedDate *time.Time
	Notes        string
}

type CreatePurchaseOrderUseCase struct {
	db     *gorm.DB
	poRepo procurementDomain.PurchaseOrderRepository
}

func NewCreatePurchaseOrderUseCase(db *gorm.DB, poRepo procurementDomain.PurchaseOrderRepository) *CreatePurchaseOrderUseCase {
	return &CreatePurchaseOrderUseCase{db: db, poRepo: poRepo}
}

func (uc *CreatePurchaseOrderUseCase) Execute(ctx context.Context, in CreatePurchaseOrderInput) (*procurementDomain.PurchaseOrder, error) {
	var po procurementDomain.PurchaseOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		status := procurementDomain.PurchaseOrderStatusDraft
		if in.SupplierID == nil {
			status = procurementDomain.PurchaseOrderStatusNeedsReview
		}
		po = procurementDomain.PurchaseOrder{
			TenantID: tenant.IDFromContext(ctx), SupplierID: in.SupplierID, WarehouseID: in.WarehouseID,
			Status: status, CurrencyCode: in.CurrencyCode,
			ExpectedDate: in.ExpectedDate, Notes: in.Notes, CreatedAt: now, UpdatedAt: now,
		}
		return uc.poRepo.Create(ctx, &po)
	})
	if err != nil {
		return nil, fmt.Errorf("createPurchaseOrder: %w", err)
	}
	return &po, nil
}
