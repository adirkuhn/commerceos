package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type skuModel struct {
	ID                      string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID                string `gorm:"not null;type:uuid"`
	VariantID               string `gorm:"not null;type:uuid"`
	SKUCode                 string `gorm:"column:sku_code;not null"`
	Barcode                 string
	PriceMinorUnits         int64 `gorm:"not null"`
	SalePriceMinorUnits     *int64
	SaleEndDate             *time.Time
	CurrencyCode            string `gorm:"not null"`
	WeightGrams             *int64
	LengthMM                *int64
	WidthMM                 *int64
	HeightMM                *int64
	IsVirtual               bool    `gorm:"not null"`
	TracksExpiry            bool    `gorm:"not null"`
	CaseOfSKUID             *string `gorm:"column:case_of_sku_id"`
	UnitsPerCase            *int64
	UnitType                string  `gorm:"column:unit_type;not null;default:unit"`
	PortionedFromSKUID      *string `gorm:"column:portioned_from_sku_id"`
	ScaleCode               *string `gorm:"column:scale_code"`
	PricingMode             string  `gorm:"column:pricing_mode;not null;default:fixed"`
	PricePerKiloMinorUnits  *int64  `gorm:"column:price_per_kilo_minor_units"`
	TaxClassID              *string `gorm:"column:tax_class_id"`
	WooTaxClassSlug         *string `gorm:"column:woo_tax_class_slug"`
	WooTaxStatus            *string `gorm:"column:woo_tax_status"`
	Status                  string  `gorm:"not null"`
	CostPriceMinorUnits     *int64  `gorm:"column:cost_price_minor_units"`
	CostPriceSource         string  `gorm:"column:cost_price_source;not null;default:computed"`
	InventoryCostMinorUnits *int64  `gorm:"column:inventory_cost_minor_units"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func (skuModel) TableName() string { return "skus" }

// ── SKURepo ───────────────────────────────────────────────────────────────────

type SKURepo struct{ db *gorm.DB }

func NewSKURepo(db *gorm.DB) *SKURepo { return &SKURepo{db: db} }

func (r *SKURepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("pms: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *SKURepo) Create(ctx context.Context, s *pmsDomain.SKU) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := skuModel{
		TenantID:               s.TenantID,
		VariantID:              s.VariantID,
		SKUCode:                s.SKUCode,
		Barcode:                s.Barcode,
		PriceMinorUnits:        s.PriceMinorUnits,
		SalePriceMinorUnits:    s.SalePriceMinorUnits,
		SaleEndDate:            s.SaleEndDate,
		CurrencyCode:           s.CurrencyCode,
		WeightGrams:            s.WeightGrams,
		LengthMM:               s.LengthMM,
		WidthMM:                s.WidthMM,
		HeightMM:               s.HeightMM,
		IsVirtual:              s.IsVirtual,
		TracksExpiry:           s.TracksExpiry,
		CaseOfSKUID:            s.CaseOfSKUID,
		UnitsPerCase:           s.UnitsPerCase,
		UnitType:               string(s.UnitType),
		PortionedFromSKUID:     s.PortionedFromSKUID,
		ScaleCode:              s.ScaleCode,
		PricingMode:            string(s.PricingMode),
		PricePerKiloMinorUnits: s.PricePerKiloMinorUnits,
		TaxClassID:             s.TaxClassID,
		WooTaxClassSlug:        s.WooTaxClassSlug,
		WooTaxStatus:           s.WooTaxStatus,
		Status:                 string(s.Status),
		CreatedAt:              s.CreatedAt,
		UpdatedAt:              s.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.ConstraintName {
			case "skus_tenant_barcode_idx":
				return pmsDomain.ErrDuplicateBarcode
			case "skus_tenant_scale_code_idx":
				return pmsDomain.ErrDuplicateScaleCode
			}
		}
		return fmt.Errorf("skuRepo.Create: %w", err)
	}
	s.ID = m.ID
	return nil
}

func (r *SKURepo) FindByID(ctx context.Context, id string) (*pmsDomain.SKU, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m skuModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrSKUNotFound
		}
		return nil, fmt.Errorf("skuRepo.FindByID: %w", err)
	}
	return toSKUDomain(m), nil
}

func (r *SKURepo) FindByCode(ctx context.Context, code string) (*pmsDomain.SKU, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m skuModel
	if err := tx.Where("sku_code = ?", code).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrSKUNotFound
		}
		return nil, fmt.Errorf("skuRepo.FindByCode: %w", err)
	}
	return toSKUDomain(m), nil
}

func (r *SKURepo) FindByBarcode(ctx context.Context, barcode string) (*pmsDomain.SKU, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m skuModel
	if err := tx.Where("barcode = ?", barcode).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrSKUNotFound
		}
		return nil, fmt.Errorf("skuRepo.FindByBarcode: %w", err)
	}
	return toSKUDomain(m), nil
}

func (r *SKURepo) FindByScaleCode(ctx context.Context, scaleCode string) (*pmsDomain.SKU, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m skuModel
	if err := tx.Where("scale_code = ?", scaleCode).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrSKUNotFound
		}
		return nil, fmt.Errorf("skuRepo.FindByScaleCode: %w", err)
	}
	return toSKUDomain(m), nil
}

func (r *SKURepo) FindByExactProductName(ctx context.Context, name string) (*pmsDomain.SKU, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	where := fmt.Sprintf(normalizedNameSQL, "products.name") + " = " + fmt.Sprintf(normalizedNameSQL, "?")
	var models []skuModel
	err = tx.Joins("JOIN variants ON variants.id = skus.variant_id").
		Joins("JOIN products ON products.id = variants.product_id").
		Where(where, name).
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("skuRepo.FindByExactProductName: %w", err)
	}
	if len(models) != 1 {
		return nil, pmsDomain.ErrSKUNotFound
	}
	return toSKUDomain(models[0]), nil
}

func (r *SKURepo) ListByVariant(ctx context.Context, variantID string) ([]pmsDomain.SKU, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []skuModel
	if err := tx.Where("variant_id = ?", variantID).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("skuRepo.ListByVariant: %w", err)
	}
	skus := make([]pmsDomain.SKU, len(rows))
	for i, row := range rows {
		skus[i] = *toSKUDomain(row)
	}
	return skus, nil
}

// ListByVariantIDs bulk-loads SKUs for multiple variants in one query --
// same N+1-avoidance rationale as VariantRepo.ListByProductIDs.
func (r *SKURepo) ListByVariantIDs(ctx context.Context, variantIDs []string) ([]pmsDomain.SKU, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if len(variantIDs) == 0 {
		return nil, nil
	}
	var rows []skuModel
	if err := tx.Where("variant_id IN ?", variantIDs).Order("created_at asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("skuRepo.ListByVariantIDs: %w", err)
	}
	skus := make([]pmsDomain.SKU, len(rows))
	for i, row := range rows {
		skus[i] = *toSKUDomain(row)
	}
	return skus, nil
}

func (r *SKURepo) Update(ctx context.Context, s *pmsDomain.SKU) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&skuModel{}).Where("id = ?", s.ID).
		Updates(map[string]any{
			"sku_code":                   s.SKUCode,
			"barcode":                    s.Barcode,
			"price_minor_units":          s.PriceMinorUnits,
			"sale_price_minor_units":     s.SalePriceMinorUnits,
			"sale_end_date":              s.SaleEndDate,
			"currency_code":              s.CurrencyCode,
			"weight_grams":               s.WeightGrams,
			"length_mm":                  s.LengthMM,
			"width_mm":                   s.WidthMM,
			"height_mm":                  s.HeightMM,
			"is_virtual":                 s.IsVirtual,
			"tracks_expiry":              s.TracksExpiry,
			"case_of_sku_id":             s.CaseOfSKUID,
			"units_per_case":             s.UnitsPerCase,
			"unit_type":                  string(s.UnitType),
			"portioned_from_sku_id":      s.PortionedFromSKUID,
			"scale_code":                 s.ScaleCode,
			"pricing_mode":               string(s.PricingMode),
			"price_per_kilo_minor_units": s.PricePerKiloMinorUnits,
			"tax_class_id":               s.TaxClassID,
			"woo_tax_class_slug":         s.WooTaxClassSlug,
			"woo_tax_status":             s.WooTaxStatus,
			"status":                     string(s.Status),
			"updated_at":                 s.UpdatedAt,
		}).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.ConstraintName {
			case "skus_tenant_barcode_idx":
				return pmsDomain.ErrDuplicateBarcode
			case "skus_tenant_id_sku_code_key":
				return pmsDomain.ErrDuplicateSKUCode
			case "skus_tenant_scale_code_idx":
				return pmsDomain.ErrDuplicateScaleCode
			}
		}
		return fmt.Errorf("skuRepo.Update: %w", err)
	}
	return nil
}

func (r *SKURepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&skuModel{}).Error; err != nil {
		if historyErr := skuHistoryViolation(err); historyErr != nil {
			return historyErr
		}
		return fmt.Errorf("skuRepo.Delete: %w", err)
	}
	return nil
}

func (r *SKURepo) ListWithExpiredSale(ctx context.Context) ([]pmsDomain.SKU, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []skuModel
	if err := tx.Where("sale_price_minor_units IS NOT NULL AND sale_end_date IS NOT NULL AND sale_end_date < CURRENT_DATE").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("skuRepo.ListWithExpiredSale: %w", err)
	}
	skus := make([]pmsDomain.SKU, len(rows))
	for i, row := range rows {
		skus[i] = *toSKUDomain(row)
	}
	return skus, nil
}

func (r *SKURepo) ListActive(ctx context.Context) ([]pmsDomain.SKU, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []skuModel
	if err := tx.Where("status = ?", string(pmsDomain.SKUStatusActive)).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("skuRepo.ListActive: %w", err)
	}
	skus := make([]pmsDomain.SKU, len(rows))
	for i, row := range rows {
		skus[i] = *toSKUDomain(row)
	}
	return skus, nil
}

func (r *SKURepo) ExistsWithTaxClass(ctx context.Context, taxClassID string) (bool, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return false, err
	}
	var count int64
	if err := tx.Model(&skuModel{}).Where("tax_class_id = ?", taxClassID).Limit(1).Count(&count).Error; err != nil {
		return false, fmt.Errorf("skuRepo.ExistsWithTaxClass: %w", err)
	}
	return count > 0, nil
}

func toSKUDomain(m skuModel) *pmsDomain.SKU {
	return &pmsDomain.SKU{
		ID:                      m.ID,
		TenantID:                m.TenantID,
		VariantID:               m.VariantID,
		SKUCode:                 m.SKUCode,
		Barcode:                 m.Barcode,
		PriceMinorUnits:         m.PriceMinorUnits,
		SalePriceMinorUnits:     m.SalePriceMinorUnits,
		SaleEndDate:             m.SaleEndDate,
		CurrencyCode:            m.CurrencyCode,
		WeightGrams:             m.WeightGrams,
		LengthMM:                m.LengthMM,
		WidthMM:                 m.WidthMM,
		HeightMM:                m.HeightMM,
		IsVirtual:               m.IsVirtual,
		TracksExpiry:            m.TracksExpiry,
		CaseOfSKUID:             m.CaseOfSKUID,
		UnitsPerCase:            m.UnitsPerCase,
		UnitType:                pmsDomain.SKUUnitType(m.UnitType),
		PortionedFromSKUID:      m.PortionedFromSKUID,
		ScaleCode:               m.ScaleCode,
		PricingMode:             pmsDomain.SKUPricingMode(m.PricingMode),
		PricePerKiloMinorUnits:  m.PricePerKiloMinorUnits,
		TaxClassID:              m.TaxClassID,
		WooTaxClassSlug:         m.WooTaxClassSlug,
		WooTaxStatus:            m.WooTaxStatus,
		Status:                  pmsDomain.SKUStatus(m.Status),
		CostPriceMinorUnits:     m.CostPriceMinorUnits,
		CostPriceSource:         pmsDomain.SKUCostPriceSource(m.CostPriceSource),
		InventoryCostMinorUnits: m.InventoryCostMinorUnits,
		CreatedAt:               m.CreatedAt,
		UpdatedAt:               m.UpdatedAt,
	}
}
