package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

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

type batchModel struct {
	ID          string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID    string `gorm:"not null;type:uuid"`
	SKUID       string `gorm:"column:sku_id;not null;type:uuid"`
	WarehouseID string `gorm:"not null;type:uuid"`
	LotNumber   string
	ExpiryDate  *time.Time
	ReceivedAt  time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (batchModel) TableName() string { return "batches" }

type stockMovementModel struct {
	ID            string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID      string `gorm:"not null;type:uuid"`
	SKUID         string `gorm:"column:sku_id;not null;type:uuid"`
	WarehouseID   string `gorm:"not null;type:uuid"`
	BatchID       *string
	Type          string `gorm:"not null"`
	QuantityDelta int64  `gorm:"not null"`
	Reason        string
	OccurredAt    time.Time
	CreatedAt     time.Time
}

func (stockMovementModel) TableName() string { return "stock_movements" }

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

// ── Batch repo ───────────────────────────────────────────────────────────────

type BatchRepo struct{ db *gorm.DB }

func NewBatchRepo(db *gorm.DB) *BatchRepo { return &BatchRepo{db: db} }

func (r *BatchRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("inventory: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *BatchRepo) Create(ctx context.Context, b *invDomain.Batch) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := batchModel{
		TenantID:    b.TenantID,
		SKUID:       b.SKUID,
		WarehouseID: b.WarehouseID,
		LotNumber:   b.LotNumber,
		ExpiryDate:  b.ExpiryDate,
		ReceivedAt:  b.ReceivedAt,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("batchRepo.Create: %w", err)
	}
	b.ID = m.ID
	return nil
}

func (r *BatchRepo) FindByID(ctx context.Context, id string) (*invDomain.Batch, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m batchModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, invDomain.ErrBatchNotFound
		}
		return nil, fmt.Errorf("batchRepo.FindByID: %w", err)
	}
	return toBatchDomain(m), nil
}

func (r *BatchRepo) ListBySKU(ctx context.Context, skuID string) ([]invDomain.Batch, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []batchModel
	if err := tx.Where("sku_id = ?", skuID).Order("received_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("batchRepo.ListBySKU: %w", err)
	}
	out := make([]invDomain.Batch, len(rows))
	for i, row := range rows {
		out[i] = *toBatchDomain(row)
	}
	return out, nil
}

func (r *BatchRepo) Update(ctx context.Context, b *invDomain.Batch) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&batchModel{}).Where("id = ?", b.ID).
		Updates(map[string]any{
			"warehouse_id": b.WarehouseID,
			"lot_number":   b.LotNumber,
			"expiry_date":  b.ExpiryDate,
			"updated_at":   b.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("batchRepo.Update: %w", err)
	}
	return nil
}

func (r *BatchRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&batchModel{}).Error; err != nil {
		return fmt.Errorf("batchRepo.Delete: %w", err)
	}
	return nil
}

func toBatchDomain(m batchModel) *invDomain.Batch {
	return &invDomain.Batch{
		ID:          m.ID,
		TenantID:    m.TenantID,
		SKUID:       m.SKUID,
		WarehouseID: m.WarehouseID,
		LotNumber:   m.LotNumber,
		ExpiryDate:  m.ExpiryDate,
		ReceivedAt:  m.ReceivedAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// ── Stock movement repo (append-only) ─────────────────────────────────────────

type StockMovementRepo struct{ db *gorm.DB }

func NewStockMovementRepo(db *gorm.DB) *StockMovementRepo { return &StockMovementRepo{db: db} }

func (r *StockMovementRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("inventory: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *StockMovementRepo) Create(ctx context.Context, mv *invDomain.StockMovement) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := stockMovementModel{
		TenantID:      mv.TenantID,
		SKUID:         mv.SKUID,
		WarehouseID:   mv.WarehouseID,
		BatchID:       mv.BatchID,
		Type:          string(mv.Type),
		QuantityDelta: mv.QuantityDelta,
		Reason:        mv.Reason,
		OccurredAt:    mv.OccurredAt,
		CreatedAt:     mv.CreatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("stockMovementRepo.Create: %w", err)
	}
	mv.ID = m.ID
	return nil
}

func (r *StockMovementRepo) ListBySKU(ctx context.Context, skuID string) ([]invDomain.StockMovement, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []stockMovementModel
	if err := tx.Where("sku_id = ?", skuID).Order("occurred_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("stockMovementRepo.ListBySKU: %w", err)
	}
	out := make([]invDomain.StockMovement, len(rows))
	for i, row := range rows {
		out[i] = *toStockMovementDomain(row)
	}
	return out, nil
}

func (r *StockMovementRepo) ListByBatch(ctx context.Context, batchID string) ([]invDomain.StockMovement, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []stockMovementModel
	if err := tx.Where("batch_id = ?", batchID).Order("occurred_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("stockMovementRepo.ListByBatch: %w", err)
	}
	out := make([]invDomain.StockMovement, len(rows))
	for i, row := range rows {
		out[i] = *toStockMovementDomain(row)
	}
	return out, nil
}

func toStockMovementDomain(m stockMovementModel) *invDomain.StockMovement {
	return &invDomain.StockMovement{
		ID:            m.ID,
		TenantID:      m.TenantID,
		SKUID:         m.SKUID,
		WarehouseID:   m.WarehouseID,
		BatchID:       m.BatchID,
		Type:          invDomain.MovementType(m.Type),
		QuantityDelta: m.QuantityDelta,
		Reason:        m.Reason,
		OccurredAt:    m.OccurredAt,
		CreatedAt:     m.CreatedAt,
	}
}
