package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── Supplier cost history (COM-11) ──────────────────────────────────────────

type supplierCostEntryModel struct {
	ID                       string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID                 string    `gorm:"type:uuid;not null"`
	SKUID                    string    `gorm:"column:sku_id;type:uuid;not null"`
	SupplierID               *string   `gorm:"column:supplier_id;type:uuid"`
	PurchaseOrderID          string    `gorm:"column:purchase_order_id;type:uuid;not null"`
	PurchaseOrderLineItemID  string    `gorm:"column:purchase_order_line_item_id;type:uuid;not null"`
	BatchID                  *string   `gorm:"column:batch_id;type:uuid"`
	Kind                     string    `gorm:"column:kind;not null"`
	Quantity                 int64     `gorm:"column:quantity;not null"`
	UnitCostMinorUnits       int64     `gorm:"column:unit_cost_minor_units;not null"`
	LandedUnitCostMinorUnits int64     `gorm:"column:landed_unit_cost_minor_units;not null"`
	CurrencyCode             string    `gorm:"column:currency_code;not null"`
	ReceivedAt               time.Time `gorm:"column:received_at;not null"`
	SourceKey                string    `gorm:"column:source_key;not null"`
	CreatedAt                time.Time `gorm:"column:created_at"`
}

func (supplierCostEntryModel) TableName() string { return "supplier_cost_entries" }

// SupplierCostRepo is procurementDomain.SupplierCostRepository on Postgres.
type SupplierCostRepo struct{ db *gorm.DB }

func NewSupplierCostRepo(db *gorm.DB) *SupplierCostRepo { return &SupplierCostRepo{db: db} }

var _ procurementDomain.SupplierCostRepository = (*SupplierCostRepo)(nil)

func (r *SupplierCostRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("supplierCostRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Append inserts entries; one whose (tenant, source_key) already exists is
// skipped, so replaying the same source is harmless.
func (r *SupplierCostRepo) Append(ctx context.Context, entries []procurementDomain.SupplierCostEntry) error {
	if len(entries) == 0 {
		return nil
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	models := make([]supplierCostEntryModel, len(entries))
	for i, e := range entries {
		created := e.CreatedAt
		if created.IsZero() {
			created = now
		}
		models[i] = supplierCostEntryModel{
			TenantID: e.TenantID, SKUID: e.SKUID, SupplierID: e.SupplierID,
			PurchaseOrderID: e.PurchaseOrderID, PurchaseOrderLineItemID: e.PurchaseOrderLineItemID, BatchID: e.BatchID,
			Kind: string(e.Kind), Quantity: e.Quantity, UnitCostMinorUnits: e.UnitCostMinorUnits,
			LandedUnitCostMinorUnits: e.LandedUnitCostMinorUnits, CurrencyCode: e.CurrencyCode,
			ReceivedAt: e.ReceivedAt, SourceKey: e.SourceKey, CreatedAt: created,
		}
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "source_key"}},
		DoNothing: true,
	}).Create(&models).Error; err != nil {
		return fmt.Errorf("supplierCostRepo.Append: %w", err)
	}
	return nil
}

// ListBySKU is a SKU's whole history, oldest first.
func (r *SupplierCostRepo) ListBySKU(ctx context.Context, skuID string) ([]procurementDomain.SupplierCostEntry, error) {
	return r.list(ctx, "sku_id = ?", skuID)
}

// ListByLine is one purchase order line's rows, oldest first.
func (r *SupplierCostRepo) ListByLine(ctx context.Context, lineItemID string) ([]procurementDomain.SupplierCostEntry, error) {
	return r.list(ctx, "purchase_order_line_item_id = ?", lineItemID)
}

func (r *SupplierCostRepo) list(ctx context.Context, where string, arg string) ([]procurementDomain.SupplierCostEntry, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var models []supplierCostEntryModel
	if err := tx.Where(where, arg).Order("received_at, created_at, id").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("supplierCostRepo.list: %w", err)
	}
	out := make([]procurementDomain.SupplierCostEntry, len(models))
	for i, m := range models {
		out[i] = procurementDomain.SupplierCostEntry{
			ID: m.ID, TenantID: m.TenantID, SKUID: m.SKUID, SupplierID: m.SupplierID,
			PurchaseOrderID: m.PurchaseOrderID, PurchaseOrderLineItemID: m.PurchaseOrderLineItemID, BatchID: m.BatchID,
			Kind: procurementDomain.SupplierCostKind(m.Kind), Quantity: m.Quantity, UnitCostMinorUnits: m.UnitCostMinorUnits,
			LandedUnitCostMinorUnits: m.LandedUnitCostMinorUnits, CurrencyCode: m.CurrencyCode,
			ReceivedAt: m.ReceivedAt, SourceKey: m.SourceKey, CreatedAt: m.CreatedAt,
		}
	}
	return out, nil
}
