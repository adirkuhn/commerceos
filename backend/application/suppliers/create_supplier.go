package suppliers

import (
	"context"
	"fmt"
	"time"

	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateSupplierInput struct {
	Name         string
	Email        string
	Phone        string
	Country      string
	LeadTimeDays *int
	Status       suppliersDomain.SupplierStatus
	Notes        string
}

type CreateSupplierUseCase struct {
	db           *gorm.DB
	supplierRepo suppliersDomain.SupplierRepository
}

func NewCreateSupplierUseCase(db *gorm.DB, supplierRepo suppliersDomain.SupplierRepository) *CreateSupplierUseCase {
	return &CreateSupplierUseCase{db: db, supplierRepo: supplierRepo}
}

func (uc *CreateSupplierUseCase) Execute(ctx context.Context, in CreateSupplierInput) (*suppliersDomain.Supplier, error) {
	var supplier suppliersDomain.Supplier
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		supplier = suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx),
			Name:     in.Name, Email: in.Email, Phone: in.Phone, Country: in.Country,
			LeadTimeDays: in.LeadTimeDays, Status: in.Status, Notes: in.Notes,
			CreatedAt: now, UpdatedAt: now,
		}
		return uc.supplierRepo.Create(ctx, &supplier)
	})
	if err != nil {
		return nil, fmt.Errorf("createSupplier: %w", err)
	}
	return &supplier, nil
}
