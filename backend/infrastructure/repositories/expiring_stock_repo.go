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

// ExpiringStockRepo is the read model behind the stock-close-to-expiry
// report -- see invDomain.ExpiringStockReader.
type ExpiringStockRepo struct{ db *gorm.DB }

func NewExpiringStockRepo(db *gorm.DB) *ExpiringStockRepo { return &ExpiringStockRepo{db: db} }

var _ invDomain.ExpiringStockReader = (*ExpiringStockRepo)(nil)

type expiringStockRow struct {
	SKUID               string     `gorm:"column:sku_id"`
	SKUCode             string     `gorm:"column:sku_code"`
	ProductName         string     `gorm:"column:product_name"`
	ImageURL            string     `gorm:"column:image_url"`
	ExpiryDate          time.Time  `gorm:"column:expiry_date"`
	Quantity            int64      `gorm:"column:quantity"`
	PriceMinorUnits     int64      `gorm:"column:price_minor_units"`
	SalePriceMinorUnits *int64     `gorm:"column:sale_price_minor_units"`
	SaleEndDate         *time.Time `gorm:"column:sale_end_date"`
	CurrencyCode        string     `gorm:"column:currency_code"`
}

func (r *ExpiringStockRepo) ListExpiringStock(ctx context.Context, from, to time.Time, windowDays int) ([]invDomain.ExpiringStockLine, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("expiringStockRepo: operation called outside a transaction")
	}
	var rows []expiringStockRow
	err := tx.WithContext(ctx).Raw(`
		WITH batch_stock AS (
			SELECT b.sku_id, b.expiry_date, SUM(sm.quantity_delta) AS quantity
			FROM batches b
			JOIN stock_movements sm ON sm.batch_id = b.id
			WHERE b.expiry_date BETWEEN ?::date AND ?::date
				AND b.status <> 'expired'
				AND NOT EXISTS (
					SELECT 1 FROM batch_expiry_acknowledgements a
					WHERE a.batch_id = b.id AND a.window_days = ?
				)
			GROUP BY b.id, b.sku_id, b.expiry_date
			HAVING SUM(sm.quantity_delta) > 0
		)
		SELECT
			s.id AS sku_id, s.sku_code, p.name AS product_name,
			COALESCE(fm.url, pm.url, '') AS image_url,
			bs.expiry_date, SUM(bs.quantity) AS quantity,
			s.price_minor_units, s.sale_price_minor_units, s.sale_end_date, s.currency_code
		FROM batch_stock bs
		JOIN skus s ON s.id = bs.sku_id
		JOIN variants v ON v.id = s.variant_id
		JOIN products p ON p.id = v.product_id
		LEFT JOIN media fm ON fm.id = v.featured_media_id
		LEFT JOIN LATERAL (
			SELECT m.url FROM media m WHERE m.product_id = p.id ORDER BY m.position, m.created_at LIMIT 1
		) pm ON true
		GROUP BY s.id, s.sku_code, p.name, fm.url, pm.url, bs.expiry_date,
			s.price_minor_units, s.sale_price_minor_units, s.sale_end_date, s.currency_code
		ORDER BY bs.expiry_date, p.name, s.sku_code
	`, from.Format("2006-01-02"), to.Format("2006-01-02"), windowDays).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("expiringStockRepo.ListExpiringStock: %w", err)
	}
	out := make([]invDomain.ExpiringStockLine, len(rows))
	for i, row := range rows {
		out[i] = invDomain.ExpiringStockLine{
			SKUID: row.SKUID, SKUCode: row.SKUCode, ProductName: row.ProductName, ImageURL: row.ImageURL,
			ExpiryDate: row.ExpiryDate, Quantity: row.Quantity,
			PriceMinorUnits: row.PriceMinorUnits, SalePriceMinorUnits: row.SalePriceMinorUnits,
			SaleEndDate: row.SaleEndDate, CurrencyCode: row.CurrencyCode,
		}
	}
	return out, nil
}
