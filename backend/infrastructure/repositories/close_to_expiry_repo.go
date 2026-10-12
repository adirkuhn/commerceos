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

// CloseToExpiryRepo is the read model behind the close-to-expiry screen --
// see invDomain.CloseToExpiryReader.
type CloseToExpiryRepo struct{ db *gorm.DB }

func NewCloseToExpiryRepo(db *gorm.DB) *CloseToExpiryRepo { return &CloseToExpiryRepo{db: db} }

var _ invDomain.CloseToExpiryReader = (*CloseToExpiryRepo)(nil)

// closeToExpiryBatchesCTE selects the batches with stock left expiring in
// [from, to], not flagged expired, with their acknowledgement for the
// window (if any). Parameters: from, to, window days.
const closeToExpiryBatchesCTE = `
	WITH batch_stock AS (
		SELECT b.id, b.sku_id, b.warehouse_id, b.lot_number, b.expiry_date,
			SUM(sm.quantity_delta) AS quantity, a.acknowledged_at
		FROM batches b
		JOIN stock_movements sm ON sm.batch_id = b.id
		LEFT JOIN batch_expiry_acknowledgements a ON a.batch_id = b.id AND a.window_days = ?
		WHERE b.expiry_date BETWEEN ?::date AND ?::date
			AND b.status <> 'expired'
		GROUP BY b.id, b.sku_id, b.warehouse_id, b.lot_number, b.expiry_date, a.acknowledged_at
		HAVING SUM(sm.quantity_delta) > 0
	)`

type closeToExpiryRow struct {
	BatchID             string     `gorm:"column:batch_id"`
	LotNumber           string     `gorm:"column:lot_number"`
	ExpiryDate          time.Time  `gorm:"column:expiry_date"`
	Quantity            int64      `gorm:"column:quantity"`
	WarehouseID         string     `gorm:"column:warehouse_id"`
	WarehouseName       string     `gorm:"column:warehouse_name"`
	SKUID               string     `gorm:"column:sku_id"`
	SKUCode             string     `gorm:"column:sku_code"`
	ProductID           string     `gorm:"column:product_id"`
	ProductName         string     `gorm:"column:product_name"`
	ImageURL            string     `gorm:"column:image_url"`
	PriceMinorUnits     int64      `gorm:"column:price_minor_units"`
	SalePriceMinorUnits *int64     `gorm:"column:sale_price_minor_units"`
	SaleEndDate         *time.Time `gorm:"column:sale_end_date"`
	CurrencyCode        string     `gorm:"column:currency_code"`
	AcknowledgedAt      *time.Time `gorm:"column:acknowledged_at"`
	PromotionID         *string    `gorm:"column:promotion_id"`
	PromotionName       string     `gorm:"column:promotion_name"`
}

func (r *CloseToExpiryRepo) ListCloseToExpiryBatches(ctx context.Context, q invDomain.CloseToExpiryQuery) ([]invDomain.CloseToExpiryBatch, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("closeToExpiryRepo: operation called outside a transaction")
	}
	var rows []closeToExpiryRow
	err := tx.WithContext(ctx).Raw(closeToExpiryBatchesCTE+`
		SELECT
			bs.id AS batch_id, bs.lot_number, bs.expiry_date, bs.quantity,
			w.id AS warehouse_id, w.name AS warehouse_name,
			s.id AS sku_id, s.sku_code, p.id AS product_id, p.name AS product_name,
			COALESCE(fm.url, pm.url, '') AS image_url,
			s.price_minor_units, s.sale_price_minor_units, s.sale_end_date, s.currency_code,
			bs.acknowledged_at, cl.promotion_id, COALESCE(cl.promotion_name, '') AS promotion_name
		FROM batch_stock bs
		JOIN warehouses w ON w.id = bs.warehouse_id
		JOIN skus s ON s.id = bs.sku_id
		JOIN variants v ON v.id = s.variant_id
		JOIN products p ON p.id = v.product_id
		LEFT JOIN media fm ON fm.id = v.featured_media_id
		LEFT JOIN LATERAL (
			SELECT m.url FROM media m WHERE m.product_id = p.id ORDER BY m.position, m.created_at LIMIT 1
		) pm ON true
		LEFT JOIN LATERAL (
			SELECT pr.id AS promotion_id, pr.name AS promotion_name
			FROM promotion_skus ps JOIN promotions pr ON pr.id = ps.promotion_id
			WHERE ps.batch_id = bs.id AND ps.finished_at IS NULL AND pr.status <> 'ended'
			ORDER BY pr.created_at DESC LIMIT 1
		) cl ON true
		WHERE (bs.acknowledged_at IS NOT NULL) = ?
		ORDER BY bs.expiry_date, p.name, s.sku_code, bs.lot_number, bs.id
		LIMIT ? OFFSET ?
	`, q.WindowDays, q.From.Format("2006-01-02"), q.To.Format("2006-01-02"), q.Acknowledged, q.Limit, q.Offset).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("closeToExpiryRepo.ListCloseToExpiryBatches: %w", err)
	}
	out := make([]invDomain.CloseToExpiryBatch, len(rows))
	for i, row := range rows {
		out[i] = toCloseToExpiryBatch(row)
	}
	return out, nil
}

func (r *CloseToExpiryRepo) CountCloseToExpiryBatches(ctx context.Context, from, to time.Time, windowDays int) (invDomain.CloseToExpiryCounts, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return invDomain.CloseToExpiryCounts{}, errors.New("closeToExpiryRepo: operation called outside a transaction")
	}
	var counts struct {
		Open         int `gorm:"column:open"`
		Acknowledged int `gorm:"column:acknowledged"`
	}
	err := tx.WithContext(ctx).Raw(closeToExpiryBatchesCTE+`
		SELECT
			COUNT(*) FILTER (WHERE acknowledged_at IS NULL) AS open,
			COUNT(*) FILTER (WHERE acknowledged_at IS NOT NULL) AS acknowledged
		FROM batch_stock
	`, windowDays, from.Format("2006-01-02"), to.Format("2006-01-02")).Scan(&counts).Error
	if err != nil {
		return invDomain.CloseToExpiryCounts{}, fmt.Errorf("closeToExpiryRepo.CountCloseToExpiryBatches: %w", err)
	}
	return invDomain.CloseToExpiryCounts{Open: counts.Open, Acknowledged: counts.Acknowledged}, nil
}

func toCloseToExpiryBatch(row closeToExpiryRow) invDomain.CloseToExpiryBatch {
	b := invDomain.CloseToExpiryBatch{
		BatchID: row.BatchID, LotNumber: row.LotNumber, ExpiryDate: row.ExpiryDate, Quantity: row.Quantity,
		WarehouseID: row.WarehouseID, WarehouseName: row.WarehouseName,
		SKUID: row.SKUID, SKUCode: row.SKUCode, ProductID: row.ProductID, ProductName: row.ProductName, ImageURL: row.ImageURL,
		PriceMinorUnits: row.PriceMinorUnits, SalePriceMinorUnits: row.SalePriceMinorUnits, SaleEndDate: row.SaleEndDate,
		CurrencyCode: row.CurrencyCode, AcknowledgedAt: row.AcknowledgedAt,
	}
	if row.PromotionID != nil {
		b.Clearance = &invDomain.ClearancePromotionRef{PromotionID: *row.PromotionID, PromotionName: row.PromotionName}
	}
	return b
}
