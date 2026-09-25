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

	RefundedQuantity         int64 `gorm:"column:refunded_quantity;not null;default:0"`
	RefundedAmountMinorUnits int64 `gorm:"column:refunded_amount_minor_units;not null;default:0"`

	SoldExpiryDate *time.Time `gorm:"column:sold_expiry_date;type:date"`

	CostAtSaleMinorUnits *int64 `gorm:"column:cost_at_sale_minor_units"`
	CogsMinorUnits       *int64 `gorm:"column:cogs_minor_units"`

	CreatedAt time.Time
}

func (salesOrderLineItemModel) TableName() string { return "sales_order_line_items" }

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
		RefundedQuantity: li.RefundedQuantity, RefundedAmountMinorUnits: li.RefundedAmountMinorUnits,
		SoldExpiryDate: li.SoldExpiryDate,
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
		RefundedQuantity: m.RefundedQuantity, RefundedAmountMinorUnits: m.RefundedAmountMinorUnits,
		SoldExpiryDate:       m.SoldExpiryDate,
		CostAtSaleMinorUnits: m.CostAtSaleMinorUnits, CogsMinorUnits: m.CogsMinorUnits,
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
