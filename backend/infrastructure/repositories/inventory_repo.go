package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── GORM models ──────────────────────────────────────────────────────────────

type warehouseModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string `gorm:"not null;type:uuid"`
	Name      string `gorm:"not null"`
	Address1  string `gorm:"not null;default:''"`
	Address2  string `gorm:"not null;default:''"`
	City      string `gorm:"not null;default:''"`
	Province  string `gorm:"not null;default:''"`
	Zip       string `gorm:"not null;default:''"`
	Country   string `gorm:"not null;default:''"`
	Phone     string `gorm:"not null;default:''"`
	Active    bool   `gorm:"not null;default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (warehouseModel) TableName() string { return "warehouses" }

// ── Warehouse repo ───────────────────────────────────────────────────────────

type WarehouseRepo struct{ db *gorm.DB }

func NewWarehouseRepo(db *gorm.DB) *WarehouseRepo { return &WarehouseRepo{db: db} }

func (r *WarehouseRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("inventory: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *WarehouseRepo) Create(ctx context.Context, w *invDomain.Warehouse) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := warehouseModel{
		TenantID: w.TenantID, Name: w.Name,
		Address1: w.Address1, Address2: w.Address2, City: w.City, Province: w.Province,
		Zip: w.Zip, Country: w.Country, Phone: w.Phone, Active: w.Active,
		CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "warehouses_tenant_id_name_key" {
			return invDomain.ErrDuplicateWarehouseName
		}
		return fmt.Errorf("warehouseRepo.Create: %w", err)
	}
	w.ID = m.ID
	return nil
}

func (r *WarehouseRepo) FindByID(ctx context.Context, id string) (*invDomain.Warehouse, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m warehouseModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, invDomain.ErrWarehouseNotFound
		}
		return nil, fmt.Errorf("warehouseRepo.FindByID: %w", err)
	}
	return toWarehouseDomain(m), nil
}

func (r *WarehouseRepo) List(ctx context.Context) ([]invDomain.Warehouse, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []warehouseModel
	if err := tx.Order("name asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("warehouseRepo.List: %w", err)
	}
	out := make([]invDomain.Warehouse, len(rows))
	for i, row := range rows {
		out[i] = *toWarehouseDomain(row)
	}
	return out, nil
}

func (r *WarehouseRepo) Update(ctx context.Context, w *invDomain.Warehouse) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&warehouseModel{}).Where("id = ?", w.ID).
		Updates(map[string]any{
			"name": w.Name, "updated_at": w.UpdatedAt,
			"address1": w.Address1, "address2": w.Address2, "city": w.City, "province": w.Province,
			"zip": w.Zip, "country": w.Country, "phone": w.Phone, "active": w.Active,
		}).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "warehouses_tenant_id_name_key" {
			return invDomain.ErrDuplicateWarehouseName
		}
		return fmt.Errorf("warehouseRepo.Update: %w", err)
	}
	return nil
}

func (r *WarehouseRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&warehouseModel{}).Error; err != nil {
		return fmt.Errorf("warehouseRepo.Delete: %w", err)
	}
	return nil
}

func toWarehouseDomain(m warehouseModel) *invDomain.Warehouse {
	return &invDomain.Warehouse{
		ID: m.ID, TenantID: m.TenantID, Name: m.Name,
		Address1: m.Address1, Address2: m.Address2, City: m.City, Province: m.Province,
		Zip: m.Zip, Country: m.Country, Phone: m.Phone, Active: m.Active,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
