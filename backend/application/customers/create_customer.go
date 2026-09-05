package customers

import (
	"context"
	"fmt"
	"time"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type CreateCustomerInput struct {
	WooCustomerID    *int64
	Email            string
	FirstName        string
	LastName         string
	Username         string
	Role             string
	Billing          customersDomain.Address
	Shipping         customersDomain.Address
	IsPayingCustomer bool
	AvatarURL        string
	CustomerNote     string
}

type CreateCustomerUseCase struct {
	db           *gorm.DB
	customerRepo customersDomain.CustomerRepository
}

func NewCreateCustomerUseCase(db *gorm.DB, customerRepo customersDomain.CustomerRepository) *CreateCustomerUseCase {
	return &CreateCustomerUseCase{db: db, customerRepo: customerRepo}
}

func (uc *CreateCustomerUseCase) Execute(ctx context.Context, in CreateCustomerInput) (*customersDomain.Customer, error) {
	var customer customersDomain.Customer
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		customer = customersDomain.Customer{
			TenantID: tenant.IDFromContext(ctx), WooCustomerID: in.WooCustomerID,
			Email: in.Email, FirstName: in.FirstName, LastName: in.LastName,
			Username: in.Username, Role: in.Role, Billing: in.Billing, Shipping: in.Shipping,
			IsPayingCustomer: in.IsPayingCustomer, AvatarURL: in.AvatarURL, CustomerNote: in.CustomerNote,
			DateCreated: now, DateModified: now,
		}
		return uc.customerRepo.Create(ctx, &customer)
	})
	if err != nil {
		return nil, fmt.Errorf("createCustomer: %w", err)
	}
	return &customer, nil
}
