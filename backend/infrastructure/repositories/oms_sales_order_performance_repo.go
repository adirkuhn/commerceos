package repositories

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type productPerformanceRow struct {
	SKUID             *string `gorm:"column:sku_id"`
	SKUCode           string
	ProductName       string `gorm:"column:name"`
	QuantitySold      int64
	RevenueMinorUnits int64
	OrderCount        int64
	FirstSaleAt       time.Time
	LastSaleAt        time.Time
}

// ProductPerformanceBetween is TopProductsSince generalized: no limit, plus
// distinct order count and first/last sale date -- the Phase 0 "Product
// Performance" report. Grouped by (sku_id, sku_code, name) same as
// TopProductsSince, for the same reason (a since-renamed product's older
// and newer line items would otherwise merge into a misleading row).
func (r *SalesOrderRepo) ProductPerformanceBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]omsDomain.ProductPerformance, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []productPerformanceRow
	err = tx.Raw(`
		SELECT
			li.sku_id AS sku_id,
			li.sku_code AS sku_code,
			li.name AS name,
			SUM(li.quantity) AS quantity_sold,
			SUM(li.total_minor_units) AS revenue_minor_units,
			COUNT(DISTINCT li.sales_order_id) AS order_count,
			MIN(so.occurred_at) AS first_sale_at,
			MAX(so.occurred_at) AS last_sale_at
		FROM sales_order_line_items li
		JOIN sales_orders so ON so.id = li.sales_order_id
		WHERE so.occurred_at >= ? AND so.occurred_at < ? AND so.currency_code = ?
			AND so.status IN ?
		GROUP BY li.sku_id, li.sku_code, li.name
		ORDER BY revenue_minor_units DESC
	`, from, to, currencyCode, revenueCountingStatuses).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.ProductPerformanceBetween: %w", err)
	}
	out := make([]omsDomain.ProductPerformance, len(rows))
	for i, row := range rows {
		out[i] = omsDomain.ProductPerformance{
			SKUID: row.SKUID, SKUCode: row.SKUCode, ProductName: row.ProductName,
			QuantitySold: row.QuantitySold, RevenueMinorUnits: row.RevenueMinorUnits, OrderCount: row.OrderCount,
			FirstSaleAt: row.FirstSaleAt, LastSaleAt: row.LastSaleAt,
		}
	}
	return out, nil
}

type customerPerformanceRow struct {
	CustomerID        string
	CustomerName      string
	OrderCount        int64
	RevenueMinorUnits int64
	FirstOrderAt      time.Time
	LastOrderAt       time.Time
}

// CustomerPerformanceBetween aggregates orders per customer -- same status/
// currency filtering as RevenueSince, plus excluding guest orders
// (customer_id IS NULL, no stable identity to group them under).
// CustomerName joins the customers table for the current name rather than
// snapshotting a per-order billing name, since (unlike a SKU's code/name)
// there's no risk of "the customer since changed" making history
// unreadable -- a customer record is the canonical source of their own
// name.
func (r *SalesOrderRepo) CustomerPerformanceBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]omsDomain.CustomerPerformance, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []customerPerformanceRow
	err = tx.Raw(`
		SELECT
			so.customer_id AS customer_id,
			COALESCE(NULLIF(TRIM(CONCAT(c.first_name, ' ', c.last_name)), ''), '') AS customer_name,
			COUNT(*) AS order_count,
			COALESCE(SUM(so.total_minor_units), 0) AS revenue_minor_units,
			MIN(so.occurred_at) AS first_order_at,
			MAX(so.occurred_at) AS last_order_at
		FROM sales_orders so
		LEFT JOIN customers c ON c.id = so.customer_id
		WHERE so.customer_id IS NOT NULL
			AND so.occurred_at >= ? AND so.occurred_at < ? AND so.currency_code = ?
			AND so.status IN ?
		GROUP BY so.customer_id, c.first_name, c.last_name
		ORDER BY revenue_minor_units DESC
	`, from, to, currencyCode, revenueCountingStatuses).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.CustomerPerformanceBetween: %w", err)
	}
	out := make([]omsDomain.CustomerPerformance, len(rows))
	for i, row := range rows {
		var aov int64
		if row.OrderCount > 0 {
			aov = row.RevenueMinorUnits / row.OrderCount
		}
		out[i] = omsDomain.CustomerPerformance{
			CustomerID: row.CustomerID, CustomerName: row.CustomerName,
			OrderCount: row.OrderCount, RevenueMinorUnits: row.RevenueMinorUnits, AverageOrderValueMinorUnits: aov,
			FirstOrderAt: row.FirstOrderAt, LastOrderAt: row.LastOrderAt,
		}
	}
	return out, nil
}

type categoryPerformanceRow struct {
	CategoryID        *string
	CategoryName      string
	QuantitySold      int64
	RevenueMinorUnits int64
	OrderCount        int64
}

// CategoryPerformanceBetween joins each line item through its SKU's
// variant/product to the product's category -- LEFT JOINs throughout so a
// line item with no linked SKU, a since-deleted SKU, or a product with no
// category all still count, rolled into one COALESCE'd "Uncategorized"
// bucket rather than being silently dropped. Same status/currency
// filtering as RevenueSince.
func (r *SalesOrderRepo) CategoryPerformanceBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]omsDomain.CategoryPerformance, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []categoryPerformanceRow
	err = tx.Raw(`
		SELECT
			c.id AS category_id,
			COALESCE(c.name, 'Uncategorized') AS category_name,
			SUM(li.quantity) AS quantity_sold,
			SUM(li.total_minor_units) AS revenue_minor_units,
			COUNT(DISTINCT li.sales_order_id) AS order_count
		FROM sales_order_line_items li
		JOIN sales_orders so ON so.id = li.sales_order_id
		LEFT JOIN skus sk ON sk.id = li.sku_id
		LEFT JOIN variants v ON v.id = sk.variant_id
		LEFT JOIN products p ON p.id = v.product_id
		LEFT JOIN categories c ON c.id = p.category_id
		WHERE so.occurred_at >= ? AND so.occurred_at < ? AND so.currency_code = ?
			AND so.status IN ?
		GROUP BY c.id, c.name
		ORDER BY revenue_minor_units DESC
	`, from, to, currencyCode, revenueCountingStatuses).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.CategoryPerformanceBetween: %w", err)
	}
	out := make([]omsDomain.CategoryPerformance, len(rows))
	for i, row := range rows {
		out[i] = omsDomain.CategoryPerformance{
			CategoryID: row.CategoryID, CategoryName: row.CategoryName,
			QuantitySold: row.QuantitySold, RevenueMinorUnits: row.RevenueMinorUnits, OrderCount: row.OrderCount,
		}
	}
	return out, nil
}

type profitabilityRow struct {
	RevenueMinorUnits          int64
	CostKnownRevenueMinorUnits int64
	CostMinorUnits             int64
}

// ProfitabilityBetween aggregates revenue/COGS/profit for [from, to) --
// same status/currency filtering as RevenueSince. Cost is computed only
// for lines whose SKU has a known cost_price_minor_units (conditional
// aggregation via CASE, single pass) -- see domain/oms.ProfitabilitySummary's
// doc for why an unknown cost isn't treated as zero.
func (r *SalesOrderRepo) ProfitabilityBetween(ctx context.Context, from, to time.Time, currencyCode string) (omsDomain.ProfitabilitySummary, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return omsDomain.ProfitabilitySummary{}, err
	}
	var row profitabilityRow
	err = tx.Raw(`
		SELECT
			COALESCE(SUM(li.total_minor_units), 0) AS revenue_minor_units,
			COALESCE(SUM(CASE WHEN sk.cost_price_minor_units IS NOT NULL THEN li.total_minor_units ELSE 0 END), 0) AS cost_known_revenue_minor_units,
			COALESCE(SUM(CASE WHEN sk.cost_price_minor_units IS NOT NULL THEN li.quantity * sk.cost_price_minor_units ELSE 0 END), 0) AS cost_minor_units
		FROM sales_order_line_items li
		JOIN sales_orders so ON so.id = li.sales_order_id
		LEFT JOIN skus sk ON sk.id = li.sku_id
		WHERE so.occurred_at >= ? AND so.occurred_at < ? AND so.currency_code = ?
			AND so.status IN ?
	`, from, to, currencyCode, revenueCountingStatuses).Scan(&row).Error
	if err != nil {
		return omsDomain.ProfitabilitySummary{}, fmt.Errorf("salesOrderRepo.ProfitabilityBetween: %w", err)
	}

	summary := omsDomain.ProfitabilitySummary{
		RevenueMinorUnits: row.RevenueMinorUnits, CostKnownRevenueMinorUnits: row.CostKnownRevenueMinorUnits,
		CostMinorUnits: row.CostMinorUnits, ProfitMinorUnits: row.CostKnownRevenueMinorUnits - row.CostMinorUnits,
	}
	if summary.CostKnownRevenueMinorUnits > 0 {
		summary.MarginBasisPoints = summary.ProfitMinorUnits * 10000 / summary.CostKnownRevenueMinorUnits
	}
	return summary, nil
}

type productSalesRow struct {
	SKUID             *string `gorm:"column:sku_id"`
	SKUCode           string
	ProductName       string `gorm:"column:name"`
	QuantitySold      int64
	RevenueMinorUnits int64
}

func (r *SalesOrderRepo) TopProductsSince(ctx context.Context, since time.Time, currencyCode string, limit int) ([]omsDomain.ProductSales, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []productSalesRow
	err = tx.Raw(`
		SELECT
			li.sku_id AS sku_id,
			li.sku_code AS sku_code,
			li.name AS name,
			SUM(li.quantity) AS quantity_sold,
			SUM(li.total_minor_units) AS revenue_minor_units
		FROM sales_order_line_items li
		JOIN sales_orders so ON so.id = li.sales_order_id
		WHERE so.occurred_at >= ? AND so.currency_code = ?
			AND so.status IN ?
		GROUP BY li.sku_id, li.sku_code, li.name
		ORDER BY quantity_sold DESC
		LIMIT ?
	`, since, currencyCode, revenueCountingStatuses, limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.TopProductsSince: %w", err)
	}
	out := make([]omsDomain.ProductSales, len(rows))
	for i, row := range rows {
		out[i] = omsDomain.ProductSales{
			SKUID: row.SKUID, SKUCode: row.SKUCode, ProductName: row.ProductName,
			QuantitySold: row.QuantitySold, RevenueMinorUnits: row.RevenueMinorUnits,
		}
	}
	return out, nil
}

type profitabilityTrendRow struct {
	PeriodStart                time.Time
	RevenueMinorUnits          int64
	CostKnownRevenueMinorUnits int64
	CostMinorUnits             int64
}

// ProfitabilityTrendBetween is ProfitabilityBetween's daily/weekly/monthly
// view -- same revenue/cost join, additionally bucketed by date_trunc.
func (r *SalesOrderRepo) ProfitabilityTrendBetween(ctx context.Context, from, to time.Time, currencyCode string, granularity omsDomain.SalesTrendGranularity) ([]omsDomain.ProfitabilityPoint, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []profitabilityTrendRow
	err = tx.Raw(`
		SELECT
			date_trunc(?, so.occurred_at) AS period_start,
			COALESCE(SUM(li.total_minor_units), 0) AS revenue_minor_units,
			COALESCE(SUM(CASE WHEN sk.cost_price_minor_units IS NOT NULL THEN li.total_minor_units ELSE 0 END), 0) AS cost_known_revenue_minor_units,
			COALESCE(SUM(CASE WHEN sk.cost_price_minor_units IS NOT NULL THEN li.quantity * sk.cost_price_minor_units ELSE 0 END), 0) AS cost_minor_units
		FROM sales_order_line_items li
		JOIN sales_orders so ON so.id = li.sales_order_id
		LEFT JOIN skus sk ON sk.id = li.sku_id
		WHERE so.occurred_at >= ? AND so.occurred_at < ? AND so.currency_code = ?
			AND so.status IN ?
		GROUP BY period_start
		ORDER BY period_start ASC
	`, string(granularity), from, to, currencyCode, revenueCountingStatuses).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.ProfitabilityTrendBetween: %w", err)
	}
	points := make([]omsDomain.ProfitabilityPoint, len(rows))
	for i, row := range rows {
		profit := row.CostKnownRevenueMinorUnits - row.CostMinorUnits
		var marginBP int64
		if row.CostKnownRevenueMinorUnits > 0 {
			marginBP = profit * 10000 / row.CostKnownRevenueMinorUnits
		}
		points[i] = omsDomain.ProfitabilityPoint{
			PeriodStart: row.PeriodStart, RevenueMinorUnits: row.RevenueMinorUnits,
			CostKnownRevenueMinorUnits: row.CostKnownRevenueMinorUnits, CostMinorUnits: row.CostMinorUnits,
			ProfitMinorUnits: profit, MarginBasisPoints: marginBP,
		}
	}
	return points, nil
}
