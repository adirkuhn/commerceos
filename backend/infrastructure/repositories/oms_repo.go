package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── GORM models ──────────────────────────────────────────────────────────────

type salesOrderModel struct {
	ID              string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID        string  `gorm:"not null;type:uuid"`
	Source          string  `gorm:"not null"`
	ExternalOrderID string  `gorm:"column:external_order_id;not null"`
	ConnectionID    *string `gorm:"column:connection_id;type:uuid"`
	Number          string
	OrderKey        string `gorm:"column:order_key"`
	Status          string `gorm:"not null;default:completed"`

	WarehouseID string  `gorm:"not null;type:uuid"`
	CustomerID  *string `gorm:"type:uuid"`

	BillingFirstName  string
	BillingLastName   string
	BillingCompany    string
	BillingAddress1   string `gorm:"column:billing_address_1"`
	BillingAddress2   string `gorm:"column:billing_address_2"`
	BillingCity       string
	BillingState      string
	BillingPostcode   string
	BillingCountry    string
	BillingEmail      string
	BillingPhone      string
	ShippingFirstName string
	ShippingLastName  string
	ShippingCompany   string
	ShippingAddress1  string `gorm:"column:shipping_address_1"`
	ShippingAddress2  string `gorm:"column:shipping_address_2"`
	ShippingCity      string
	ShippingState     string
	ShippingPostcode  string
	ShippingCountry   string
	ShippingPhone     string
	CustomerNote      string

	CurrencyCode            string `gorm:"not null;default:'USD'"`
	DiscountTotalMinorUnits int64  `gorm:"not null;default:0"`
	ShippingTotalMinorUnits int64  `gorm:"not null;default:0"`
	TaxTotalMinorUnits      int64  `gorm:"not null;default:0"`
	TotalMinorUnits         int64  `gorm:"not null;default:0"`

	PaymentMethod       string
	PaymentMethodTitle  string
	ShippingMethod      string `gorm:"column:shipping_method;not null;default:''"`
	ShippingMethodTitle string `gorm:"column:shipping_method_title;not null;default:''"`
	TransactionID       string
	DatePaid            *time.Time
	DateCompleted       *time.Time

	OccurredAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time

	StockDecremented bool `gorm:"not null;default:false"`
	NeedsStockReview bool `gorm:"column:needs_stock_review;not null;default:false"`
	NeedsSKUReview   bool `gorm:"column:needs_sku_review;not null;default:false"`
	Fulfilled        bool `gorm:"not null;default:false"`
}

func (salesOrderModel) TableName() string { return "sales_orders" }

type salesOrderLineItemModel struct {
	ID           string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string  `gorm:"not null;type:uuid"`
	SalesOrderID string  `gorm:"column:sales_order_id;not null;type:uuid"`
	SKUID        *string `gorm:"column:sku_id;type:uuid"`
	Name         string
	SKUCode      string `gorm:"column:sku_code"`
	Barcode      string `gorm:"column:barcode"`
	Quantity     int64  `gorm:"not null"`
	StockStatus  string `gorm:"column:stock_status;not null;default:decremented"`

	PriceMinorUnits       int64   `gorm:"not null;default:0"`
	SubtotalMinorUnits    int64   `gorm:"not null;default:0"`
	SubtotalTaxMinorUnits int64   `gorm:"not null;default:0"`
	TotalMinorUnits       int64   `gorm:"not null;default:0"`
	TotalTaxMinorUnits    int64   `gorm:"not null;default:0"`
	TaxClassID            *string `gorm:"column:tax_class_id;type:uuid"`

	CreatedAt time.Time
}

func (salesOrderLineItemModel) TableName() string { return "sales_order_line_items" }

// ── SalesOrderRepo ───────────────────────────────────────────────────────────

type SalesOrderRepo struct{ db *gorm.DB }

func NewSalesOrderRepo(db *gorm.DB) *SalesOrderRepo { return &SalesOrderRepo{db: db} }

func (r *SalesOrderRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("salesOrderRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func fromAddressDomain(a omsDomain.Address) (firstName, lastName, company, address1, address2, city, state, postcode, country, email, phone string) {
	return a.FirstName, a.LastName, a.Company, a.Address1, a.Address2, a.City, a.State, a.Postcode, a.Country, a.Email, a.Phone
}

func salesOrderModelFromDomain(o *omsDomain.SalesOrder) salesOrderModel {
	bFirst, bLast, bCompany, bAddr1, bAddr2, bCity, bState, bPostcode, bCountry, bEmail, bPhone := fromAddressDomain(o.Billing)
	sFirst, sLast, sCompany, sAddr1, sAddr2, sCity, sState, sPostcode, sCountry, _, sPhone := fromAddressDomain(o.Shipping)
	return salesOrderModel{
		ID: o.ID, TenantID: o.TenantID, Source: string(o.Source), ExternalOrderID: o.ExternalOrderID,
		ConnectionID: o.ConnectionID, Number: o.Number, OrderKey: o.OrderKey, Status: string(o.Status),
		WarehouseID: o.WarehouseID, CustomerID: o.CustomerID,
		BillingFirstName: bFirst, BillingLastName: bLast, BillingCompany: bCompany,
		BillingAddress1: bAddr1, BillingAddress2: bAddr2, BillingCity: bCity, BillingState: bState,
		BillingPostcode: bPostcode, BillingCountry: bCountry, BillingEmail: bEmail, BillingPhone: bPhone,
		ShippingFirstName: sFirst, ShippingLastName: sLast, ShippingCompany: sCompany,
		ShippingAddress1: sAddr1, ShippingAddress2: sAddr2, ShippingCity: sCity, ShippingState: sState,
		ShippingPostcode: sPostcode, ShippingCountry: sCountry, ShippingPhone: sPhone,
		CustomerNote: o.CustomerNote, CurrencyCode: o.CurrencyCode,
		DiscountTotalMinorUnits: o.DiscountTotalMinorUnits, ShippingTotalMinorUnits: o.ShippingTotalMinorUnits,
		TaxTotalMinorUnits: o.TaxTotalMinorUnits, TotalMinorUnits: o.TotalMinorUnits,
		PaymentMethod: o.PaymentMethod, PaymentMethodTitle: o.PaymentMethodTitle,
		ShippingMethod: o.ShippingMethod, ShippingMethodTitle: o.ShippingMethodTitle, TransactionID: o.TransactionID,
		DatePaid: o.DatePaid, DateCompleted: o.DateCompleted,
		OccurredAt: o.OccurredAt, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
		StockDecremented: o.StockDecremented,
		NeedsStockReview: o.NeedsStockReview, NeedsSKUReview: o.NeedsSKUReview,
		Fulfilled: o.Fulfilled,
	}
}

func (r *SalesOrderRepo) Create(ctx context.Context, o *omsDomain.SalesOrder) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := salesOrderModelFromDomain(o)
	if err := tx.Create(&m).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "sales_orders_tenant_source_external_id_unique" {
			return omsDomain.ErrDuplicateExternalOrder
		}
		return fmt.Errorf("salesOrderRepo.Create: %w", err)
	}
	o.ID = m.ID
	return nil
}

// TrySetStockDecremented atomically flips StockDecremented from `from` to
// `to`, conditioned on the row's current value still being `from` -- the
// DB-level guard against the double-decrement race two concurrent callers
// of UpdateSalesOrderUseCase could otherwise hit (both reading
// StockDecremented=false via FindByID before either's write lands). Returns
// false (not an error) when the row's value had already moved on -- the
// caller's job is done already, by whoever won the race.
func (r *SalesOrderRepo) TrySetStockDecremented(ctx context.Context, id string, from, to bool) (bool, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return false, err
	}
	res := tx.Model(&salesOrderModel{}).
		Where("id = ? AND stock_decremented = ?", id, from).
		Updates(map[string]any{"stock_decremented": to, "updated_at": time.Now()})
	if res.Error != nil {
		return false, fmt.Errorf("salesOrderRepo.TrySetStockDecremented: %w", res.Error)
	}
	return res.RowsAffected > 0, nil
}

func (r *SalesOrderRepo) FindByID(ctx context.Context, id string) (*omsDomain.SalesOrder, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m salesOrderModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, omsDomain.ErrSalesOrderNotFound
		}
		return nil, fmt.Errorf("salesOrderRepo.FindByID: %w", err)
	}
	return toSalesOrderDomain(m), nil
}

func (r *SalesOrderRepo) FindByExternalID(ctx context.Context, source omsDomain.SalesOrderSource, externalOrderID string) (*omsDomain.SalesOrder, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m salesOrderModel
	err = tx.Where("source = ? AND external_order_id = ?", string(source), externalOrderID).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("salesOrderRepo.FindByExternalID: %w", err)
	}
	return toSalesOrderDomain(m), nil
}

// salesOrderOrderClause maps a filter's SortBy/SortDir to a safe ORDER BY
// clause via a closed switch over omsDomain.SalesOrderSortField -- never
// interpolates either value directly, so an invalid/unknown field or
// direction can't reach raw SQL. Unset/unknown SortBy defaults to
// occurred_at; unset/unknown SortDir defaults to desc (newest first).
func salesOrderOrderClause(filter omsDomain.ListSalesOrdersFilter) string {
	dir := "desc"
	if filter.SortDir == omsDomain.SortAsc {
		dir = "asc"
	}
	switch filter.SortBy {
	case omsDomain.SalesOrderSortCustomer:
		return "billing_last_name " + dir + ", billing_first_name " + dir
	case omsDomain.SalesOrderSortSource:
		return "source " + dir
	case omsDomain.SalesOrderSortStatus:
		return "status " + dir
	case omsDomain.SalesOrderSortTotal:
		return "total_minor_units " + dir
	default:
		return "occurred_at " + dir
	}
}

func (r *SalesOrderRepo) List(ctx context.Context, filter omsDomain.ListSalesOrdersFilter) ([]omsDomain.SalesOrder, int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, 0, err
	}
	q := tx.Model(&salesOrderModel{})
	if filter.NeedsStockReview != nil {
		q = q.Where("needs_stock_review = ?", *filter.NeedsStockReview)
	}
	if filter.NeedsSKUReview != nil {
		q = q.Where("needs_sku_review = ?", *filter.NeedsSKUReview)
	}
	if len(filter.Source) > 0 {
		sources := make([]string, len(filter.Source))
		for i, s := range filter.Source {
			sources[i] = string(s)
		}
		q = q.Where("source IN ?", sources)
	}
	if filter.Fulfilled != nil {
		q = q.Where("fulfilled = ?", *filter.Fulfilled)
	}
	if len(filter.Status) > 0 {
		statuses := make([]string, len(filter.Status))
		for i, s := range filter.Status {
			statuses[i] = string(s)
		}
		q = q.Where("status IN ?", statuses)
	}
	if filter.CustomerID != nil {
		q = q.Where("customer_id = ?", *filter.CustomerID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("salesOrderRepo.List: count: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 200
	}
	var rows []salesOrderModel
	if err := q.Order(salesOrderOrderClause(filter)).Limit(limit).Offset(filter.Offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("salesOrderRepo.List: %w", err)
	}
	out := make([]omsDomain.SalesOrder, len(rows))
	for i, row := range rows {
		out[i] = *toSalesOrderDomain(row)
	}
	return out, total, nil
}

func (r *SalesOrderRepo) ListByConnection(ctx context.Context, connectionID string) ([]omsDomain.SalesOrder, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []salesOrderModel
	if err := tx.Where("connection_id = ?", connectionID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("salesOrderRepo.ListByConnection: %w", err)
	}
	out := make([]omsDomain.SalesOrder, len(rows))
	for i, row := range rows {
		out[i] = *toSalesOrderDomain(row)
	}
	return out, nil
}

func toSalesOrderDomain(m salesOrderModel) *omsDomain.SalesOrder {
	return &omsDomain.SalesOrder{
		ID: m.ID, TenantID: m.TenantID, Source: omsDomain.SalesOrderSource(m.Source),
		ExternalOrderID: m.ExternalOrderID, ConnectionID: m.ConnectionID, Number: m.Number, OrderKey: m.OrderKey,
		Status: omsDomain.SalesOrderStatus(m.Status), WarehouseID: m.WarehouseID, CustomerID: m.CustomerID,
		Billing: omsDomain.Address{
			FirstName: m.BillingFirstName, LastName: m.BillingLastName, Company: m.BillingCompany,
			Address1: m.BillingAddress1, Address2: m.BillingAddress2, City: m.BillingCity, State: m.BillingState,
			Postcode: m.BillingPostcode, Country: m.BillingCountry, Email: m.BillingEmail, Phone: m.BillingPhone,
		},
		Shipping: omsDomain.Address{
			FirstName: m.ShippingFirstName, LastName: m.ShippingLastName, Company: m.ShippingCompany,
			Address1: m.ShippingAddress1, Address2: m.ShippingAddress2, City: m.ShippingCity, State: m.ShippingState,
			Postcode: m.ShippingPostcode, Country: m.ShippingCountry, Phone: m.ShippingPhone,
		},
		CustomerNote: m.CustomerNote, CurrencyCode: m.CurrencyCode,
		DiscountTotalMinorUnits: m.DiscountTotalMinorUnits, ShippingTotalMinorUnits: m.ShippingTotalMinorUnits,
		TaxTotalMinorUnits: m.TaxTotalMinorUnits, TotalMinorUnits: m.TotalMinorUnits,
		PaymentMethod: m.PaymentMethod, PaymentMethodTitle: m.PaymentMethodTitle,
		ShippingMethod: m.ShippingMethod, ShippingMethodTitle: m.ShippingMethodTitle, TransactionID: m.TransactionID,
		DatePaid: m.DatePaid, DateCompleted: m.DateCompleted,
		OccurredAt: m.OccurredAt, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
		StockDecremented: m.StockDecremented,
		NeedsStockReview: m.NeedsStockReview, NeedsSKUReview: m.NeedsSKUReview,
		Fulfilled: m.Fulfilled,
	}
}

func (r *SalesOrderRepo) Search(ctx context.Context, query string, limit int) ([]omsDomain.SalesOrder, error) {
	if query == "" {
		return nil, nil
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 200
	}
	pattern := "%" + query + "%"
	var rows []salesOrderModel
	err = tx.Where(
		"unaccent(number) ILIKE unaccent(?) OR "+
			"similarity(immutable_unaccent(number), immutable_unaccent(?)) > ? OR "+
			"unaccent(billing_first_name || ' ' || billing_last_name) ILIKE unaccent(?) OR "+
			"similarity(immutable_unaccent(billing_first_name || ' ' || billing_last_name), immutable_unaccent(?)) > ? OR "+
			"billing_email ILIKE ?",
		pattern, query, fuzzySearchSimilarityThreshold,
		pattern, query, fuzzySearchSimilarityThreshold,
		pattern,
	).Clauses(clause.OrderBy{
		Expression: clause.Expr{
			SQL: "GREATEST(" +
				"similarity(immutable_unaccent(number), immutable_unaccent(?)), " +
				"similarity(immutable_unaccent(billing_first_name || ' ' || billing_last_name), immutable_unaccent(?))" +
				") DESC, occurred_at desc",
			Vars: []interface{}{query, query},
		},
	}).Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.Search: %w", err)
	}
	out := make([]omsDomain.SalesOrder, len(rows))
	for i, row := range rows {
		out[i] = *toSalesOrderDomain(row)
	}
	return out, nil
}

func (r *SalesOrderRepo) Update(ctx context.Context, id string, fields map[string]any) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Model(&salesOrderModel{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return fmt.Errorf("salesOrderRepo.Update: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return omsDomain.ErrSalesOrderNotFound
	}
	return nil
}

func (r *SalesOrderRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Where("id = ?", id).Delete(&salesOrderModel{})
	if res.Error != nil {
		return fmt.Errorf("salesOrderRepo.Delete: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return omsDomain.ErrSalesOrderNotFound
	}
	return nil
}

// revenueCountingStatuses mirrors omsDomain.SalesOrderStatus.ReducesStock's
// set exactly -- a status that never reduced stock (pending, cancelled,
// refunded, failed) was never real revenue either. Kept as a package-level
// slice (rather than calling ReducesStock per-row) since these three
// queries filter in SQL, not in Go.
var revenueCountingStatuses = []string{
	string(omsDomain.SalesOrderStatusOnHold),
	string(omsDomain.SalesOrderStatusProcessing),
	string(omsDomain.SalesOrderStatusCompleted),
}

func (r *SalesOrderRepo) RevenueSince(ctx context.Context, since time.Time, currencyCode string) (int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return 0, err
	}
	var total int64
	err = tx.Model(&salesOrderModel{}).
		Where("occurred_at >= ? AND currency_code = ? AND status IN ?", since, currencyCode, revenueCountingStatuses).
		Select("COALESCE(SUM(total_minor_units), 0)").Scan(&total).Error
	if err != nil {
		return 0, fmt.Errorf("salesOrderRepo.RevenueSince: %w", err)
	}
	return total, nil
}

type channelRevenueRow struct {
	Source          string
	TotalMinorUnits int64
}

func (r *SalesOrderRepo) RevenueByChannelSince(ctx context.Context, since time.Time, currencyCode string) ([]omsDomain.ChannelRevenue, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []channelRevenueRow
	err = tx.Model(&salesOrderModel{}).
		Where("occurred_at >= ? AND currency_code = ? AND status IN ?", since, currencyCode, revenueCountingStatuses).
		Select("source, COALESCE(SUM(total_minor_units), 0) AS total_minor_units").
		Group("source").Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.RevenueByChannelSince: %w", err)
	}
	out := make([]omsDomain.ChannelRevenue, len(rows))
	for i, row := range rows {
		out[i] = omsDomain.ChannelRevenue{Source: omsDomain.SalesOrderSource(row.Source), TotalMinorUnits: row.TotalMinorUnits}
	}
	return out, nil
}

type salesSummaryRow struct {
	OrderCount        int64
	RevenueMinorUnits int64
}

// SalesSummaryBetween aggregates order count/revenue from sales_orders and
// units sold from sales_order_line_items in two queries (a line-item join
// would multiply-count orders with several lines against the order-level
// total) -- same status/currency filtering as RevenueSince.
func (r *SalesOrderRepo) SalesSummaryBetween(ctx context.Context, from, to time.Time, currencyCode string) (omsDomain.SalesSummary, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return omsDomain.SalesSummary{}, err
	}
	var summary salesSummaryRow
	err = tx.Model(&salesOrderModel{}).
		Where("occurred_at >= ? AND occurred_at < ? AND currency_code = ? AND status IN ?", from, to, currencyCode, revenueCountingStatuses).
		Select("COUNT(*) AS order_count, COALESCE(SUM(total_minor_units), 0) AS revenue_minor_units").
		Scan(&summary).Error
	if err != nil {
		return omsDomain.SalesSummary{}, fmt.Errorf("salesOrderRepo.SalesSummaryBetween: %w", err)
	}

	var unitsSold int64
	err = tx.Raw(`
		SELECT COALESCE(SUM(li.quantity), 0)
		FROM sales_order_line_items li
		JOIN sales_orders so ON so.id = li.sales_order_id
		WHERE so.occurred_at >= ? AND so.occurred_at < ? AND so.currency_code = ?
			-- keep in sync with revenueCountingStatuses above
			AND so.status IN ('on-hold', 'processing', 'completed')
	`, from, to, currencyCode).Scan(&unitsSold).Error
	if err != nil {
		return omsDomain.SalesSummary{}, fmt.Errorf("salesOrderRepo.SalesSummaryBetween: %w", err)
	}

	result := omsDomain.SalesSummary{
		RevenueMinorUnits: summary.RevenueMinorUnits, OrderCount: summary.OrderCount, UnitsSold: unitsSold,
	}
	if result.OrderCount > 0 {
		result.AverageOrderValueMinorUnits = result.RevenueMinorUnits / result.OrderCount
	}
	return result, nil
}

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
			-- keep in sync with revenueCountingStatuses above
			AND so.status IN ('on-hold', 'processing', 'completed')
		GROUP BY li.sku_id, li.sku_code, li.name
		ORDER BY revenue_minor_units DESC
	`, from, to, currencyCode).Scan(&rows).Error
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
			-- keep in sync with revenueCountingStatuses above
			AND so.status IN ('on-hold', 'processing', 'completed')
		GROUP BY so.customer_id, c.first_name, c.last_name
		ORDER BY revenue_minor_units DESC
	`, from, to, currencyCode).Scan(&rows).Error
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
			-- keep in sync with revenueCountingStatuses above
			AND so.status IN ('on-hold', 'processing', 'completed')
		GROUP BY c.id, c.name
		ORDER BY revenue_minor_units DESC
	`, from, to, currencyCode).Scan(&rows).Error
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
			-- keep in sync with revenueCountingStatuses above
			AND so.status IN ('on-hold', 'processing', 'completed')
	`, from, to, currencyCode).Scan(&row).Error
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
			-- keep in sync with revenueCountingStatuses above
			AND so.status IN ('on-hold', 'processing', 'completed')
		GROUP BY li.sku_id, li.sku_code, li.name
		ORDER BY quantity_sold DESC
		LIMIT ?
	`, since, currencyCode, limit).Scan(&rows).Error
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

type salesTrendBucketRow struct {
	PeriodStart       time.Time
	RevenueMinorUnits int64
	OrderCount        int64
}

type salesTrendUnitsRow struct {
	PeriodStart time.Time
	UnitsSold   int64
}

// SalesTrendBetween buckets revenue/order count (from sales_orders) and
// units sold (from sales_order_line_items) by the given granularity --
// two queries for the same multiply-counting reason as SalesSummaryBetween,
// merged in Go by truncated timestamp. Granularity is validated by the
// caller (SalesTrendGranularity.Valid()) before it ever reaches this
// parameterized date_trunc call.
func (r *SalesOrderRepo) SalesTrendBetween(ctx context.Context, from, to time.Time, currencyCode string, granularity omsDomain.SalesTrendGranularity) ([]omsDomain.SalesTrendPoint, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var buckets []salesTrendBucketRow
	err = tx.Raw(`
		SELECT date_trunc(?, occurred_at) AS period_start,
			COALESCE(SUM(total_minor_units), 0) AS revenue_minor_units,
			COUNT(*) AS order_count
		FROM sales_orders
		WHERE occurred_at >= ? AND occurred_at < ? AND currency_code = ?
			-- keep in sync with revenueCountingStatuses above
			AND status IN ('on-hold', 'processing', 'completed')
		GROUP BY period_start
		ORDER BY period_start ASC
	`, string(granularity), from, to, currencyCode).Scan(&buckets).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.SalesTrendBetween: %w", err)
	}

	var unitsRows []salesTrendUnitsRow
	err = tx.Raw(`
		SELECT date_trunc(?, so.occurred_at) AS period_start,
			COALESCE(SUM(li.quantity), 0) AS units_sold
		FROM sales_order_line_items li
		JOIN sales_orders so ON so.id = li.sales_order_id
		WHERE so.occurred_at >= ? AND so.occurred_at < ? AND so.currency_code = ?
			-- keep in sync with revenueCountingStatuses above
			AND so.status IN ('on-hold', 'processing', 'completed')
		GROUP BY period_start
	`, string(granularity), from, to, currencyCode).Scan(&unitsRows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.SalesTrendBetween: %w", err)
	}
	unitsByPeriod := make(map[int64]int64, len(unitsRows))
	for _, u := range unitsRows {
		unitsByPeriod[u.PeriodStart.UTC().Unix()] = u.UnitsSold
	}

	points := make([]omsDomain.SalesTrendPoint, len(buckets))
	for i, b := range buckets {
		points[i] = omsDomain.SalesTrendPoint{
			PeriodStart: b.PeriodStart, RevenueMinorUnits: b.RevenueMinorUnits, OrderCount: b.OrderCount,
			UnitsSold: unitsByPeriod[b.PeriodStart.UTC().Unix()],
		}
	}
	return points, nil
}

type weekdaySalesRow struct {
	Weekday           int
	RevenueMinorUnits int64
	OrderCount        int64
}

// SalesByWeekdayBetween is the "seasonality" half of the Sales Trends
// report -- sums revenue/orders across every occurrence of each weekday
// within [from, to). Postgres's EXTRACT(DOW) numbers Sunday=0..Saturday=6,
// same as time.Weekday, so no remapping is needed.
func (r *SalesOrderRepo) SalesByWeekdayBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]omsDomain.WeekdaySales, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []weekdaySalesRow
	err = tx.Raw(`
		SELECT EXTRACT(DOW FROM occurred_at)::int AS weekday,
			COALESCE(SUM(total_minor_units), 0) AS revenue_minor_units,
			COUNT(*) AS order_count
		FROM sales_orders
		WHERE occurred_at >= ? AND occurred_at < ? AND currency_code = ?
			-- keep in sync with revenueCountingStatuses above
			AND status IN ('on-hold', 'processing', 'completed')
		GROUP BY weekday
		ORDER BY weekday ASC
	`, from, to, currencyCode).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.SalesByWeekdayBetween: %w", err)
	}
	result := make([]omsDomain.WeekdaySales, len(rows))
	for i, row := range rows {
		result[i] = omsDomain.WeekdaySales{
			Weekday: time.Weekday(row.Weekday), RevenueMinorUnits: row.RevenueMinorUnits, OrderCount: row.OrderCount,
		}
	}
	return result, nil
}

// cohortCustomersCTE is shared by CohortSizesBetween and
// CohortRetentionBetween: it defines each customer's cohort as the
// calendar month of their all-time first order (not just their first
// order within [from, to)), then keeps only the customers whose cohort
// month falls inside the queried window -- a customer who already existed
// before the window isn't a new cohort for this report.
const cohortCustomersCTE = `
	WITH first_orders AS (
		SELECT customer_id, date_trunc('month', MIN(occurred_at)) AS cohort_month
		FROM sales_orders
		WHERE customer_id IS NOT NULL AND currency_code = ?
			-- keep in sync with revenueCountingStatuses above
			AND status IN ('on-hold', 'processing', 'completed')
		GROUP BY customer_id
	),
	cohort_customers AS (
		SELECT customer_id, cohort_month FROM first_orders
		WHERE cohort_month >= ? AND cohort_month < ?
	)
`

type cohortSizeRow struct {
	CohortMonth   time.Time
	CustomerCount int64
}

// CohortSizesBetween returns the number of customers newly acquired in
// each calendar month within [from, to) -- see cohortCustomersCTE's doc
// for what "newly acquired" means.
func (r *SalesOrderRepo) CohortSizesBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]omsDomain.CohortSize, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []cohortSizeRow
	err = tx.Raw(cohortCustomersCTE+`
		SELECT cohort_month, COUNT(DISTINCT customer_id) AS customer_count
		FROM cohort_customers
		GROUP BY cohort_month
		ORDER BY cohort_month ASC
	`, currencyCode, from, to).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.CohortSizesBetween: %w", err)
	}
	out := make([]omsDomain.CohortSize, len(rows))
	for i, row := range rows {
		out[i] = omsDomain.CohortSize{CohortMonth: row.CohortMonth, CustomerCount: row.CustomerCount}
	}
	return out, nil
}

type cohortRetentionRow struct {
	CohortMonth       time.Time
	MonthsSinceCohort int64
	RetainedCustomers int64
	RevenueMinorUnits int64
}

// CohortRetentionBetween returns one row per (cohort month, months since
// acquisition) for every order placed by a cohort customer up to `to` --
// including orders after the acquisition month itself, which is how a
// cohort's retention across later months is measured. months_since_cohort
// 0 is the acquisition month.
func (r *SalesOrderRepo) CohortRetentionBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]omsDomain.CohortRetentionPoint, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []cohortRetentionRow
	err = tx.Raw(cohortCustomersCTE+`
		SELECT
			cc.cohort_month AS cohort_month,
			(DATE_PART('year', date_trunc('month', so.occurred_at)) - DATE_PART('year', cc.cohort_month)) * 12
				+ (DATE_PART('month', date_trunc('month', so.occurred_at)) - DATE_PART('month', cc.cohort_month)) AS months_since_cohort,
			COUNT(DISTINCT so.customer_id) AS retained_customers,
			COALESCE(SUM(so.total_minor_units), 0) AS revenue_minor_units
		FROM cohort_customers cc
		JOIN sales_orders so ON so.customer_id = cc.customer_id
		WHERE so.occurred_at < ? AND so.currency_code = ?
			-- keep in sync with revenueCountingStatuses above
			AND so.status IN ('on-hold', 'processing', 'completed')
		GROUP BY cc.cohort_month, months_since_cohort
		ORDER BY cc.cohort_month ASC, months_since_cohort ASC
	`, currencyCode, from, to, to, currencyCode).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.CohortRetentionBetween: %w", err)
	}
	out := make([]omsDomain.CohortRetentionPoint, len(rows))
	for i, row := range rows {
		out[i] = omsDomain.CohortRetentionPoint{
			CohortMonth: row.CohortMonth, MonthsSinceCohort: row.MonthsSinceCohort,
			RetainedCustomers: row.RetainedCustomers, RevenueMinorUnits: row.RevenueMinorUnits,
		}
	}
	return out, nil
}

type skuSetSalesRow struct {
	RevenueMinorUnits int64
	UnitsSold         int64
	OrderCount        int64
}

// SKUSetSalesBetween aggregates revenue/units/distinct-order-count for an
// arbitrary set of SKUs within [from, to) -- the Phase 0 "Promotion
// Analytics" report's before/during building block. An empty skuIDs
// returns a zero-value summary without querying (an empty SQL IN () is
// invalid).
func (r *SalesOrderRepo) SKUSetSalesBetween(ctx context.Context, skuIDs []string, from, to time.Time, currencyCode string) (omsDomain.SKUSetSalesSummary, error) {
	if len(skuIDs) == 0 {
		return omsDomain.SKUSetSalesSummary{}, nil
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return omsDomain.SKUSetSalesSummary{}, err
	}
	var row skuSetSalesRow
	err = tx.Raw(`
		SELECT
			COALESCE(SUM(li.total_minor_units), 0) AS revenue_minor_units,
			COALESCE(SUM(li.quantity), 0) AS units_sold,
			COUNT(DISTINCT li.sales_order_id) AS order_count
		FROM sales_order_line_items li
		JOIN sales_orders so ON so.id = li.sales_order_id
		WHERE li.sku_id IN ? AND so.occurred_at >= ? AND so.occurred_at < ? AND so.currency_code = ?
			-- keep in sync with revenueCountingStatuses above
			AND so.status IN ('on-hold', 'processing', 'completed')
	`, skuIDs, from, to, currencyCode).Scan(&row).Error
	if err != nil {
		return omsDomain.SKUSetSalesSummary{}, fmt.Errorf("salesOrderRepo.SKUSetSalesBetween: %w", err)
	}
	return omsDomain.SKUSetSalesSummary{
		RevenueMinorUnits: row.RevenueMinorUnits, UnitsSold: row.UnitsSold, OrderCount: row.OrderCount,
	}, nil
}

type orderSKURow struct {
	OrderID string `gorm:"column:order_id"`
	SKUID   string `gorm:"column:sku_id"`
}

// ListOrderSKUsBetween returns one (order_id, sku_id) row per distinct SKU
// in each qualifying order -- deliberately flat rather than a Postgres
// array_agg, since scanning a Postgres array column into a Go []string
// isn't supported by GORM's plain database/sql Scan without an extra
// array-type dependency; grouping into omsDomain.OrderSKUs happens here in
// Go instead, order preserved via a parallel slice.
func (r *SalesOrderRepo) ListOrderSKUsBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]omsDomain.OrderSKUs, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []orderSKURow
	err = tx.Raw(`
		SELECT DISTINCT so.id AS order_id, li.sku_id AS sku_id
		FROM sales_order_line_items li
		JOIN sales_orders so ON so.id = li.sales_order_id
		WHERE li.sku_id IS NOT NULL
			AND so.occurred_at >= ? AND so.occurred_at < ? AND so.currency_code = ?
			-- keep in sync with revenueCountingStatuses above
			AND so.status IN ('on-hold', 'processing', 'completed')
		ORDER BY so.id
	`, from, to, currencyCode).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.ListOrderSKUsBetween: %w", err)
	}

	var orders []omsDomain.OrderSKUs
	var current *omsDomain.OrderSKUs
	for _, row := range rows {
		if current == nil || current.OrderID != row.OrderID {
			orders = append(orders, omsDomain.OrderSKUs{OrderID: row.OrderID})
			current = &orders[len(orders)-1]
		}
		current.SKUIDs = append(current.SKUIDs, row.SKUID)
	}
	return orders, nil
}

// RevenueByChannelBetween is RevenueByChannelSince's [from, to) counterpart
// for the Reports "Revenue by Channel" report -- same grouping, bounded
// window instead of an open-ended "since".
func (r *SalesOrderRepo) RevenueByChannelBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]omsDomain.ChannelRevenue, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []channelRevenueRow
	err = tx.Model(&salesOrderModel{}).
		Where("occurred_at >= ? AND occurred_at < ? AND currency_code = ? AND status IN ?", from, to, currencyCode, revenueCountingStatuses).
		Select("source, COALESCE(SUM(total_minor_units), 0) AS total_minor_units").
		Group("source").Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.RevenueByChannelBetween: %w", err)
	}
	out := make([]omsDomain.ChannelRevenue, len(rows))
	for i, row := range rows {
		out[i] = omsDomain.ChannelRevenue{Source: omsDomain.SalesOrderSource(row.Source), TotalMinorUnits: row.TotalMinorUnits}
	}
	return out, nil
}

type channelRevenueTrendRow struct {
	PeriodStart       time.Time
	Source            string
	RevenueMinorUnits int64
}

// RevenueByChannelTrendBetween is the daily/weekly/monthly view of the
// Reports "Revenue by Channel" report -- same date_trunc bucketing as
// SalesTrendBetween, additionally grouped by source.
func (r *SalesOrderRepo) RevenueByChannelTrendBetween(ctx context.Context, from, to time.Time, currencyCode string, granularity omsDomain.SalesTrendGranularity) ([]omsDomain.ChannelRevenuePoint, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []channelRevenueTrendRow
	err = tx.Raw(`
		SELECT date_trunc(?, occurred_at) AS period_start,
			source,
			COALESCE(SUM(total_minor_units), 0) AS revenue_minor_units
		FROM sales_orders
		WHERE occurred_at >= ? AND occurred_at < ? AND currency_code = ?
			-- keep in sync with revenueCountingStatuses above
			AND status IN ('on-hold', 'processing', 'completed')
		GROUP BY period_start, source
		ORDER BY period_start ASC
	`, string(granularity), from, to, currencyCode).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("salesOrderRepo.RevenueByChannelTrendBetween: %w", err)
	}
	points := make([]omsDomain.ChannelRevenuePoint, len(rows))
	for i, row := range rows {
		points[i] = omsDomain.ChannelRevenuePoint{
			PeriodStart: row.PeriodStart, Source: omsDomain.SalesOrderSource(row.Source), RevenueMinorUnits: row.RevenueMinorUnits,
		}
	}
	return points, nil
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
			-- keep in sync with revenueCountingStatuses above
			AND so.status IN ('on-hold', 'processing', 'completed')
		GROUP BY period_start
		ORDER BY period_start ASC
	`, string(granularity), from, to, currencyCode).Scan(&rows).Error
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

// ── SalesOrderLineItemRepo ───────────────────────────────────────────────────

type SalesOrderLineItemRepo struct{ db *gorm.DB }

func NewSalesOrderLineItemRepo(db *gorm.DB) *SalesOrderLineItemRepo {
	return &SalesOrderLineItemRepo{db: db}
}

func (r *SalesOrderLineItemRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("salesOrderLineItemRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *SalesOrderLineItemRepo) Create(ctx context.Context, li *omsDomain.SalesOrderLineItem) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	stockStatus := li.StockStatus
	if stockStatus == "" {
		stockStatus = omsDomain.StockStatusDecremented
	}
	m := salesOrderLineItemModel{
		TenantID: li.TenantID, SalesOrderID: li.SalesOrderID, SKUID: li.SKUID,
		Name: li.Name, SKUCode: li.SKUCode, Barcode: li.Barcode, Quantity: li.Quantity,
		StockStatus:     string(stockStatus),
		PriceMinorUnits: li.PriceMinorUnits, SubtotalMinorUnits: li.SubtotalMinorUnits,
		SubtotalTaxMinorUnits: li.SubtotalTaxMinorUnits, TotalMinorUnits: li.TotalMinorUnits,
		TotalTaxMinorUnits: li.TotalTaxMinorUnits, TaxClassID: li.TaxClassID, CreatedAt: li.CreatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("salesOrderLineItemRepo.Create: %w", err)
	}
	li.ID = m.ID
	return nil
}

func (r *SalesOrderLineItemRepo) FindByID(ctx context.Context, id string) (*omsDomain.SalesOrderLineItem, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m salesOrderLineItemModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, omsDomain.ErrSalesOrderLineItemNotFound
		}
		return nil, fmt.Errorf("salesOrderLineItemRepo.FindByID: %w", err)
	}
	return toSalesOrderLineItemDomain(m), nil
}

func (r *SalesOrderLineItemRepo) Update(ctx context.Context, id string, fields map[string]any) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Model(&salesOrderLineItemModel{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return fmt.Errorf("salesOrderLineItemRepo.Update: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return omsDomain.ErrSalesOrderLineItemNotFound
	}
	return nil
}

func toSalesOrderLineItemDomain(m salesOrderLineItemModel) *omsDomain.SalesOrderLineItem {
	return &omsDomain.SalesOrderLineItem{
		ID: m.ID, TenantID: m.TenantID, SalesOrderID: m.SalesOrderID, SKUID: m.SKUID,
		Name: m.Name, SKUCode: m.SKUCode, Barcode: m.Barcode, Quantity: m.Quantity,
		StockStatus:     omsDomain.StockStatus(m.StockStatus),
		PriceMinorUnits: m.PriceMinorUnits, SubtotalMinorUnits: m.SubtotalMinorUnits,
		SubtotalTaxMinorUnits: m.SubtotalTaxMinorUnits, TotalMinorUnits: m.TotalMinorUnits,
		TotalTaxMinorUnits: m.TotalTaxMinorUnits, TaxClassID: m.TaxClassID, CreatedAt: m.CreatedAt,
	}
}

func (r *SalesOrderLineItemRepo) ListBySalesOrder(ctx context.Context, salesOrderID string) ([]omsDomain.SalesOrderLineItem, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []salesOrderLineItemModel
	if err := tx.Where("sales_order_id = ?", salesOrderID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("salesOrderLineItemRepo.ListBySalesOrder: %w", err)
	}
	out := make([]omsDomain.SalesOrderLineItem, len(rows))
	for i, m := range rows {
		out[i] = *toSalesOrderLineItemDomain(m)
	}
	return out, nil
}

func (r *SalesOrderLineItemRepo) DeleteBySalesOrder(ctx context.Context, salesOrderID string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("sales_order_id = ?", salesOrderID).Delete(&salesOrderLineItemModel{}).Error; err != nil {
		return fmt.Errorf("salesOrderLineItemRepo.DeleteBySalesOrder: %w", err)
	}
	return nil
}

// ── SalesOrderLineItemStockMovementRepo ─────────────────────────────────────

type salesOrderLineItemStockMovementModel struct {
	ID                   string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID             string  `gorm:"not null;type:uuid"`
	SalesOrderLineItemID string  `gorm:"column:sales_order_line_item_id;not null;type:uuid"`
	StockMovementID      string  `gorm:"column:stock_movement_id;not null;type:uuid"`
	BatchID              *string `gorm:"column:batch_id;type:uuid"`
	QuantityDelta        int64   `gorm:"column:quantity_delta;not null;default:0"`
	CreatedAt            time.Time
}

func (salesOrderLineItemStockMovementModel) TableName() string {
	return "sales_order_line_item_stock_movements"
}

type SalesOrderLineItemStockMovementRepo struct{ db *gorm.DB }

func NewSalesOrderLineItemStockMovementRepo(db *gorm.DB) *SalesOrderLineItemStockMovementRepo {
	return &SalesOrderLineItemStockMovementRepo{db: db}
}

func (r *SalesOrderLineItemStockMovementRepo) Create(ctx context.Context, l *omsDomain.SalesOrderLineItemStockMovement) error {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return errors.New("salesOrderLineItemStockMovementRepo: operation called outside a transaction")
	}
	m := salesOrderLineItemStockMovementModel{
		TenantID: l.TenantID, SalesOrderLineItemID: l.SalesOrderLineItemID,
		StockMovementID: l.StockMovementID, BatchID: l.BatchID, QuantityDelta: l.QuantityDelta,
		CreatedAt: l.CreatedAt,
	}
	if err := tx.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("salesOrderLineItemStockMovementRepo.Create: %w", err)
	}
	l.ID = m.ID
	return nil
}

func (r *SalesOrderLineItemStockMovementRepo) ListByLineItem(ctx context.Context, lineItemID string) ([]omsDomain.SalesOrderLineItemStockMovement, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("salesOrderLineItemStockMovementRepo: operation called outside a transaction")
	}
	var rows []salesOrderLineItemStockMovementModel
	if err := tx.WithContext(ctx).Where("sales_order_line_item_id = ?", lineItemID).
		Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("salesOrderLineItemStockMovementRepo.ListByLineItem: %w", err)
	}
	out := make([]omsDomain.SalesOrderLineItemStockMovement, len(rows))
	for i, r := range rows {
		out[i] = omsDomain.SalesOrderLineItemStockMovement{
			ID: r.ID, TenantID: r.TenantID, SalesOrderLineItemID: r.SalesOrderLineItemID,
			StockMovementID: r.StockMovementID, BatchID: r.BatchID, QuantityDelta: r.QuantityDelta,
			CreatedAt: r.CreatedAt,
		}
	}
	return out, nil
}

func (r *SalesOrderLineItemStockMovementRepo) ExistsForLineItem(ctx context.Context, lineItemID string) (bool, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return false, errors.New("salesOrderLineItemStockMovementRepo: operation called outside a transaction")
	}
	var count int64
	if err := tx.WithContext(ctx).Model(&salesOrderLineItemStockMovementModel{}).
		Where("sales_order_line_item_id = ?", lineItemID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("salesOrderLineItemStockMovementRepo.ExistsForLineItem: %w", err)
	}
	return count > 0, nil
}
