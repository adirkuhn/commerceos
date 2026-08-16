package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

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
}

func (salesOrderModel) TableName() string { return "sales_orders" }

type salesOrderLineItemModel struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID     string `gorm:"not null;type:uuid"`
	SalesOrderID string `gorm:"column:sales_order_id;not null;type:uuid"`
	SKUID        string `gorm:"column:sku_id;not null;type:uuid"`
	Name         string
	SKUCode      string `gorm:"column:sku_code"`
	Barcode      string `gorm:"column:barcode"`
	Quantity     int64  `gorm:"not null"`

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
	}
}

func (r *SalesOrderRepo) Create(ctx context.Context, o *omsDomain.SalesOrder) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := salesOrderModelFromDomain(o)
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("salesOrderRepo.Create: %w", err)
	}
	o.ID = m.ID
	return nil
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
	}
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
	m := salesOrderLineItemModel{
		TenantID: li.TenantID, SalesOrderID: li.SalesOrderID, SKUID: li.SKUID,
		Name: li.Name, SKUCode: li.SKUCode, Barcode: li.Barcode, Quantity: li.Quantity,
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
		out[i] = omsDomain.SalesOrderLineItem{
			ID: m.ID, TenantID: m.TenantID, SalesOrderID: m.SalesOrderID, SKUID: m.SKUID,
			Name: m.Name, SKUCode: m.SKUCode, Barcode: m.Barcode, Quantity: m.Quantity,
			PriceMinorUnits: m.PriceMinorUnits, SubtotalMinorUnits: m.SubtotalMinorUnits,
			SubtotalTaxMinorUnits: m.SubtotalTaxMinorUnits, TotalMinorUnits: m.TotalMinorUnits,
			TotalTaxMinorUnits: m.TotalTaxMinorUnits, TaxClassID: m.TaxClassID, CreatedAt: m.CreatedAt,
		}
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
	ID                   string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID             string `gorm:"not null;type:uuid"`
	SalesOrderLineItemID string `gorm:"column:sales_order_line_item_id;not null;type:uuid"`
	StockMovementID      string `gorm:"column:stock_movement_id;not null;type:uuid"`
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
		StockMovementID: l.StockMovementID, CreatedAt: l.CreatedAt,
	}
	if err := tx.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("salesOrderLineItemStockMovementRepo.Create: %w", err)
	}
	l.ID = m.ID
	return nil
}
