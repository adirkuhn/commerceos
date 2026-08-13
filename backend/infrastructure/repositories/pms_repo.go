package repositories

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

// ── GORM models ──────────────────────────────────────────────────────────────

// stringSlice is a zero-dependency []string <-> jsonb Scanner/Valuer --
// avoids pulling in a Postgres-array driver type for the one column that
// needs it (products.tags).
type stringSlice []string

func (s stringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal([]string(s))
	return string(b), err
}

func (s *stringSlice) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("stringSlice.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(b, (*[]string)(s))
}

type categoryModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string `gorm:"not null;type:uuid"`
	ParentID  *string
	Name      string `gorm:"not null"`
	Slug      string `gorm:"not null"`
	Position  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (categoryModel) TableName() string { return "categories" }

type productModel struct {
	ID                          string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID                    string `gorm:"not null;type:uuid"`
	CategoryID                  *string
	Name                        string `gorm:"not null"`
	Slug                        string `gorm:"not null"`
	Description                 string
	Status                      string      `gorm:"not null"`
	HasVariants                 bool        `gorm:"not null"`
	Tags                        stringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	SyncVariantsAsWooVariations bool        `gorm:"column:sync_variants_as_woo_variations;not null;default:false"`
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

func (productModel) TableName() string { return "products" }

type variantModel struct {
	ID              string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID        string `gorm:"not null;type:uuid"`
	ProductID       string `gorm:"not null;type:uuid"`
	Name            string `gorm:"not null"`
	Position        int
	FeaturedMediaID *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (variantModel) TableName() string { return "variants" }

type skuModel struct {
	ID                  string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID            string `gorm:"not null;type:uuid"`
	VariantID           string `gorm:"not null;type:uuid"`
	SKUCode             string `gorm:"column:sku_code;not null"`
	Barcode             string
	PriceMinorUnits     int64 `gorm:"not null"`
	SalePriceMinorUnits *int64
	SaleEndDate         *time.Time
	CurrencyCode        string `gorm:"not null"`
	WeightGrams         *int64
	LengthMM            *int64
	WidthMM             *int64
	HeightMM            *int64
	IsVirtual           bool    `gorm:"not null"`
	TracksExpiry        bool    `gorm:"not null"`
	CaseOfSKUID         *string `gorm:"column:case_of_sku_id"`
	UnitsPerCase        *int64
	UnitType            string  `gorm:"column:unit_type;not null;default:unit"`
	PortionedFromSKUID  *string `gorm:"column:portioned_from_sku_id"`
	Status              string `gorm:"not null"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (skuModel) TableName() string { return "skus" }

// ── CategoryRepo ──────────────────────────────────────────────────────────────

type CategoryRepo struct{ db *gorm.DB }

func NewCategoryRepo(db *gorm.DB) *CategoryRepo { return &CategoryRepo{db: db} }

func (r *CategoryRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("pms: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *CategoryRepo) Create(ctx context.Context, c *pmsDomain.Category) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := categoryModel{
		TenantID:  c.TenantID,
		ParentID:  c.ParentID,
		Name:      c.Name,
		Slug:      c.Slug,
		Position:  c.Position,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("categoryRepo.Create: %w", err)
	}
	c.ID = m.ID
	return nil
}

func (r *CategoryRepo) FindByID(ctx context.Context, id string) (*pmsDomain.Category, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m categoryModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("categoryRepo.FindByID: %w", err)
	}
	return toCategoryDomain(m), nil
}

func (r *CategoryRepo) FindBySlug(ctx context.Context, slug string) (*pmsDomain.Category, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m categoryModel
	if err := tx.Where("slug = ?", slug).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("categoryRepo.FindBySlug: %w", err)
	}
	return toCategoryDomain(m), nil
}

func (r *CategoryRepo) List(ctx context.Context) ([]pmsDomain.Category, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []categoryModel
	if err := tx.Order("position asc, name asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("categoryRepo.List: %w", err)
	}
	categories := make([]pmsDomain.Category, len(rows))
	for i, row := range rows {
		categories[i] = *toCategoryDomain(row)
	}
	return categories, nil
}

func (r *CategoryRepo) Update(ctx context.Context, c *pmsDomain.Category) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&categoryModel{}).Where("id = ?", c.ID).
		Updates(map[string]any{
			"parent_id":  c.ParentID,
			"name":       c.Name,
			"updated_at": c.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("categoryRepo.Update: %w", err)
	}
	return nil
}

func (r *CategoryRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&categoryModel{}).Error; err != nil {
		return fmt.Errorf("categoryRepo.Delete: %w", err)
	}
	return nil
}

func toCategoryDomain(m categoryModel) *pmsDomain.Category {
	return &pmsDomain.Category{
		ID:        m.ID,
		TenantID:  m.TenantID,
		ParentID:  m.ParentID,
		Name:      m.Name,
		Slug:      m.Slug,
		Position:  m.Position,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// ── ProductRepo ───────────────────────────────────────────────────────────────

type ProductRepo struct{ db *gorm.DB }

func NewProductRepo(db *gorm.DB) *ProductRepo { return &ProductRepo{db: db} }

func (r *ProductRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("pms: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ProductRepo) Create(ctx context.Context, p *pmsDomain.Product) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := productModel{
		TenantID:                    p.TenantID,
		CategoryID:                  p.CategoryID,
		Name:                        p.Name,
		Slug:                        p.Slug,
		Description:                 p.Description,
		Status:                      string(p.Status),
		HasVariants:                 p.HasVariants,
		Tags:                        stringSlice(p.Tags),
		SyncVariantsAsWooVariations: p.SyncVariantsAsWooVariations,
		CreatedAt:                   p.CreatedAt,
		UpdatedAt:                   p.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("productRepo.Create: %w", err)
	}
	p.ID = m.ID
	return nil
}

func (r *ProductRepo) FindByID(ctx context.Context, id string) (*pmsDomain.Product, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m productModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrProductNotFound
		}
		return nil, fmt.Errorf("productRepo.FindByID: %w", err)
	}
	return toProductDomain(m), nil
}

func (r *ProductRepo) FindBySlug(ctx context.Context, slug string) (*pmsDomain.Product, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m productModel
	if err := tx.Where("slug = ?", slug).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrProductNotFound
		}
		return nil, fmt.Errorf("productRepo.FindBySlug: %w", err)
	}
	return toProductDomain(m), nil
}

func (r *ProductRepo) List(ctx context.Context, filter pmsDomain.ProductFilter) ([]pmsDomain.Product, int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, 0, err
	}
	q := tx.Model(&productModel{})
	if filter.CategoryID != nil {
		q = q.Where("category_id = ?", *filter.CategoryID)
	}
	if filter.Status != nil {
		q = q.Where("status = ?", string(*filter.Status))
	}
	if filter.Search != nil && *filter.Search != "" {
		// unaccent() so "acai" matches "Açaí", "feijao" matches "Feijão",
		// etc. — plain ILIKE is case-insensitive but not accent-insensitive.
		// sku_code/barcode are plain ASCII codes, not natural-language names,
		// so no unaccent() needed there. EXISTS rather than a JOIN so a
		// product with several matching SKUs still comes back exactly once.
		pattern := "%" + *filter.Search + "%"
		q = q.Where(
			"unaccent(name) ILIKE unaccent(?) OR unaccent(slug) ILIKE unaccent(?) OR EXISTS ("+
				"SELECT 1 FROM skus s JOIN variants v ON v.id = s.variant_id "+
				"WHERE v.product_id = products.id AND (s.sku_code ILIKE ? OR s.barcode ILIKE ?)"+
				")",
			pattern, pattern, pattern, pattern,
		)
	}
	if filter.UpdatedAfter != nil {
		q = q.Where("updated_at > ?", *filter.UpdatedAfter)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("productRepo.List: count: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 200
	}
	var rows []productModel
	if err := q.Order("name asc").Limit(limit).Offset(filter.Offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("productRepo.List: %w", err)
	}
	products := make([]pmsDomain.Product, len(rows))
	for i, row := range rows {
		products[i] = *toProductDomain(row)
	}
	return products, total, nil
}

func (r *ProductRepo) SearchByAnyNameToken(ctx context.Context, tokens []string, limit int) ([]pmsDomain.Product, error) {
	if len(tokens) == 0 {
		return nil, nil
	}
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 200
	}

	conds := make([]string, len(tokens))
	args := make([]any, len(tokens))
	for i, t := range tokens {
		conds[i] = "unaccent(name) ILIKE unaccent(?)"
		args[i] = "%" + t + "%"
	}
	var rows []productModel
	if err := tx.Model(&productModel{}).Where(strings.Join(conds, " OR "), args...).
		Order("name asc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("productRepo.SearchByAnyNameToken: %w", err)
	}
	products := make([]pmsDomain.Product, len(rows))
	for i, row := range rows {
		products[i] = *toProductDomain(row)
	}
	return products, nil
}

func (r *ProductRepo) Update(ctx context.Context, p *pmsDomain.Product) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	tagsValue, err := stringSlice(p.Tags).Value()
	if err != nil {
		return fmt.Errorf("productRepo.Update: %w", err)
	}
	if err := tx.Model(&productModel{}).Where("id = ?", p.ID).
		Updates(map[string]any{
			"category_id":                     p.CategoryID,
			"name":                            p.Name,
			"description":                     p.Description,
			"status":                          string(p.Status),
			"has_variants":                    p.HasVariants,
			"tags":                            tagsValue,
			"sync_variants_as_woo_variations": p.SyncVariantsAsWooVariations,
			"updated_at":                      p.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("productRepo.Update: %w", err)
	}
	return nil
}

func (r *ProductRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&productModel{}).Error; err != nil {
		return fmt.Errorf("productRepo.Delete: %w", err)
	}
	return nil
}

func toProductDomain(m productModel) *pmsDomain.Product {
	return &pmsDomain.Product{
		ID:                          m.ID,
		TenantID:                    m.TenantID,
		CategoryID:                  m.CategoryID,
		Name:                        m.Name,
		Slug:                        m.Slug,
		Description:                 m.Description,
		Status:                      pmsDomain.ProductStatus(m.Status),
		HasVariants:                 m.HasVariants,
		Tags:                        []string(m.Tags),
		SyncVariantsAsWooVariations: m.SyncVariantsAsWooVariations,
		CreatedAt:                   m.CreatedAt,
		UpdatedAt:                   m.UpdatedAt,
	}
}

// ── VariantRepo ───────────────────────────────────────────────────────────────

type VariantRepo struct{ db *gorm.DB }

func NewVariantRepo(db *gorm.DB) *VariantRepo { return &VariantRepo{db: db} }

func (r *VariantRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("pms: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *VariantRepo) Create(ctx context.Context, v *pmsDomain.Variant) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := variantModel{
		TenantID:        v.TenantID,
		ProductID:       v.ProductID,
		Name:            v.Name,
		Position:        v.Position,
		FeaturedMediaID: v.FeaturedMediaID,
		CreatedAt:       v.CreatedAt,
		UpdatedAt:       v.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("variantRepo.Create: %w", err)
	}
	v.ID = m.ID
	return nil
}

func (r *VariantRepo) FindByID(ctx context.Context, id string) (*pmsDomain.Variant, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m variantModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrVariantNotFound
		}
		return nil, fmt.Errorf("variantRepo.FindByID: %w", err)
	}
	return toVariantDomain(m), nil
}

func (r *VariantRepo) ListByProduct(ctx context.Context, productID string) ([]pmsDomain.Variant, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []variantModel
	if err := tx.Where("product_id = ?", productID).Order("position asc, name asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("variantRepo.ListByProduct: %w", err)
	}
	variants := make([]pmsDomain.Variant, len(rows))
	for i, row := range rows {
		variants[i] = *toVariantDomain(row)
	}
	return variants, nil
}

func (r *VariantRepo) Update(ctx context.Context, v *pmsDomain.Variant) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&variantModel{}).Where("id = ?", v.ID).
		Updates(map[string]any{
			"name":              v.Name,
			"position":          v.Position,
			"featured_media_id": v.FeaturedMediaID,
			"updated_at":        v.UpdatedAt,
		}).Error; err != nil {
		return fmt.Errorf("variantRepo.Update: %w", err)
	}
	return nil
}

func (r *VariantRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&variantModel{}).Error; err != nil {
		return fmt.Errorf("variantRepo.Delete: %w", err)
	}
	return nil
}

func toVariantDomain(m variantModel) *pmsDomain.Variant {
	return &pmsDomain.Variant{
		ID:              m.ID,
		TenantID:        m.TenantID,
		ProductID:       m.ProductID,
		Name:            m.Name,
		Position:        m.Position,
		FeaturedMediaID: m.FeaturedMediaID,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

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
		TenantID:            s.TenantID,
		VariantID:           s.VariantID,
		SKUCode:             s.SKUCode,
		Barcode:             s.Barcode,
		PriceMinorUnits:     s.PriceMinorUnits,
		SalePriceMinorUnits: s.SalePriceMinorUnits,
		SaleEndDate:         s.SaleEndDate,
		CurrencyCode:        s.CurrencyCode,
		WeightGrams:         s.WeightGrams,
		LengthMM:            s.LengthMM,
		WidthMM:             s.WidthMM,
		HeightMM:            s.HeightMM,
		IsVirtual:           s.IsVirtual,
		TracksExpiry:        s.TracksExpiry,
		CaseOfSKUID:         s.CaseOfSKUID,
		UnitsPerCase:        s.UnitsPerCase,
		UnitType:            string(s.UnitType),
		PortionedFromSKUID:  s.PortionedFromSKUID,
		Status:              string(s.Status),
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
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

func (r *SKURepo) Update(ctx context.Context, s *pmsDomain.SKU) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&skuModel{}).Where("id = ?", s.ID).
		Updates(map[string]any{
			"barcode":                s.Barcode,
			"price_minor_units":      s.PriceMinorUnits,
			"sale_price_minor_units": s.SalePriceMinorUnits,
			"sale_end_date":          s.SaleEndDate,
			"currency_code":          s.CurrencyCode,
			"weight_grams":           s.WeightGrams,
			"length_mm":              s.LengthMM,
			"width_mm":               s.WidthMM,
			"height_mm":              s.HeightMM,
			"is_virtual":             s.IsVirtual,
			"tracks_expiry":          s.TracksExpiry,
			"case_of_sku_id":         s.CaseOfSKUID,
			"units_per_case":         s.UnitsPerCase,
			"unit_type":              string(s.UnitType),
			"portioned_from_sku_id":  s.PortionedFromSKUID,
			"status":                 string(s.Status),
			"updated_at":             s.UpdatedAt,
		}).Error; err != nil {
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

func toSKUDomain(m skuModel) *pmsDomain.SKU {
	return &pmsDomain.SKU{
		ID:                  m.ID,
		TenantID:            m.TenantID,
		VariantID:           m.VariantID,
		SKUCode:             m.SKUCode,
		Barcode:             m.Barcode,
		PriceMinorUnits:     m.PriceMinorUnits,
		SalePriceMinorUnits: m.SalePriceMinorUnits,
		SaleEndDate:         m.SaleEndDate,
		CurrencyCode:        m.CurrencyCode,
		WeightGrams:         m.WeightGrams,
		LengthMM:            m.LengthMM,
		WidthMM:             m.WidthMM,
		HeightMM:            m.HeightMM,
		IsVirtual:           m.IsVirtual,
		TracksExpiry:        m.TracksExpiry,
		CaseOfSKUID:         m.CaseOfSKUID,
		UnitsPerCase:        m.UnitsPerCase,
		UnitType:            pmsDomain.SKUUnitType(m.UnitType),
		PortionedFromSKUID:  m.PortionedFromSKUID,
		Status:              pmsDomain.SKUStatus(m.Status),
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
}

// ── Media repo ───────────────────────────────────────────────────────────────

type mediaModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string `gorm:"not null;type:uuid"`
	ProductID *string
	VariantID *string
	URL       string `gorm:"not null"`
	Type      string `gorm:"not null"`
	Position  int    `gorm:"not null"`
	AltText   string
	CreatedAt time.Time
}

func (mediaModel) TableName() string { return "media" }

type MediaRepo struct{ db *gorm.DB }

func NewMediaRepo(db *gorm.DB) *MediaRepo { return &MediaRepo{db: db} }

func (r *MediaRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("pms: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *MediaRepo) Attach(ctx context.Context, m *pmsDomain.Media) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	row := mediaModel{
		TenantID:  m.TenantID,
		ProductID: m.ProductID,
		VariantID: m.VariantID,
		URL:       m.URL,
		Type:      string(m.Type),
		Position:  m.Position,
		AltText:   m.AltText,
		CreatedAt: m.CreatedAt,
	}
	if err := tx.Create(&row).Error; err != nil {
		return fmt.Errorf("mediaRepo.Attach: %w", err)
	}
	m.ID = row.ID
	return nil
}

func (r *MediaRepo) FindByID(ctx context.Context, id string) (*pmsDomain.Media, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var row mediaModel
	if err := tx.Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pmsDomain.ErrMediaNotFound
		}
		return nil, fmt.Errorf("mediaRepo.FindByID: %w", err)
	}
	return toMediaDomain(row), nil
}

func (r *MediaRepo) ListByProduct(ctx context.Context, productID string) ([]pmsDomain.Media, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []mediaModel
	if err := tx.Where("product_id = ?", productID).Order("position asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("mediaRepo.ListByProduct: %w", err)
	}
	out := make([]pmsDomain.Media, len(rows))
	for i, row := range rows {
		out[i] = *toMediaDomain(row)
	}
	return out, nil
}

func (r *MediaRepo) Update(ctx context.Context, m *pmsDomain.Media) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Model(&mediaModel{}).Where("id = ?", m.ID).
		Update("alt_text", m.AltText).Error; err != nil {
		return fmt.Errorf("mediaRepo.Update: %w", err)
	}
	return nil
}

func (r *MediaRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&mediaModel{}).Error; err != nil {
		return fmt.Errorf("mediaRepo.Delete: %w", err)
	}
	return nil
}

func toMediaDomain(m mediaModel) *pmsDomain.Media {
	return &pmsDomain.Media{
		ID:        m.ID,
		TenantID:  m.TenantID,
		ProductID: m.ProductID,
		VariantID: m.VariantID,
		URL:       m.URL,
		Type:      pmsDomain.MediaType(m.Type),
		Position:  m.Position,
		AltText:   m.AltText,
		CreatedAt: m.CreatedAt,
	}
}
