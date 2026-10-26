package repositories

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── GORM model ───────────────────────────────────────────────────────────────

type customerModel struct {
	ID            string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID      string `gorm:"not null;type:uuid"`
	WooCustomerID *int64

	CustomerType string `gorm:"column:customer_type;not null;default:private"`
	CompanyName  string `gorm:"column:company_name"`
	ClientNumber string `gorm:"column:client_number"`
	KVKNumber    string `gorm:"column:kvk_number"`
	VATNumber    string `gorm:"column:vat_number"`

	Email     string
	FirstName string
	LastName  string
	PhoneE164 string `gorm:"column:phone_e164"`
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
	CustomerNote     string `gorm:"column:customer_note"`

	DateCreated  time.Time
	DateModified time.Time
}

func (customerModel) TableName() string { return "customers" }

func customerModelFromDomain(c *customersDomain.Customer) customerModel {
	bFirst, bLast, bCompany, bAddr1, bAddr2, bCity, bState, bPostcode, bCountry, bEmail, bPhone := customerAddressFields(c.Billing)
	sFirst, sLast, sCompany, sAddr1, sAddr2, sCity, sState, sPostcode, sCountry, _, sPhone := customerAddressFields(c.Shipping)
	return customerModel{
		ID: c.ID, TenantID: c.TenantID, WooCustomerID: c.WooCustomerID,
		CustomerType: string(c.TypeOrPrivate()), CompanyName: c.CompanyName, ClientNumber: c.ClientNumber,
		KVKNumber: c.KVKNumber, VATNumber: c.VATNumber,
		Email: c.Email, FirstName: c.FirstName, LastName: c.LastName, PhoneE164: c.PhoneE164, Username: c.Username, Role: c.Role,
		BillingFirstName: bFirst, BillingLastName: bLast, BillingCompany: bCompany,
		BillingAddress1: bAddr1, BillingAddress2: bAddr2, BillingCity: bCity, BillingState: bState,
		BillingPostcode: bPostcode, BillingCountry: bCountry, BillingEmail: bEmail, BillingPhone: bPhone,
		ShippingFirstName: sFirst, ShippingLastName: sLast, ShippingCompany: sCompany,
		ShippingAddress1: sAddr1, ShippingAddress2: sAddr2, ShippingCity: sCity, ShippingState: sState,
		ShippingPostcode: sPostcode, ShippingCountry: sCountry, ShippingPhone: sPhone,
		IsPayingCustomer: c.IsPayingCustomer, AvatarURL: c.AvatarURL, CustomerNote: c.CustomerNote,
		DateCreated: c.DateCreated, DateModified: c.DateModified,
	}
}

func customerAddressFields(a customersDomain.Address) (firstName, lastName, company, address1, address2, city, state, postcode, country, email, phone string) {
	return a.FirstName, a.LastName, a.Company, a.Address1, a.Address2, a.City, a.State, a.Postcode, a.Country, a.Email, a.Phone
}

func toCustomerDomain(m customerModel) *customersDomain.Customer {
	return &customersDomain.Customer{
		ID: m.ID, TenantID: m.TenantID, WooCustomerID: m.WooCustomerID,
		Type: customersDomain.Type(m.CustomerType), CompanyName: m.CompanyName, ClientNumber: m.ClientNumber,
		KVKNumber: m.KVKNumber, VATNumber: m.VATNumber,
		Email: m.Email, FirstName: m.FirstName, LastName: m.LastName, PhoneE164: m.PhoneE164, Username: m.Username, Role: m.Role,
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
		IsPayingCustomer: m.IsPayingCustomer, AvatarURL: m.AvatarURL, CustomerNote: m.CustomerNote,
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
	if c.ClientNumber == "" {
		if c.ClientNumber, err = nextClientNumber(tx, c.TenantID); err != nil {
			return fmt.Errorf("customerRepo.Create: %w", err)
		}
	}
	m := customerModelFromDomain(c)
	if err := tx.Create(&m).Error; err != nil {
		if mapped := customerConstraintError(err); mapped != nil {
			return mapped
		}
		return fmt.Errorf("customerRepo.Create: %w", err)
	}
	c.ID = m.ID
	return nil
}

// nextClientNumber is one above the tenant's highest numeric client number.
// A transaction-scoped advisory lock per tenant serializes concurrent
// creates, so two new customers never draw the same number.
func nextClientNumber(tx *gorm.DB, tenantID string) (string, error) {
	if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtext('customers.client_number:' || ?))`, tenantID).Error; err != nil {
		return "", err
	}
	var next int64
	err := tx.Raw(`SELECT COALESCE(MAX(client_number::BIGINT), 0) + 1 FROM customers
		WHERE tenant_id = ? AND client_number ~ '^[0-9]{1,18}$'`, tenantID).Scan(&next).Error
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(next, 10), nil
}

// customerConstraintError maps a unique-index violation to its domain error.
func customerConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}
	switch pgErr.ConstraintName {
	case "customers_tenant_woo_customer_id_idx":
		return customersDomain.ErrDuplicateWooCustomerID
	case "customers_tenant_client_number_idx":
		return customersDomain.ErrDuplicateClientNumber
	}
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

func (r *CustomerRepo) FindByWooCustomerID(ctx context.Context, wooCustomerID int64) (*customersDomain.Customer, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m customerModel
	err = tx.Where("woo_customer_id = ?", wooCustomerID).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("customerRepo.FindByWooCustomerID: %w", err)
	}
	return toCustomerDomain(m), nil
}

func (r *CustomerRepo) FindByEmail(ctx context.Context, email string) (*customersDomain.Customer, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m customerModel
	err = tx.Where("LOWER(email) = LOWER(?)", email).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("customerRepo.FindByEmail: %w", err)
	}
	return toCustomerDomain(m), nil
}

func (r *CustomerRepo) FindByPhoneE164(ctx context.Context, phoneE164 string) (*customersDomain.Customer, error) {
	if phoneE164 == "" {
		return nil, nil
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m customerModel
	err = tx.Where("phone_e164 = ?", phoneE164).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("customerRepo.FindByPhoneE164: %w", err)
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

func (r *CustomerRepo) Search(ctx context.Context, query string, limit int) ([]customersDomain.Customer, error) {
	if query == "" {
		return nil, nil
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 200
	}
	pattern := "%" + query + "%"
	var rows []customerModel
	err = tx.Where(
		"unaccent(first_name || ' ' || last_name) ILIKE unaccent(?) OR "+
			"similarity(immutable_unaccent(first_name || ' ' || last_name), immutable_unaccent(?)) > ? OR "+
			"unaccent(company_name) ILIKE unaccent(?) OR "+
			"similarity(immutable_unaccent(company_name), immutable_unaccent(?)) > ? OR "+
			"unaccent(billing_company) ILIKE unaccent(?) OR "+
			"email ILIKE ? OR client_number = ? OR kvk_number ILIKE ? OR vat_number ILIKE ? OR "+
			"EXISTS (SELECT 1 FROM customer_contact_persons cp WHERE cp.customer_id = customers.id "+
			"AND unaccent(cp.first_name || ' ' || cp.last_name) ILIKE unaccent(?))",
		pattern, query, fuzzySearchSimilarityThreshold,
		pattern, query, fuzzySearchSimilarityThreshold,
		pattern, pattern, query, pattern, pattern, pattern,
	).Clauses(clause.OrderBy{
		Expression: clause.Expr{
			SQL: "GREATEST(similarity(immutable_unaccent(first_name || ' ' || last_name), immutable_unaccent(?)), " +
				"similarity(immutable_unaccent(company_name), immutable_unaccent(?))) DESC, company_name asc, first_name asc",
			Vars: []interface{}{query, query},
		},
	}).Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("customerRepo.Search: %w", err)
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
		"customer_type": m.CustomerType, "company_name": m.CompanyName, "client_number": m.ClientNumber,
		"kvk_number": m.KVKNumber, "vat_number": m.VATNumber,
		"woo_customer_id": m.WooCustomerID, "email": m.Email, "first_name": m.FirstName, "last_name": m.LastName,
		"phone_e164": m.PhoneE164, "username": m.Username, "role": m.Role,
		"billing_first_name": m.BillingFirstName, "billing_last_name": m.BillingLastName, "billing_company": m.BillingCompany,
		"billing_address_1": m.BillingAddress1, "billing_address_2": m.BillingAddress2, "billing_city": m.BillingCity,
		"billing_state": m.BillingState, "billing_postcode": m.BillingPostcode, "billing_country": m.BillingCountry,
		"billing_email": m.BillingEmail, "billing_phone": m.BillingPhone,
		"shipping_first_name": m.ShippingFirstName, "shipping_last_name": m.ShippingLastName, "shipping_company": m.ShippingCompany,
		"shipping_address_1": m.ShippingAddress1, "shipping_address_2": m.ShippingAddress2, "shipping_city": m.ShippingCity,
		"shipping_state": m.ShippingState, "shipping_postcode": m.ShippingPostcode, "shipping_country": m.ShippingCountry,
		"shipping_phone":     m.ShippingPhone,
		"is_paying_customer": m.IsPayingCustomer, "avatar_url": m.AvatarURL, "customer_note": m.CustomerNote, "date_modified": m.DateModified,
	}
	if err := tx.Model(&customerModel{}).Where("id = ?", c.ID).Updates(values).Error; err != nil {
		if mapped := customerConstraintError(err); mapped != nil {
			return mapped
		}
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
