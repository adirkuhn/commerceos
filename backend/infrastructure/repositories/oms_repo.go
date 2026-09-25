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
	SalesChannelID  *string `gorm:"column:sales_channel_id;type:uuid"`
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
	CouponCode          string `gorm:"column:coupon_code;not null;default:''"`
	TransactionID       string
	PaymentProvider     string `gorm:"column:payment_provider;not null;default:''"`
	PaymentExternalID   string `gorm:"column:payment_external_id;not null;default:''"`
	DatePaid            *time.Time
	DateCompleted       *time.Time

	OccurredAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time

	StockDecremented        bool `gorm:"not null;default:false"`
	NeedsStockReview        bool `gorm:"column:needs_stock_review;not null;default:false"`
	NeedsSKUReview          bool `gorm:"column:needs_sku_review;not null;default:false"`
	NeedsManualRefundReview bool `gorm:"column:needs_manual_refund_review;not null;default:false"`
	Fulfilled               bool `gorm:"not null;default:false"`
	Packed                  bool `gorm:"not null;default:false"`
}

func (salesOrderModel) TableName() string { return "sales_orders" }

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
		ConnectionID: o.ConnectionID, SalesChannelID: o.SalesChannelID, Number: o.Number, OrderKey: o.OrderKey, Status: string(o.Status),
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
		ShippingMethod: o.ShippingMethod, ShippingMethodTitle: o.ShippingMethodTitle, CouponCode: o.CouponCode, TransactionID: o.TransactionID,
		PaymentProvider: o.PaymentProvider, PaymentExternalID: o.PaymentExternalID,
		DatePaid: o.DatePaid, DateCompleted: o.DateCompleted,
		OccurredAt: o.OccurredAt, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
		StockDecremented: o.StockDecremented,
		NeedsStockReview: o.NeedsStockReview, NeedsSKUReview: o.NeedsSKUReview,
		NeedsManualRefundReview: o.NeedsManualRefundReview,
		Fulfilled:               o.Fulfilled,
		Packed:                  o.Packed,
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

func (r *SalesOrderRepo) FindByNumber(ctx context.Context, number string) (*omsDomain.SalesOrder, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m salesOrderModel
	err = tx.Where("number = ?", number).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("salesOrderRepo.FindByNumber: %w", err)
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

func salesOrderStatusStrings(statuses []omsDomain.SalesOrderStatus) []string {
	out := make([]string, len(statuses))
	for i, s := range statuses {
		out[i] = string(s)
	}
	return out
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
	if filter.Packed != nil {
		q = q.Where("packed = ?", *filter.Packed)
	}
	if filter.AwaitingPicking {
		q = q.Where("fulfilled = false").
			Where("status IN ?", salesOrderStatusStrings(omsDomain.PickableStatuses())).
			Where("source <> ?", string(omsDomain.SalesOrderSourcePOS))
	}
	if filter.AwaitingPacking {
		q = q.Where("packed = false").
			Where("status IN ?", salesOrderStatusStrings(omsDomain.PickableStatuses())).
			Where("source <> ?", string(omsDomain.SalesOrderSourcePOS))
	}
	if len(filter.Status) > 0 {
		q = q.Where("status IN ?", salesOrderStatusStrings(filter.Status))
	}
	if filter.CustomerID != nil {
		q = q.Where("customer_id = ?", *filter.CustomerID)
	}
	if filter.SalesChannelID != nil {
		q = q.Where("sales_channel_id = ?", *filter.SalesChannelID)
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
		ExternalOrderID: m.ExternalOrderID, ConnectionID: m.ConnectionID, SalesChannelID: m.SalesChannelID, Number: m.Number, OrderKey: m.OrderKey,
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
		ShippingMethod: m.ShippingMethod, ShippingMethodTitle: m.ShippingMethodTitle, CouponCode: m.CouponCode, TransactionID: m.TransactionID,
		PaymentProvider: m.PaymentProvider, PaymentExternalID: m.PaymentExternalID,
		DatePaid: m.DatePaid, DateCompleted: m.DateCompleted,
		OccurredAt: m.OccurredAt, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
		StockDecremented: m.StockDecremented,
		NeedsStockReview: m.NeedsStockReview, NeedsSKUReview: m.NeedsSKUReview,
		NeedsManualRefundReview: m.NeedsManualRefundReview,
		Fulfilled:               m.Fulfilled,
		Packed:                  m.Packed,
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
