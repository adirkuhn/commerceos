package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── SKU cost price history repo ─────────────────────────────────────────────

const defaultCostPriceHistoryLimit = 90

type skuCostPriceHistoryModel struct {
	ID                         string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID                   string    `gorm:"not null;type:uuid"`
	SKUID                      string    `gorm:"column:sku_id;not null;type:uuid"`
	CostPriceMinorUnits        int64     `gorm:"not null"`
	RecommendedPriceMinorUnits int64     `gorm:"not null"`
	OccurredOn                 time.Time `gorm:"type:date;not null"`
	CreatedAt                  time.Time
}

func (skuCostPriceHistoryModel) TableName() string { return "sku_cost_price_history" }

type SKUCostPriceHistoryRepo struct{ db *gorm.DB }

func NewSKUCostPriceHistoryRepo(db *gorm.DB) *SKUCostPriceHistoryRepo {
	return &SKUCostPriceHistoryRepo{db: db}
}

func (r *SKUCostPriceHistoryRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("pms: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *SKUCostPriceHistoryRepo) Upsert(ctx context.Context, h *pmsDomain.SKUCostPriceHistory) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := skuCostPriceHistoryModel{
		TenantID: h.TenantID, SKUID: h.SKUID,
		CostPriceMinorUnits: h.CostPriceMinorUnits, RecommendedPriceMinorUnits: h.RecommendedPriceMinorUnits,
		OccurredOn: h.OccurredOn, CreatedAt: h.CreatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "sku_id"}, {Name: "occurred_on"}},
		DoUpdates: clause.AssignmentColumns([]string{"cost_price_minor_units", "recommended_price_minor_units"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("skuCostPriceHistoryRepo.Upsert: %w", err)
	}
	h.ID = m.ID
	return nil
}

func (r *SKUCostPriceHistoryRepo) ListBySKU(ctx context.Context, skuID string, limit int) ([]pmsDomain.SKUCostPriceHistory, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultCostPriceHistoryLimit
	}
	var rows []skuCostPriceHistoryModel
	if err := tx.Where("sku_id = ?", skuID).Order("occurred_on desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("skuCostPriceHistoryRepo.ListBySKU: %w", err)
	}
	history := make([]pmsDomain.SKUCostPriceHistory, len(rows))
	for i, row := range rows {
		history[i] = pmsDomain.SKUCostPriceHistory{
			ID: row.ID, TenantID: row.TenantID, SKUID: row.SKUID,
			CostPriceMinorUnits: row.CostPriceMinorUnits, RecommendedPriceMinorUnits: row.RecommendedPriceMinorUnits,
			OccurredOn: row.OccurredOn, CreatedAt: row.CreatedAt,
		}
	}
	return history, nil
}

// ── SKU cost price ledger repo ──────────────────────────────────────────────

const defaultCostPriceLedgerLimit = 200

type skuCostPriceLedgerModel struct {
	ID                       string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID                 string  `gorm:"not null;type:uuid"`
	SKUID                    string  `gorm:"column:sku_id;not null;type:uuid"`
	SourceType               string  `gorm:"column:source_type;not null"`
	ReferenceType            *string `gorm:"column:reference_type"`
	ReferenceID              *string `gorm:"column:reference_id;type:uuid"`
	QuantityDelta            int64   `gorm:"not null"`
	QuantityAfter            int64   `gorm:"column:quantity_after;not null"`
	CostPriceMinorUnits      *int64
	InventoryValueMinorUnits *int64
	OccurredAt               time.Time `gorm:"not null"`
	CreatedAt                time.Time
}

func (skuCostPriceLedgerModel) TableName() string { return "sku_cost_price_ledger" }

// SKUCostPriceLedgerRepo is the GORM adapter for
// pmsDomain.SKUCostPriceLedgerRepository -- append-only, so it exposes only
// Create/ListBySKU, no Update/Delete.
type SKUCostPriceLedgerRepo struct{ db *gorm.DB }

func NewSKUCostPriceLedgerRepo(db *gorm.DB) *SKUCostPriceLedgerRepo {
	return &SKUCostPriceLedgerRepo{db: db}
}

func (r *SKUCostPriceLedgerRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("pms: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *SKUCostPriceLedgerRepo) Create(ctx context.Context, e *pmsDomain.SKUCostPriceLedgerEntry) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := skuCostPriceLedgerModel{
		TenantID: e.TenantID, SKUID: e.SKUID, SourceType: string(e.SourceType),
		ReferenceType: e.ReferenceType, ReferenceID: e.ReferenceID,
		QuantityDelta: e.QuantityDelta, QuantityAfter: e.QuantityAfter,
		CostPriceMinorUnits: e.CostPriceMinorUnits, InventoryValueMinorUnits: e.InventoryValueMinorUnits,
		OccurredAt: e.OccurredAt, CreatedAt: e.CreatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("skuCostPriceLedgerRepo.Create: %w", err)
	}
	e.ID = m.ID
	return nil
}

func (r *SKUCostPriceLedgerRepo) ListBySKU(ctx context.Context, skuID string, limit int) ([]pmsDomain.SKUCostPriceLedgerEntry, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultCostPriceLedgerLimit
	}
	var rows []skuCostPriceLedgerModel
	if err := tx.Where("sku_id = ?", skuID).Order("occurred_at desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("skuCostPriceLedgerRepo.ListBySKU: %w", err)
	}
	entries := make([]pmsDomain.SKUCostPriceLedgerEntry, len(rows))
	for i, row := range rows {
		entries[i] = pmsDomain.SKUCostPriceLedgerEntry{
			ID: row.ID, TenantID: row.TenantID, SKUID: row.SKUID,
			SourceType:    pmsDomain.SKUCostPriceLedgerSourceType(row.SourceType),
			ReferenceType: row.ReferenceType, ReferenceID: row.ReferenceID,
			QuantityDelta: row.QuantityDelta, QuantityAfter: row.QuantityAfter,
			CostPriceMinorUnits: row.CostPriceMinorUnits, InventoryValueMinorUnits: row.InventoryValueMinorUnits,
			OccurredAt: row.OccurredAt, CreatedAt: row.CreatedAt,
		}
	}
	return entries, nil
}

// ── SKU price history repo ──────────────────────────────────────────────────

const defaultPriceHistoryLimit = 90

type skuPriceHistoryModel struct {
	ID                  string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID            string `gorm:"not null;type:uuid"`
	SKUID               string `gorm:"column:sku_id;not null;type:uuid"`
	PriceMinorUnits     int64  `gorm:"not null"`
	SalePriceMinorUnits *int64
	CurrencyCode        string `gorm:"not null"`
	ValidFrom           time.Time
	ValidUntil          *time.Time
	Reason              string `gorm:"not null"`
	CreatedAt           time.Time
}

func (skuPriceHistoryModel) TableName() string { return "sku_price_history" }

type SKUPriceHistoryRepo struct{ db *gorm.DB }

func NewSKUPriceHistoryRepo(db *gorm.DB) *SKUPriceHistoryRepo {
	return &SKUPriceHistoryRepo{db: db}
}

func (r *SKUPriceHistoryRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("pms: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *SKUPriceHistoryRepo) Create(ctx context.Context, h *pmsDomain.SKUPriceHistory) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := skuPriceHistoryModel{
		TenantID: h.TenantID, SKUID: h.SKUID,
		PriceMinorUnits: h.PriceMinorUnits, SalePriceMinorUnits: h.SalePriceMinorUnits,
		CurrencyCode: h.CurrencyCode, ValidFrom: h.ValidFrom, ValidUntil: h.ValidUntil,
		Reason: string(h.Reason), CreatedAt: h.CreatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("skuPriceHistoryRepo.Create: %w", err)
	}
	h.ID = m.ID
	return nil
}

func (r *SKUPriceHistoryRepo) CloseOpen(ctx context.Context, skuID string, until time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	err = tx.Model(&skuPriceHistoryModel{}).
		Where("sku_id = ? AND valid_until IS NULL", skuID).
		Update("valid_until", until).Error
	if err != nil {
		return fmt.Errorf("skuPriceHistoryRepo.CloseOpen: %w", err)
	}
	return nil
}

func (r *SKUPriceHistoryRepo) ListBySKU(ctx context.Context, skuID string, limit int) ([]pmsDomain.SKUPriceHistory, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultPriceHistoryLimit
	}
	var rows []skuPriceHistoryModel
	if err := tx.Where("sku_id = ?", skuID).Order("valid_from desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("skuPriceHistoryRepo.ListBySKU: %w", err)
	}
	history := make([]pmsDomain.SKUPriceHistory, len(rows))
	for i, row := range rows {
		history[i] = pmsDomain.SKUPriceHistory{
			ID: row.ID, TenantID: row.TenantID, SKUID: row.SKUID,
			PriceMinorUnits: row.PriceMinorUnits, SalePriceMinorUnits: row.SalePriceMinorUnits,
			CurrencyCode: row.CurrencyCode, ValidFrom: row.ValidFrom, ValidUntil: row.ValidUntil,
			Reason: pmsDomain.PriceChangeReason(row.Reason), CreatedAt: row.CreatedAt,
		}
	}
	return history, nil
}
