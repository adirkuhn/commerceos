package suppliers

import (
	"context"
	"fmt"

	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListSuppliersUseCase struct {
	db           *gorm.DB
	supplierRepo suppliersDomain.SupplierRepository
}

func NewListSuppliersUseCase(db *gorm.DB, supplierRepo suppliersDomain.SupplierRepository) *ListSuppliersUseCase {
	return &ListSuppliersUseCase{db: db, supplierRepo: supplierRepo}
}

func (uc *ListSuppliersUseCase) Execute(ctx context.Context) ([]suppliersDomain.Supplier, error) {
	var suppliers []suppliersDomain.Supplier
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		suppliers, err = uc.supplierRepo.List(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listSuppliers: %w", err)
	}
	return suppliers, nil
}
