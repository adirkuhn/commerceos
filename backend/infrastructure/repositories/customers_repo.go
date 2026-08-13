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

// ── GORM model ───────────────────────────────────────────────────────────────

type customerModel struct {
	ID            string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID      string `gorm:"not null;type:uuid"`
	WooCustomerID *int64

	Email     string
	FirstName string
	LastName  string
	Username  string
	Role      string

	BillingFirstName  string
	BillingLastName   string
	BillingCompany    string
	BillingAddress1   string `gorm:"column:billing_address_1"`
	BillingAddress2   string `gorm:"column:billing_address_2"`
	BillingCity       string
	BillingState      string
	BillingPostcode   string
	BillingCountry    string
	BillingEmail      string
	BillingPhone      string
	ShippingFirstName string
	ShippingLastName  string
	ShippingCompany   string
	ShippingAddress1  string `gorm:"column:shipping_address_1"`
	ShippingAddress2  string `gorm:"column:shipping_address_2"`
	ShippingCity      string
	ShippingState     string
	ShippingPostcode  string
	ShippingCountry   string
	ShippingPhone     string

	IsPayingCustomer bool
	AvatarURL        string `gorm:"column:avatar_url"`

	DateCreated  time.Time
	DateModified time.Time
}

func (customerModel) TableName() string { return "customers" }

func customerModelFromDomain(c *customersDomain.Customer) customerModel {
	bFirst, bLast, bCompany, bAddr1, bAddr2, bCity, bState, bPostcode, bCountry, bEmail, bPhone := customerAddressFields(c.Billing)
	sFirst, sLast, sCompany, sAddr1, sAddr2, sCity, sState, sPostcode, sCountry, _, sPhone := customerAddressFields(c.Shipping)
	return customerModel{
		ID: c.ID, TenantID: c.TenantID, WooCustomerID: c.WooCustomerID,
		Email: c.Email, FirstName: c.FirstName, LastName: c.LastName, Username: c.Username, Role: c.Role,
		BillingFirstName: bFirst, BillingLastName: bLast, BillingCompany: bCompany,
		BillingAddress1: bAddr1, BillingAddress2: bAddr2, BillingCity: bCity, BillingState: bState,
		BillingPostcode: bPostcode, BillingCountry: bCountry, BillingEmail: bEmail, BillingPhone: bPhone,
		ShippingFirstName: sFirst, ShippingLastName: sLast, ShippingCompany: sCompany,
		ShippingAddress1: sAddr1, ShippingAddress2: sAddr2, ShippingCity: sCity, ShippingState: sState,
		ShippingPostcode: sPostcode, ShippingCountry: sCountry, ShippingPhone: sPhone,
		IsPayingCustomer: c.IsPayingCustomer, AvatarURL: c.AvatarURL,
		DateCreated: c.DateCreated, DateModified: c.DateModified,
	}
}

func customerAddressFields(a customersDomain.Address) (firstName, lastName, company, address1, address2, city, state, postcode, country, email, phone string) {
	return a.FirstName, a.LastName, a.Company, a.Address1, a.Address2, a.City, a.State, a.Postcode, a.Country, a.Email, a.Phone
}

func toCustomerDomain(m customerModel) *customersDomain.Customer {
	return &customersDomain.Customer{
		ID: m.ID, TenantID: m.TenantID, WooCustomerID: m.WooCustomerID,
		Email: m.Email, FirstName: m.FirstName, LastName: m.LastName, Username: m.Username, Role: m.Role,
		Billing: customersDomain.Address{
			FirstName: m.BillingFirstName, LastName: m.BillingLastName, Company: m.BillingCompany,
			Address1: m.BillingAddress1, Address2: m.BillingAddress2, City: m.BillingCity, State: m.BillingState,
			Postcode: m.BillingPostcode, Country: m.BillingCountry, Email: m.BillingEmail, Phone: m.BillingPhone,
		},
		Shipping: customersDomain.Address{
			FirstName: m.ShippingFirstName, LastName: m.ShippingLastName, Company: m.ShippingCompany,
			Address1: m.ShippingAddress1, Address2: m.ShippingAddress2, City: m.ShippingCity, State: m.ShippingState,
			Postcode: m.ShippingPostcode, Country: m.ShippingCountry, Phone: m.ShippingPhone,
		},
		IsPayingCustomer: m.IsPayingCustomer, AvatarURL: m.AvatarURL,
		DateCreated: m.DateCreated, DateModified: m.DateModified,
	}
}

// ── CustomerRepo ─────────────────────────────────────────────────────────────

type CustomerRepo struct{ db *gorm.DB }

func NewCustomerRepo(db *gorm.DB) *CustomerRepo { return &CustomerRepo{db: db} }

func (r *CustomerRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("customerRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *CustomerRepo) Create(ctx context.Context, c *customersDomain.Customer) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := customerModelFromDomain(c)
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("customerRepo.Create: %w", err)
	}
	c.ID = m.ID
	return nil
}

func (r *CustomerRepo) FindByID(ctx context.Context, id string) (*customersDomain.Customer, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m customerModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, customersDomain.ErrCustomerNotFound
		}
		return nil, fmt.Errorf("customerRepo.FindByID: %w", err)
	}
	return toCustomerDomain(m), nil
}

func (r *CustomerRepo) List(ctx context.Context) ([]customersDomain.Customer, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []customerModel
	if err := tx.Order("date_created desc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("customerRepo.List: %w", err)
	}
	out := make([]customersDomain.Customer, len(rows))
	for i, row := range rows {
		out[i] = *toCustomerDomain(row)
	}
	return out, nil
}

func (r *CustomerRepo) Update(ctx context.Context, c *customersDomain.Customer) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := customerModelFromDomain(c)
	// A map, not the struct itself -- GORM's struct-based Updates silently
	// skips zero-value fields (false, "", nil), which would make it
	// impossible to e.g. clear IsPayingCustomer back to false or blank out
	// an address field.
	values := map[string]any{
		"woo_customer_id": m.WooCustomerID, "email": m.Email, "first_name": m.FirstName, "last_name": m.LastName,
		"username": m.Username, "role": m.Role,
		"billing_first_name": m.BillingFirstName, "billing_last_name": m.BillingLastName, "billing_company": m.BillingCompany,
		"billing_address_1": m.BillingAddress1, "billing_address_2": m.BillingAddress2, "billing_city": m.BillingCity,
		"billing_state": m.BillingState, "billing_postcode": m.BillingPostcode, "billing_country": m.BillingCountry,
		"billing_email": m.BillingEmail, "billing_phone": m.BillingPhone,
		"shipping_first_name": m.ShippingFirstName, "shipping_last_name": m.ShippingLastName, "shipping_company": m.ShippingCompany,
		"shipping_address_1": m.ShippingAddress1, "shipping_address_2": m.ShippingAddress2, "shipping_city": m.ShippingCity,
		"shipping_state": m.ShippingState, "shipping_postcode": m.ShippingPostcode, "shipping_country": m.ShippingCountry,
		"shipping_phone":     m.ShippingPhone,
		"is_paying_customer": m.IsPayingCustomer, "avatar_url": m.AvatarURL, "date_modified": m.DateModified,
	}
	if err := tx.Model(&customerModel{}).Where("id = ?", c.ID).Updates(values).Error; err != nil {
		return fmt.Errorf("customerRepo.Update: %w", err)
	}
	return nil
}

func (r *CustomerRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&customerModel{}).Error; err != nil {
		return fmt.Errorf("customerRepo.Delete: %w", err)
	}
	return nil
}
