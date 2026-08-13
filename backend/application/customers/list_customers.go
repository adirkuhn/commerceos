package customers

import (
	"context"
	"fmt"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListCustomersUseCase struct {
	db           *gorm.DB
	customerRepo customersDomain.CustomerRepository
}

func NewListCustomersUseCase(db *gorm.DB, customerRepo customersDomain.CustomerRepository) *ListCustomersUseCase {
	return &ListCustomersUseCase{db: db, customerRepo: customerRepo}
}

func (uc *ListCustomersUseCase) Execute(ctx context.Context) ([]customersDomain.Customer, error) {
	var out []customersDomain.Customer
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		out, err = uc.customerRepo.List(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listCustomers: %w", err)
	}
	return out, nil
}
