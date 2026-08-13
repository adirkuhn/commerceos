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
	ID              string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID        string `gorm:"not null;type:uuid"`
	Source          string `gorm:"not null"`
	ExternalOrderID string `gorm:"column:external_order_id;not null"`
	WarehouseID     string `gorm:"not null;type:uuid"`
	CurrencyCode    string `gorm:"not null;default:'USD'"`
	TotalMinorUnits int64  `gorm:"not null;default:0"`
	OccurredAt      time.Time
	CreatedAt       time.Time
}

func (salesOrderModel) TableName() string { return "sales_orders" }

type salesOrderLineItemModel struct {
	ID                  string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID            string `gorm:"not null;type:uuid"`
	SalesOrderID        string `gorm:"column:sales_order_id;not null;type:uuid"`
	SKUID               string `gorm:"column:sku_id;not null;type:uuid"`
	Quantity            int64  `gorm:"not null"`
	UnitPriceMinorUnits int64  `gorm:"not null;default:0"`
	CreatedAt           time.Time
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

func (r *SalesOrderRepo) Create(ctx context.Context, o *omsDomain.SalesOrder) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := salesOrderModel{
		TenantID: o.TenantID, Source: string(o.Source), ExternalOrderID: o.ExternalOrderID,
		WarehouseID: o.WarehouseID, CurrencyCode: o.CurrencyCode, TotalMinorUnits: o.TotalMinorUnits,
		OccurredAt: o.OccurredAt, CreatedAt: o.CreatedAt,
	}
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

func toSalesOrderDomain(m salesOrderModel) *omsDomain.SalesOrder {
	return &omsDomain.SalesOrder{
		ID: m.ID, TenantID: m.TenantID, Source: omsDomain.SalesOrderSource(m.Source),
		ExternalOrderID: m.ExternalOrderID, WarehouseID: m.WarehouseID, CurrencyCode: m.CurrencyCode,
		TotalMinorUnits: m.TotalMinorUnits, OccurredAt: m.OccurredAt, CreatedAt: m.CreatedAt,
	}
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
		Quantity: li.Quantity, UnitPriceMinorUnits: li.UnitPriceMinorUnits, CreatedAt: li.CreatedAt,
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
			Quantity: m.Quantity, UnitPriceMinorUnits: m.UnitPriceMinorUnits, CreatedAt: m.CreatedAt,
		}
	}
	return out, nil
}
