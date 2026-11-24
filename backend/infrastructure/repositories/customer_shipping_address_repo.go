package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type customerShippingAddressModel struct {
	ID         string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID   string `gorm:"not null;type:uuid"`
	CustomerID string `gorm:"column:customer_id;not null;type:uuid"`
	Label      string
	FirstName  string `gorm:"column:first_name"`
	LastName   string `gorm:"column:last_name"`
	Company    string
	Address1   string `gorm:"column:address_1"`
	Address2   string `gorm:"column:address_2"`
	City       string
	State      string
	Postcode   string
	Country    string
	Phone      string
	IsDefault  bool `gorm:"column:is_default"`
	Position   int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (customerShippingAddressModel) TableName() string { return "customer_shipping_addresses" }

func shippingAddressModelFromDomain(a *customersDomain.ShippingAddress) customerShippingAddressModel {
	ad := a.Address
	return customerShippingAddressModel{
		ID: a.ID, TenantID: a.TenantID, CustomerID: a.CustomerID, Label: a.Label,
		FirstName: ad.FirstName, LastName: ad.LastName, Company: ad.Company, Address1: ad.Address1, Address2: ad.Address2,
		City: ad.City, State: ad.State, Postcode: ad.Postcode, Country: ad.Country, Phone: ad.Phone,
		IsDefault: a.IsDefault, Position: a.Position, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}

func toShippingAddressDomain(m customerShippingAddressModel) customersDomain.ShippingAddress {
	return customersDomain.ShippingAddress{
		ID: m.ID, TenantID: m.TenantID, CustomerID: m.CustomerID, Label: m.Label,
		Address: customersDomain.Address{
			FirstName: m.FirstName, LastName: m.LastName, Company: m.Company, Address1: m.Address1, Address2: m.Address2,
			City: m.City, State: m.State, Postcode: m.Postcode, Country: m.Country, Phone: m.Phone,
		},
		IsDefault: m.IsDefault, Position: m.Position, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

// CustomerShippingAddressRepo stores customers' shipping address books.
type CustomerShippingAddressRepo struct{ db *gorm.DB }

func NewCustomerShippingAddressRepo(db *gorm.DB) *CustomerShippingAddressRepo {
	return &CustomerShippingAddressRepo{db: db}
}

func (r *CustomerShippingAddressRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("customerShippingAddressRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *CustomerShippingAddressRepo) ListByCustomer(ctx context.Context, customerID string) (customersDomain.ShippingBook, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []customerShippingAddressModel
	if err := tx.Where("customer_id = ?", customerID).Order("position, created_at, id").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("customerShippingAddressRepo.ListByCustomer: %w", err)
	}
	out := make(customersDomain.ShippingBook, len(rows))
	for i, m := range rows {
		out[i] = toShippingAddressDomain(m)
	}
	return out, nil
}

func (r *CustomerShippingAddressRepo) Create(ctx context.Context, a *customersDomain.ShippingAddress) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := shippingAddressModelFromDomain(a)
	m.ID = ""
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("customerShippingAddressRepo.Create: %w", err)
	}
	a.ID = m.ID
	return nil
}

// Update saves the address fields and label; the default flag moves only
// through SetDefault.
func (r *CustomerShippingAddressRepo) Update(ctx context.Context, a *customersDomain.ShippingAddress) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := shippingAddressModelFromDomain(a)
	res := tx.Model(&customerShippingAddressModel{}).Where("id = ? AND customer_id = ?", a.ID, a.CustomerID).Updates(map[string]any{
		"label": m.Label, "first_name": m.FirstName, "last_name": m.LastName, "company": m.Company,
		"address_1": m.Address1, "address_2": m.Address2, "city": m.City, "state": m.State,
		"postcode": m.Postcode, "country": m.Country, "phone": m.Phone, "updated_at": m.UpdatedAt,
	})
	if res.Error != nil {
		return fmt.Errorf("customerShippingAddressRepo.Update: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return customersDomain.ErrShippingAddressNotFound
	}
	return nil
}

func (r *CustomerShippingAddressRepo) Delete(ctx context.Context, customerID, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Where("id = ? AND customer_id = ?", id, customerID).Delete(&customerShippingAddressModel{})
	if res.Error != nil {
		return fmt.Errorf("customerShippingAddressRepo.Delete: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return customersDomain.ErrShippingAddressNotFound
	}
	return nil
}

// SetDefault clears the old default before setting the new one: the
// one-default unique index checks each statement.
func (r *CustomerShippingAddressRepo) SetDefault(ctx context.Context, customerID, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	var exists int64
	if err := tx.Model(&customerShippingAddressModel{}).Where("id = ? AND customer_id = ?", id, customerID).Count(&exists).Error; err != nil {
		return fmt.Errorf("customerShippingAddressRepo.SetDefault: %w", err)
	}
	if exists == 0 {
		return customersDomain.ErrShippingAddressNotFound
	}
	if err := tx.Model(&customerShippingAddressModel{}).Where("customer_id = ? AND is_default AND id <> ?", customerID, id).
		Update("is_default", false).Error; err != nil {
		return fmt.Errorf("customerShippingAddressRepo.SetDefault: %w", err)
	}
	if err := tx.Model(&customerShippingAddressModel{}).Where("id = ?", id).Update("is_default", true).Error; err != nil {
		return fmt.Errorf("customerShippingAddressRepo.SetDefault: %w", err)
	}
	return nil
}
