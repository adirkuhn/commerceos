package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── GORM model ───────────────────────────────────────────────────────────────

type supplierModel struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string `gorm:"not null;type:uuid"`
	Name         string `gorm:"not null"`
	Email        string `gorm:"not null;default:''"`
	Phone        string `gorm:"not null;default:''"`
	Country      string `gorm:"not null;default:''"`
	LeadTimeDays *int
	Status       string `gorm:"not null;default:'active'"`
	Notes        string `gorm:"not null;default:''"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (supplierModel) TableName() string { return "suppliers" }

// ── SupplierRepo ─────────────────────────────────────────────────────────────

type SupplierRepo struct{ db *gorm.DB }

func NewSupplierRepo(db *gorm.DB) *SupplierRepo { return &SupplierRepo{db: db} }

func (r *SupplierRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("suppliers: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *SupplierRepo) Create(ctx context.Context, s *suppliersDomain.Supplier) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := supplierModel{
		TenantID: s.TenantID, Name: s.Name, Email: s.Email, Phone: s.Phone, Country: s.Country,
		LeadTimeDays: s.LeadTimeDays, Status: string(s.Status), Notes: s.Notes,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("supplierRepo.Create: %w", err)
	}
	s.ID = m.ID
	return nil
}

func (r *SupplierRepo) FindByID(ctx context.Context, id string) (*suppliersDomain.Supplier, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m supplierModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, suppliersDomain.ErrSupplierNotFound
		}
		return nil, fmt.Errorf("supplierRepo.FindByID: %w", err)
	}
	return toSupplierDomain(m), nil
}

func (r *SupplierRepo) FindByName(ctx context.Context, name string) (*suppliersDomain.Supplier, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m supplierModel
	if err := tx.Where("LOWER(TRIM(name)) = LOWER(TRIM(?))", name).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, suppliersDomain.ErrSupplierNotFound
		}
		return nil, fmt.Errorf("supplierRepo.FindByName: %w", err)
	}
	return toSupplierDomain(m), nil
}

func (r *SupplierRepo) List(ctx context.Context) ([]suppliersDomain.Supplier, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []supplierModel
	if err := tx.Order("name asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("supplierRepo.List: %w", err)
	}
	out := make([]suppliersDomain.Supplier, len(rows))
	for i, row := range rows {
		out[i] = *toSupplierDomain(row)
	}
	return out, nil
}

func (r *SupplierRepo) Update(ctx context.Context, s *suppliersDomain.Supplier) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&supplierModel{}).Where("id = ?", s.ID).
		Updates(map[string]any{
			"name":           s.Name,
			"email":          s.Email,
			"phone":          s.Phone,
			"country":        s.Country,
			"lead_time_days": s.LeadTimeDays,
			"status":         string(s.Status),
			"notes":          s.Notes,
			"updated_at":     s.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("supplierRepo.Update: %w", err)
	}
	return nil
}

func (r *SupplierRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&supplierModel{}).Error; err != nil {
		return fmt.Errorf("supplierRepo.Delete: %w", err)
	}
	return nil
}

func toSupplierDomain(m supplierModel) *suppliersDomain.Supplier {
	return &suppliersDomain.Supplier{
		ID: m.ID, TenantID: m.TenantID, Name: m.Name, Email: m.Email, Phone: m.Phone, Country: m.Country,
		LeadTimeDays: m.LeadTimeDays, Status: suppliersDomain.SupplierStatus(m.Status), Notes: m.Notes,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
