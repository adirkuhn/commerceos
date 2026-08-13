package customers

import (
	"context"
	"fmt"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteCustomerUseCase struct {
	db           *gorm.DB
	customerRepo customersDomain.CustomerRepository
}

func NewDeleteCustomerUseCase(db *gorm.DB, customerRepo customersDomain.CustomerRepository) *DeleteCustomerUseCase {
	return &DeleteCustomerUseCase{db: db, customerRepo: customerRepo}
}

func (uc *DeleteCustomerUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.customerRepo.FindByID(ctx, id); err != nil {
			return err
		}
		return uc.customerRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteCustomer: %w", err)
	}
	return nil
}
