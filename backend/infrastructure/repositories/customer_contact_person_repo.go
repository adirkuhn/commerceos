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

type customerContactPersonModel struct {
	ID         string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID   string `gorm:"not null;type:uuid"`
	CustomerID string `gorm:"column:customer_id;not null;type:uuid"`
	FirstName  string `gorm:"column:first_name"`
	LastName   string `gorm:"column:last_name"`
	Role       string
	Email      string
	Phone      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (customerContactPersonModel) TableName() string { return "customer_contact_persons" }

func toContactPersonDomain(m customerContactPersonModel) customersDomain.ContactPerson {
	return customersDomain.ContactPerson{
		ID: m.ID, TenantID: m.TenantID, CustomerID: m.CustomerID, FirstName: m.FirstName, LastName: m.LastName,
		Role: m.Role, Email: m.Email, Phone: m.Phone, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

// CustomerContactPersonRepo stores a company customer's contact persons.
type CustomerContactPersonRepo struct{ db *gorm.DB }

func NewCustomerContactPersonRepo(db *gorm.DB) *CustomerContactPersonRepo {
	return &CustomerContactPersonRepo{db: db}
}

func (r *CustomerContactPersonRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("customerContactPersonRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *CustomerContactPersonRepo) Create(ctx context.Context, p *customersDomain.ContactPerson) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := customerContactPersonModel{
		TenantID: p.TenantID, CustomerID: p.CustomerID, FirstName: p.FirstName, LastName: p.LastName,
		Role: p.Role, Email: p.Email, Phone: p.Phone, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("customerContactPersonRepo.Create: %w", err)
	}
	p.ID = m.ID
	return nil
}

func (r *CustomerContactPersonRepo) FindByID(ctx context.Context, customerID, id string) (*customersDomain.ContactPerson, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m customerContactPersonModel
	if err := tx.Where("id = ? AND customer_id = ?", id, customerID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, customersDomain.ErrContactPersonNotFound
		}
		return nil, fmt.Errorf("customerContactPersonRepo.FindByID: %w", err)
	}
	p := toContactPersonDomain(m)
	return &p, nil
}

func (r *CustomerContactPersonRepo) ListByCustomer(ctx context.Context, customerID string) ([]customersDomain.ContactPerson, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []customerContactPersonModel
	if err := tx.Where("customer_id = ?", customerID).Order("created_at, id").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("customerContactPersonRepo.ListByCustomer: %w", err)
	}
	out := make([]customersDomain.ContactPerson, len(rows))
	for i, m := range rows {
		out[i] = toContactPersonDomain(m)
	}
	return out, nil
}

func (r *CustomerContactPersonRepo) Update(ctx context.Context, p *customersDomain.ContactPerson) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Model(&customerContactPersonModel{}).Where("id = ? AND customer_id = ?", p.ID, p.CustomerID).Updates(map[string]any{
		"first_name": p.FirstName, "last_name": p.LastName, "role": p.Role, "email": p.Email, "phone": p.Phone, "updated_at": p.UpdatedAt,
	})
	if res.Error != nil {
		return fmt.Errorf("customerContactPersonRepo.Update: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return customersDomain.ErrContactPersonNotFound
	}
	return nil
}

func (r *CustomerContactPersonRepo) Delete(ctx context.Context, customerID, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Where("id = ? AND customer_id = ?", id, customerID).Delete(&customerContactPersonModel{})
	if res.Error != nil {
		return fmt.Errorf("customerContactPersonRepo.Delete: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return customersDomain.ErrContactPersonNotFound
	}
	return nil
}
