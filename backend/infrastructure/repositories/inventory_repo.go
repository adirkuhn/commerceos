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
	Status      string `gorm:"not null;default:'active'"`
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
	status := b.Status
	if status == "" {
		status = invDomain.BatchStatusActive
	}
	m := batchModel{
		TenantID:    b.TenantID,
		SKUID:       b.SKUID,
		WarehouseID: b.WarehouseID,
		LotNumber:   b.LotNumber,
		ExpiryDate:  b.ExpiryDate,
		ReceivedAt:  b.ReceivedAt,
		Status:      status,
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

func (r *BatchRepo) ListBySKUs(ctx context.Context, skuIDs []string) ([]invDomain.Batch, error) {
	if len(skuIDs) == 0 {
		return nil, nil
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []batchModel
	if err := tx.Where("sku_id IN ?", skuIDs).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("batchRepo.ListBySKUs: %w", err)
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

func (r *BatchRepo) MarkExpired(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&batchModel{}).Where("id = ?", id).
		Updates(map[string]any{"status": invDomain.BatchStatusExpired, "updated_at": time.Now()}).Error; err != nil {
		return fmt.Errorf("batchRepo.MarkExpired: %w", err)
	}
	return nil
}

// batchToExpireRow scans ListToExpire's joined query -- not a persisted
// model, just a query-result shape.
type batchToExpireRow struct {
	BatchID  string `gorm:"column:batch_id"`
	TenantID string `gorm:"column:tenant_id"`
}

func (r *BatchRepo) ListToExpire(ctx context.Context) ([]invDomain.BatchToExpire, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []batchToExpireRow
	err = tx.Raw(`
		SELECT b.id AS batch_id, b.tenant_id AS tenant_id
		FROM batches b
		JOIN tenant_settings ts ON ts.tenant_id = b.tenant_id
		WHERE b.status = ?
			AND b.expiry_date IS NOT NULL
			AND ts.batch_expiry_threshold_days > 0
			AND b.expiry_date <= (now() - make_interval(days => ts.batch_expiry_threshold_days))
	`, invDomain.BatchStatusActive).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("batchRepo.ListToExpire: %w", err)
	}
	out := make([]invDomain.BatchToExpire, len(rows))
	for i, row := range rows {
		out[i] = invDomain.BatchToExpire{BatchID: row.BatchID, TenantID: row.TenantID}
	}
	return out, nil
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

// expiringBatchRow scans the joined report query in ListExpiringSoon --
// it isn't a persisted model (no TableName), just a query-result shape.
type expiringBatchRow struct {
	BatchID string `gorm:"column:batch_id"`
	// GORM's naming strategy doesn't split the "SKUID" acronym into
	// "sku_id" for raw-query result scanning the way it does for
	// compound words like WarehouseID -> warehouse_id, so this needs an
	// explicit column tag (same reason batchModel.SKUID has one).
	SKUID         string `gorm:"column:sku_id"`
	SKUCode       string
	ProductName   string
	WarehouseID   string
	WarehouseName string
	LotNumber     string
	ExpiryDate    time.Time
	Quantity      int64
}

func (r *BatchRepo) ListExpiringSoon(ctx context.Context, withinDays int) ([]invDomain.ExpiringBatch, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []expiringBatchRow
	err = tx.Raw(`
		SELECT
			b.id AS batch_id,
			b.sku_id AS sku_id,
			s.sku_code AS sku_code,
			p.name AS product_name,
			b.warehouse_id AS warehouse_id,
			w.name AS warehouse_name,
			b.lot_number AS lot_number,
			b.expiry_date AS expiry_date,
			COALESCE(SUM(sm.quantity_delta), 0) AS quantity
		FROM batches b
		JOIN skus s ON s.id = b.sku_id
		JOIN variants v ON v.id = s.variant_id
		JOIN products p ON p.id = v.product_id
		JOIN warehouses w ON w.id = b.warehouse_id
		LEFT JOIN stock_movements sm ON sm.batch_id = b.id
		WHERE b.expiry_date IS NOT NULL
			AND b.expiry_date <= (now() + make_interval(days => ?))
		GROUP BY b.id, s.sku_code, p.name, w.name
		HAVING COALESCE(SUM(sm.quantity_delta), 0) > 0
		ORDER BY b.expiry_date ASC
	`, withinDays).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("batchRepo.ListExpiringSoon: %w", err)
	}
	out := make([]invDomain.ExpiringBatch, len(rows))
	for i, row := range rows {
		out[i] = invDomain.ExpiringBatch{
			BatchID: row.BatchID, SKUID: row.SKUID, SKUCode: row.SKUCode, ProductName: row.ProductName,
			WarehouseID: row.WarehouseID, WarehouseName: row.WarehouseName, LotNumber: row.LotNumber,
			ExpiryDate: row.ExpiryDate, Quantity: row.Quantity,
		}
	}
	return out, nil
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
		Status:      m.Status,
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

func (r *StockMovementRepo) ListBySKUs(ctx context.Context, skuIDs []string) ([]invDomain.StockMovement, error) {
	if len(skuIDs) == 0 {
		return nil, nil
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []stockMovementModel
	if err := tx.Where("sku_id IN ?", skuIDs).Order("occurred_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("stockMovementRepo.ListBySKUs: %w", err)
	}
	out := make([]invDomain.StockMovement, len(rows))
	for i, row := range rows {
		out[i] = *toStockMovementDomain(row)
	}
	return out, nil
}

func (r *StockMovementRepo) ListDistinctSKUIDs(ctx context.Context, changedSince *time.Time, limit, offset int) ([]string, int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, 0, err
	}
	q := tx.Model(&stockMovementModel{})
	if changedSince != nil {
		q = q.Where("created_at > ?", *changedSince)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Distinct("sku_id").Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("stockMovementRepo.ListDistinctSKUIDs: count: %w", err)
	}

	var skuIDs []string
	if err := q.Session(&gorm.Session{}).Distinct("sku_id").Order("sku_id").Limit(limit).Offset(offset).Pluck("sku_id", &skuIDs).Error; err != nil {
		return nil, 0, fmt.Errorf("stockMovementRepo.ListDistinctSKUIDs: %w", err)
	}
	return skuIDs, total, nil
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
