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

type batchModel struct {
	ID                      string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID                string `gorm:"not null;type:uuid"`
	SKUID                   string `gorm:"column:sku_id;not null;type:uuid"`
	WarehouseID             string `gorm:"not null;type:uuid"`
	LotNumber               string
	ExpiryDate              *time.Time
	ReceivedAt              time.Time
	Status                  string `gorm:"not null;default:'active'"`
	ReviewedAt              *time.Time
	PurchaseOrderLineItemID *string `gorm:"column:purchase_order_line_item_id;type:uuid"`
	LandedCostMinorUnits    *int64  `gorm:"column:landed_cost_minor_units"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func (batchModel) TableName() string { return "batches" }

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
		TenantID:                b.TenantID,
		SKUID:                   b.SKUID,
		WarehouseID:             b.WarehouseID,
		LotNumber:               b.LotNumber,
		ExpiryDate:              b.ExpiryDate,
		ReceivedAt:              b.ReceivedAt,
		Status:                  status,
		PurchaseOrderLineItemID: b.PurchaseOrderLineItemID,
		LandedCostMinorUnits:    b.LandedCostMinorUnits,
		CreatedAt:               b.CreatedAt,
		UpdatedAt:               b.UpdatedAt,
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
	SKUID    string `gorm:"column:sku_id"`
}

func (r *BatchRepo) ListToExpire(ctx context.Context) ([]invDomain.BatchToExpire, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []batchToExpireRow
	err = tx.Raw(`
		SELECT b.id AS batch_id, b.tenant_id AS tenant_id, b.sku_id AS sku_id
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
		out[i] = invDomain.BatchToExpire{BatchID: row.BatchID, TenantID: row.TenantID, SKUID: row.SKUID}
	}
	return out, nil
}

func (r *BatchRepo) MarkReviewed(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&batchModel{}).Where("id = ?", id).
		Update("reviewed_at", time.Now()).Error; err != nil {
		return fmt.Errorf("batchRepo.MarkReviewed: %w", err)
	}
	return nil
}

func (r *BatchRepo) ExtendExpiry(ctx context.Context, id string, newExpiryDate time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&batchModel{}).Where("id = ?", id).
		Updates(map[string]any{
			"expiry_date": newExpiryDate,
			"status":      invDomain.BatchStatusActive,
			"updated_at":  time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("batchRepo.ExtendExpiry: %w", err)
	}
	return nil
}

// expiredBatchReviewRow scans ListPendingReview's joined query -- not a
// persisted model, just a query-result shape (same shape as
// expiringBatchRow, kept separate since the two queries filter on
// different things and could diverge).
type expiredBatchReviewRow struct {
	BatchID       string `gorm:"column:batch_id"`
	SKUID         string `gorm:"column:sku_id"`
	SKUCode       string
	ProductName   string
	WarehouseID   string
	WarehouseName string
	LotNumber     string
	ExpiryDate    time.Time
	Quantity      int64
}

func (r *BatchRepo) ListPendingReview(ctx context.Context) ([]invDomain.ExpiredBatchReview, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []expiredBatchReviewRow
	err = tx.Raw(pendingReviewBatchesSQL, invDomain.BatchStatusExpired).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("batchRepo.ListPendingReview: %w", err)
	}
	out := make([]invDomain.ExpiredBatchReview, len(rows))
	for i, row := range rows {
		out[i] = invDomain.ExpiredBatchReview{
			BatchID: row.BatchID, SKUID: row.SKUID, SKUCode: row.SKUCode, ProductName: row.ProductName,
			WarehouseID: row.WarehouseID, WarehouseName: row.WarehouseName, LotNumber: row.LotNumber,
			ExpiryDate: row.ExpiryDate, Quantity: row.Quantity,
		}
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
	err = tx.Raw(expiringSoonBatchesSQL, withinDays).Scan(&rows).Error
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
		ID:                      m.ID,
		TenantID:                m.TenantID,
		SKUID:                   m.SKUID,
		WarehouseID:             m.WarehouseID,
		LotNumber:               m.LotNumber,
		ExpiryDate:              m.ExpiryDate,
		ReceivedAt:              m.ReceivedAt,
		Status:                  m.Status,
		ReviewedAt:              m.ReviewedAt,
		PurchaseOrderLineItemID: m.PurchaseOrderLineItemID,
		LandedCostMinorUnits:    m.LandedCostMinorUnits,
		CreatedAt:               m.CreatedAt,
		UpdatedAt:               m.UpdatedAt,
	}
}

// pendingReviewBatchesSQL is ListPendingReview's query (one parameter: the
// Expired status) -- shared with CountPendingReview so the dashboard's count
// is always exactly the review queue's length.
const pendingReviewBatchesSQL = `
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
		WHERE b.status = ? AND b.reviewed_at IS NULL
		GROUP BY b.id, s.sku_code, p.name, w.name
		ORDER BY b.expiry_date ASC
	`

// expiringSoonBatchesSQL is ListExpiringSoon's query (one parameter: the
// window in days) -- shared with CountExpiringSoon, same reason as
// pendingReviewBatchesSQL.
const expiringSoonBatchesSQL = `
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
	`

var _ invDomain.BatchQueueCounter = (*BatchRepo)(nil)

// CountExpiringSoon is len(ListExpiringSoon(withinDays)) without loading
// the rows -- see invDomain.BatchQueueCounter.
func (r *BatchRepo) CountExpiringSoon(ctx context.Context, withinDays int) (int64, error) {
	n, err := r.countRows(ctx, expiringSoonBatchesSQL, withinDays)
	if err != nil {
		return 0, fmt.Errorf("batchRepo.CountExpiringSoon: %w", err)
	}
	return n, nil
}

// CountPendingReview is len(ListPendingReview()) without loading the rows
// -- see invDomain.BatchQueueCounter.
func (r *BatchRepo) CountPendingReview(ctx context.Context) (int64, error) {
	n, err := r.countRows(ctx, pendingReviewBatchesSQL, invDomain.BatchStatusExpired)
	if err != nil {
		return 0, fmt.Errorf("batchRepo.CountPendingReview: %w", err)
	}
	return n, nil
}

// countRows counts the rows a list query returns by wrapping it as a
// subquery, so the count can never drift from the list it counts.
func (r *BatchRepo) countRows(ctx context.Context, listSQL string, args ...any) (int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return 0, err
	}
	var n int64
	if err := tx.Raw("SELECT COUNT(*) FROM ("+listSQL+") AS listed", args...).Scan(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}
