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

// ── DiscardRecord repo ───────────────────────────────────────────────────────

type discardRecordModel struct {
	ID          string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID    string `gorm:"not null;type:uuid"`
	SKUID       string `gorm:"column:sku_id;not null;type:uuid"`
	WarehouseID string `gorm:"not null;type:uuid"`
	BatchID     *string
	MovementID  string `gorm:"not null;type:uuid"`
	Type        string `gorm:"not null"`
	Quantity    int64  `gorm:"not null"`
	Reason      string
	PhotoURL    *string `gorm:"column:photo_url"`
	DiscardedBy string
	DiscardedAt time.Time
	CreatedAt   time.Time
}

func (discardRecordModel) TableName() string { return "discard_records" }

type DiscardRecordRepo struct{ db *gorm.DB }

func NewDiscardRecordRepo(db *gorm.DB) *DiscardRecordRepo { return &DiscardRecordRepo{db: db} }

func (r *DiscardRecordRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("inventory: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *DiscardRecordRepo) Create(ctx context.Context, d *invDomain.DiscardRecord) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := discardRecordModel{
		TenantID: d.TenantID, SKUID: d.SKUID, WarehouseID: d.WarehouseID, BatchID: d.BatchID,
		MovementID: d.MovementID, Type: string(d.Type), Quantity: d.Quantity, Reason: d.Reason, PhotoURL: d.PhotoURL,
		DiscardedBy: d.DiscardedBy, DiscardedAt: d.DiscardedAt, CreatedAt: d.CreatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("discardRecordRepo.Create: %w", err)
	}
	d.ID = m.ID
	return nil
}

// discardRecordReportRow scans the joined report query in List -- not a
// persisted model, just a query-result shape (same reasoning as
// expiringBatchRow/expiredBatchReviewRow).
type discardRecordReportRow struct {
	ID            string `gorm:"column:id"`
	TenantID      string
	SKUID         string `gorm:"column:sku_id"`
	WarehouseID   string
	BatchID       *string
	MovementID    string
	Type          string
	Quantity      int64
	Reason        string
	PhotoURL      *string `gorm:"column:photo_url"`
	DiscardedBy   string
	DiscardedAt   time.Time
	CreatedAt     time.Time
	SKUCode       string
	ProductName   string
	WarehouseName string
	LotNumber     string
}

func (r *DiscardRecordRepo) List(ctx context.Context, filter invDomain.DiscardRecordFilter) ([]invDomain.DiscardRecordReport, int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, 0, err
	}

	base := tx.Table("discard_records d").
		Joins("JOIN skus s ON s.id = d.sku_id").
		Joins("JOIN variants v ON v.id = s.variant_id").
		Joins("JOIN products p ON p.id = v.product_id").
		Joins("JOIN warehouses w ON w.id = d.warehouse_id").
		Joins("LEFT JOIN batches b ON b.id = d.batch_id")
	if filter.WarehouseID != nil {
		base = base.Where("d.warehouse_id = ?", *filter.WarehouseID)
	}
	if filter.SKUID != nil {
		base = base.Where("d.sku_id = ?", *filter.SKUID)
	}
	if filter.Type != nil {
		base = base.Where("d.type = ?", string(*filter.Type))
	}
	if filter.From != nil {
		base = base.Where("d.discarded_at >= ?", *filter.From)
	}
	if filter.To != nil {
		base = base.Where("d.discarded_at <= ?", *filter.To)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("discardRecordRepo.List: count: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 200
	}
	var rows []discardRecordReportRow
	err = base.Select(`
			d.id AS id, d.tenant_id AS tenant_id, d.sku_id AS sku_id, d.warehouse_id AS warehouse_id,
			d.batch_id AS batch_id, d.movement_id AS movement_id, d.type AS type, d.quantity AS quantity, d.reason AS reason,
			d.photo_url AS photo_url, d.discarded_by AS discarded_by, d.discarded_at AS discarded_at, d.created_at AS created_at,
			s.sku_code AS sku_code, p.name AS product_name, w.name AS warehouse_name,
			COALESCE(b.lot_number, '') AS lot_number
		`).
		Order("d.discarded_at DESC").Limit(limit).Offset(filter.Offset).Scan(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("discardRecordRepo.List: %w", err)
	}

	out := make([]invDomain.DiscardRecordReport, len(rows))
	for i, row := range rows {
		out[i] = invDomain.DiscardRecordReport{
			DiscardRecord: invDomain.DiscardRecord{
				ID: row.ID, TenantID: row.TenantID, SKUID: row.SKUID, WarehouseID: row.WarehouseID,
				BatchID: row.BatchID, MovementID: row.MovementID, Type: invDomain.MovementType(row.Type), Quantity: row.Quantity, Reason: row.Reason,
				PhotoURL: row.PhotoURL, DiscardedBy: row.DiscardedBy, DiscardedAt: row.DiscardedAt, CreatedAt: row.CreatedAt,
			},
			SKUCode: row.SKUCode, ProductName: row.ProductName, WarehouseName: row.WarehouseName, LotNumber: row.LotNumber,
		}
	}
	return out, total, nil
}
