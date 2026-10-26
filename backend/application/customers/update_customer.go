package customers

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"time"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateCustomerInput struct {
	ID            string
	WooCustomerID *int64
	// Type/CompanyName/ClientNumber/KVKNumber/VATNumber: nil keeps the
	// current value.
	Type             *customersDomain.Type
	CompanyName      *string
	ClientNumber     *string
	KVKNumber        *string
	VATNumber        *string
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

type UpdateCustomerUseCase struct {
	db           *gorm.DB
	customerRepo customersDomain.CustomerRepository
	outbox       eventrecord.Enqueuer
}

func NewUpdateCustomerUseCase(db *gorm.DB, customerRepo customersDomain.CustomerRepository, outbox eventrecord.Enqueuer) *UpdateCustomerUseCase {
	return &UpdateCustomerUseCase{db: db, customerRepo: customerRepo, outbox: outbox}
}

func (uc *UpdateCustomerUseCase) Execute(ctx context.Context, in UpdateCustomerInput) (*customersDomain.Customer, error) {
	var customer customersDomain.Customer
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.customerRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		before := *existing
		existing.WooCustomerID = in.WooCustomerID
		existing.Email = in.Email
		existing.FirstName = in.FirstName
		existing.LastName = in.LastName
		existing.Username = in.Username
		existing.Role = in.Role
		existing.Billing = in.Billing
		existing.Shipping = in.Shipping
		existing.IsPayingCustomer = in.IsPayingCustomer
		existing.AvatarURL = in.AvatarURL
		existing.CustomerNote = in.CustomerNote
		applyCompanyFields(existing, in)
		existing.DateModified = time.Now()
		if err := existing.Validate(); err != nil {
			return err
		}
		if err := uc.customerRepo.Update(ctx, existing); err != nil {
			return err
		}
		customer = *existing
		return recordCustomerUpdated(ctx, uc.outbox, before, customer)
	})
	if err != nil {
		return nil, fmt.Errorf("updateCustomer: %w", err)
	}
	return &customer, nil
}

func applyCompanyFields(c *customersDomain.Customer, in UpdateCustomerInput) {
	if in.Type != nil {
		c.Type = *in.Type
	}
	setIfSent(&c.CompanyName, in.CompanyName)
	setIfSent(&c.ClientNumber, in.ClientNumber)
	setIfSent(&c.KVKNumber, in.KVKNumber)
	setIfSent(&c.VATNumber, in.VATNumber)
}

func setIfSent(field *string, value *string) {
	if value != nil {
		*field = *value
	}
}
