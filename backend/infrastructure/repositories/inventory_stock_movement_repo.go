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

func (r *StockMovementRepo) TotalOnHand(ctx context.Context) (int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return 0, err
	}
	var total int64
	if err := tx.Model(&stockMovementModel{}).Select("COALESCE(SUM(quantity_delta), 0)").Scan(&total).Error; err != nil {
		return 0, fmt.Errorf("stockMovementRepo.TotalOnHand: %w", err)
	}
	return total, nil
}

// dailyStockFlowRow avoids the "in"/"out" reserved-word clash SQL would
// hit aliasing columns to those names directly.
type dailyStockFlowRow struct {
	Date   time.Time
	InQty  int64 `gorm:"column:in_qty"`
	OutQty int64 `gorm:"column:out_qty"`
}

func (r *StockMovementRepo) DailyFlow(ctx context.Context, since time.Time, timezone string) ([]invDomain.DailyStockFlow, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if err := useReportTimezone(tx, timezone); err != nil {
		return nil, fmt.Errorf("stockMovementRepo.DailyFlow: %w", err)
	}
	var rows []dailyStockFlowRow
	err = tx.Raw(`
		SELECT
			d::date AS date,
			COALESCE(SUM(CASE WHEN sm.quantity_delta > 0 THEN sm.quantity_delta ELSE 0 END), 0) AS in_qty,
			COALESCE(SUM(CASE WHEN sm.quantity_delta < 0 THEN -sm.quantity_delta ELSE 0 END), 0) AS out_qty
		FROM generate_series(?::date, now()::date, interval '1 day') d
		LEFT JOIN stock_movements sm ON sm.occurred_at::date = d
		GROUP BY d
		ORDER BY d
	`, since).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("stockMovementRepo.DailyFlow: %w", err)
	}
	out := make([]invDomain.DailyStockFlow, len(rows))
	for i, row := range rows {
		out[i] = invDomain.DailyStockFlow{Date: row.Date, In: row.InQty, Out: row.OutQty}
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
