package suppliers

import (
	"context"
	"fmt"

	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteSupplierUseCase struct {
	db           *gorm.DB
	supplierRepo suppliersDomain.SupplierRepository
}

func NewDeleteSupplierUseCase(db *gorm.DB, supplierRepo suppliersDomain.SupplierRepository) *DeleteSupplierUseCase {
	return &DeleteSupplierUseCase{db: db, supplierRepo: supplierRepo}
}

func (uc *DeleteSupplierUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.supplierRepo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.supplierRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteSupplier: %w", err)
	}
	return nil
}
