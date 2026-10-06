package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/domain/procurement/planning"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// PurchasePlanningRepo is the Purchase Planner's read model: catalog +
// stock, sales, and open/past purchase orders, each as one aggregate
// query rather than a lookup per SKU.
type PurchasePlanningRepo struct{ db *gorm.DB }

func NewPurchasePlanningRepo(db *gorm.DB) *PurchasePlanningRepo { return &PurchasePlanningRepo{db: db} }

var (
	_ planning.CatalogReader = (*PurchasePlanningRepo)(nil)
	_ planning.DemandReader  = (*PurchasePlanningRepo)(nil)
	_ planning.SupplyReader  = (*PurchasePlanningRepo)(nil)
)

func (r *PurchasePlanningRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("purchasePlanning: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// productLineTypes are the PO line types that are goods (not shipping/fees).
var productLineTypes = []string{"", string(procurementDomain.LineItemTypeProduct)}

type planningSKURow struct {
	SKUID              string  `gorm:"column:sku_id"`
	SKUCode            string  `gorm:"column:sku_code"`
	ProductName        string  `gorm:"column:product_name"`
	VariantName        string  `gorm:"column:variant_name"`
	UnitType           string  `gorm:"column:unit_type"`
	PortionedFromSKUID *string `gorm:"column:portioned_from_sku_id"`
	WeightGrams        *int64  `gorm:"column:weight_grams"`
	Active             bool    `gorm:"column:active"`
	OnHand             int64   `gorm:"column:on_hand"`
}

func (r *PurchasePlanningRepo) ListPlanningSKUs(ctx context.Context) ([]planning.PlanningSKU, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []planningSKURow
	err = tx.Raw(`
		SELECT s.id AS sku_id, s.sku_code, p.name AS product_name, v.name AS variant_name,
			s.unit_type, s.portioned_from_sku_id, s.weight_grams,
			(s.status = ? AND p.status <> ?) AS active,
			COALESCE(st.on_hand, 0) AS on_hand
		FROM skus s
		JOIN variants v ON v.id = s.variant_id
		JOIN products p ON p.id = v.product_id
		LEFT JOIN (
			SELECT sku_id, SUM(quantity_delta) AS on_hand FROM stock_movements GROUP BY sku_id
		) st ON st.sku_id = s.id
		WHERE NOT s.is_virtual
		ORDER BY p.name, s.sku_code
	`, string(pmsDomain.SKUStatusActive), string(pmsDomain.ProductStatusArchived)).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("purchasePlanningRepo.ListPlanningSKUs: %w", err)
	}
	out := make([]planning.PlanningSKU, len(rows))
	for i, row := range rows {
		out[i] = planning.PlanningSKU{
			SKUID: row.SKUID, SKUCode: row.SKUCode, ProductName: row.ProductName, VariantName: row.VariantName,
			IsWeightPool:       row.UnitType == string(pmsDomain.SKUUnitTypeKilo),
			PortionedFromSKUID: row.PortionedFromSKUID, WeightGrams: row.WeightGrams,
			Active: row.Active, OnHand: row.OnHand,
		}
	}
	return out, nil
}

type skuQuantityRow struct {
	SKUID    string `gorm:"column:sku_id"`
	Quantity int64  `gorm:"column:quantity"`
}

// UnitsSoldPerSKUBetween counts net units (refunds excluded) on orders in
// revenue-counting statuses -- the same orders every sales report counts.
// Every currency: units don't depend on what an order was paid in.
func (r *PurchasePlanningRepo) UnitsSoldPerSKUBetween(ctx context.Context, from, to time.Time) (map[string]int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []skuQuantityRow
	err = tx.Raw(`
		SELECT li.sku_id, SUM(`+netLineQuantitySQL+`) AS quantity
		FROM sales_order_line_items li
		JOIN sales_orders so ON so.id = li.sales_order_id AND `+lineNotFullyRefundedSQL+`
		WHERE li.sku_id IS NOT NULL AND so.occurred_at >= ? AND so.occurred_at < ? AND so.status IN ?
		GROUP BY li.sku_id
	`, from, to, revenueCountingStatuses).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("purchasePlanningRepo.UnitsSoldPerSKUBetween: %w", err)
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.SKUID] = row.Quantity
	}
	return out, nil
}

type skuFirstSaleRow struct {
	SKUID       string    `gorm:"column:sku_id"`
	FirstSaleAt time.Time `gorm:"column:first_sale_at"`
}

func (r *PurchasePlanningRepo) FirstSalePerSKU(ctx context.Context) (map[string]time.Time, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []skuFirstSaleRow
	err = tx.Raw(`
		SELECT li.sku_id, MIN(so.occurred_at) AS first_sale_at
		FROM sales_order_line_items li
		JOIN sales_orders so ON so.id = li.sales_order_id AND `+lineNotFullyRefundedSQL+`
		WHERE li.sku_id IS NOT NULL AND so.status IN ?
		GROUP BY li.sku_id
	`, revenueCountingStatuses).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("purchasePlanningRepo.FirstSalePerSKU: %w", err)
	}
	out := make(map[string]time.Time, len(rows))
	for _, row := range rows {
		out[row.SKUID] = row.FirstSaleAt
	}
	return out, nil
}

type openOrderRow struct {
	SKUID string `gorm:"column:sku_id"`
	Sent  int64  `gorm:"column:sent"`
	Draft int64  `gorm:"column:draft"`
}

// OpenOrderQuantities sums stock units (quantity x units per purchased
// item) on POs not received yet. A partially received PO's remainder isn't
// counted: what's still missing is often never coming.
func (r *PurchasePlanningRepo) OpenOrderQuantities(ctx context.Context) (map[string]planning.OpenOrderQuantity, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	sent := []string{string(procurementDomain.PurchaseOrderStatusSent)}
	draft := []string{string(procurementDomain.PurchaseOrderStatusDraft), string(procurementDomain.PurchaseOrderStatusNeedsReview)}
	var rows []openOrderRow
	err = tx.Raw(`
		SELECT li.sku_id,
			COALESCE(SUM(li.quantity * li.units_per_purchased_item) FILTER (WHERE po.status IN ?), 0) AS sent,
			COALESCE(SUM(li.quantity * li.units_per_purchased_item) FILTER (WHERE po.status IN ?), 0) AS draft
		FROM purchase_order_line_items li
		JOIN purchase_orders po ON po.id = li.purchase_order_id
		WHERE li.sku_id IS NOT NULL AND li.line_type IN ? AND (po.status IN ? OR po.status IN ?)
		GROUP BY li.sku_id
	`, sent, draft, productLineTypes, sent, draft).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("purchasePlanningRepo.OpenOrderQuantities: %w", err)
	}
	out := make(map[string]planning.OpenOrderQuantity, len(rows))
	for _, row := range rows {
		out[row.SKUID] = planning.OpenOrderQuantity{Sent: row.Sent, Draft: row.Draft}
	}
	return out, nil
}

type sourcingRow struct {
	SKUID                 string    `gorm:"column:sku_id"`
	SupplierID            string    `gorm:"column:supplier_id"`
	SupplierName          string    `gorm:"column:supplier_name"`
	LeadTimeDays          *int      `gorm:"column:lead_time_days"`
	UnitCostMinorUnits    int64     `gorm:"column:unit_cost_minor_units"`
	CurrencyCode          string    `gorm:"column:currency_code"`
	UnitsPerPurchasedItem int64     `gorm:"column:units_per_purchased_item"`
	LastOrderedAt         time.Time `gorm:"column:last_ordered_at"`
}

// LatestSourcing takes each SKU's newest non-cancelled PO line that has a
// supplier -- the supplier, price and pack size it's bought in today.
func (r *PurchasePlanningRepo) LatestSourcing(ctx context.Context) (map[string]planning.Sourcing, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []sourcingRow
	err = tx.Raw(`
		SELECT DISTINCT ON (li.sku_id)
			li.sku_id, po.supplier_id, s.name AS supplier_name, s.lead_time_days,
			li.unit_cost_minor_units, po.currency_code, li.units_per_purchased_item,
			po.created_at AS last_ordered_at
		FROM purchase_order_line_items li
		JOIN purchase_orders po ON po.id = li.purchase_order_id
		JOIN suppliers s ON s.id = po.supplier_id
		WHERE li.sku_id IS NOT NULL AND li.line_type IN ? AND po.status <> ?
		ORDER BY li.sku_id, po.created_at DESC, li.created_at DESC
	`, productLineTypes, string(procurementDomain.PurchaseOrderStatusCancelled)).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("purchasePlanningRepo.LatestSourcing: %w", err)
	}
	out := make(map[string]planning.Sourcing, len(rows))
	for _, row := range rows {
		out[row.SKUID] = planning.Sourcing{
			SupplierID: row.SupplierID, SupplierName: row.SupplierName, LeadTimeDays: row.LeadTimeDays,
			UnitCostMinorUnits: row.UnitCostMinorUnits, CurrencyCode: row.CurrencyCode,
			UnitsPerPurchasedItem: row.UnitsPerPurchasedItem, LastOrderedAt: row.LastOrderedAt,
		}
	}
	return out, nil
}
