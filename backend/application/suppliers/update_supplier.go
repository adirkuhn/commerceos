package suppliers

import (
	"context"
	"fmt"
	"time"

	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateSupplierInput struct {
	ID           string
	Name         string
	Email        string
	Phone        string
	Country      string
	LeadTimeDays *int
	Status       suppliersDomain.SupplierStatus
	Notes        string
}

type UpdateSupplierUseCase struct {
	db           *gorm.DB
	supplierRepo suppliersDomain.SupplierRepository
}

func NewUpdateSupplierUseCase(db *gorm.DB, supplierRepo suppliersDomain.SupplierRepository) *UpdateSupplierUseCase {
	return &UpdateSupplierUseCase{db: db, supplierRepo: supplierRepo}
}

func (uc *UpdateSupplierUseCase) Execute(ctx context.Context, in UpdateSupplierInput) (*suppliersDomain.Supplier, error) {
	var supplier suppliersDomain.Supplier
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.supplierRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		existing.Name = in.Name
		existing.Email = in.Email
		existing.Phone = in.Phone
		existing.Country = in.Country
		existing.LeadTimeDays = in.LeadTimeDays
		existing.Status = in.Status
		existing.Notes = in.Notes
		existing.UpdatedAt = time.Now()
		if err := uc.supplierRepo.Update(ctx, existing); err != nil {
			return err
		}
		supplier = *existing
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateSupplier: %w", err)
	}
	return &supplier, nil
}
