package customers

import (
	"context"
	"fmt"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// GetCustomerUseCase reads one customer -- for other modules (quotes, sales
// orders) that reference a customer by id.
type GetCustomerUseCase struct {
	db           *gorm.DB
	customerRepo customersDomain.CustomerRepository
}

func NewGetCustomerUseCase(db *gorm.DB, customerRepo customersDomain.CustomerRepository) *GetCustomerUseCase {
	return &GetCustomerUseCase{db: db, customerRepo: customerRepo}
}

func (uc *GetCustomerUseCase) Execute(ctx context.Context, id string) (*customersDomain.Customer, error) {
	var c *customersDomain.Customer
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		c, err = uc.customerRepo.FindByID(ctx, id)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getCustomer: %w", err)
	}
	return c, nil
}
