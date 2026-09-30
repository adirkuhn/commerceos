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
	"gorm.io/gorm/clause"

	"github.com/adirkuhn/commerceos/domain/locale"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

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
	// Translations is written only by SetTranslations (Update leaves it).
	Translations productTranslations `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (productModel) TableName() string { return "products" }

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

// applyProductFilter applies every ProductFilter constraint to q except
// AndCategoryID/AndIncludeIDs when includeAndCategory is false -- shared by
// List (which always applies them) and CountByCategory (which deliberately
// doesn't: a category facet's count must reflect "how many if this
// category were picked instead", not "how many within the already-picked
// category/manual list").
// Returns the search pattern too, since List's result ordering needs it.
func applyProductFilter(q *gorm.DB, filter pmsDomain.ProductFilter, includeAndCategory bool) (*gorm.DB, string, error) {
	if filter.CategoryID != nil || filter.Tag != nil || filter.IncludeIDs != nil {
		var ors []string
		var args []any
		if filter.CategoryID != nil {
			ors = append(ors, "category_id = ?")
			args = append(args, *filter.CategoryID)
		}
		if filter.Tag != nil {
			tagJSON, err := json.Marshal([]string{*filter.Tag})
			if err != nil {
				return nil, "", fmt.Errorf("marshal tag: %w", err)
			}
			ors = append(ors, "tags @> ?::jsonb")
			args = append(args, string(tagJSON))
		}
		if filter.IncludeIDs != nil && len(*filter.IncludeIDs) > 0 {
			ors = append(ors, "id IN ?")
			args = append(args, *filter.IncludeIDs)
		}
		if len(ors) == 0 {
			// CategoryID/Tag both nil and IncludeIDs a non-nil empty slice
			// -- see ProductFilter.IncludeIDs' doc comment: this must match
			// nothing, not fall through to "no constraint at all".
			q = q.Where("1 = 0")
		} else {
			q = q.Where(strings.Join(ors, " OR "), args...)
		}
	}
	if len(filter.ExcludeIDs) > 0 {
		q = q.Where("id NOT IN ?", filter.ExcludeIDs)
	}
	if includeAndCategory && filter.AndCategoryID != nil {
		q = q.Where("category_id = ?", *filter.AndCategoryID)
	}
	if includeAndCategory && filter.AndIncludeIDs != nil {
		if len(*filter.AndIncludeIDs) > 0 {
			q = q.Where("id IN ?", *filter.AndIncludeIDs)
		} else {
			q = q.Where("1 = 0")
		}
	}
	if filter.Status != nil {
		q = q.Where("status = ?", string(*filter.Status))
	}
	var pattern string
	if filter.Search != nil && *filter.Search != "" {
		// unaccent() so "acai" matches "Açaí", "feijao" matches "Feijão",
		// etc. — plain ILIKE is case-insensitive but not accent-insensitive.
		// similarity() (pg_trgm, 077_pg_trgm_extension.sql) is the typo
		// -tolerant fallback on top of that -- "Wigdet"/"Widgt" still finds
		// "Widget" even though neither is a substring of it. sku_code/
		// barcode/scale_code (plus their alias tables -- a SKU can have
		// several of each, see barcode_aliases/scale_code_aliases) are
		// plain ASCII codes, not natural-language names, so neither
		// unaccent() nor fuzzy matching applies there -- exact ILIKE only.
		// EXISTS rather than a JOIN so a product with several matching
		// SKUs/aliases still comes back exactly once.
		pattern = "%" + *filter.Search + "%"
		q = q.Where(
			"unaccent(name) ILIKE unaccent(?) OR unaccent(slug) ILIKE unaccent(?) OR "+
				"EXISTS (SELECT 1 FROM jsonb_each(products.translations) tr WHERE unaccent(tr.value->>'name') ILIKE unaccent(?)) OR "+
				"similarity(immutable_unaccent(name), immutable_unaccent(?)) > ? OR EXISTS ("+
				"SELECT 1 FROM skus s JOIN variants v ON v.id = s.variant_id "+
				"WHERE v.product_id = products.id AND ("+
				"s.sku_code ILIKE ? OR s.barcode ILIKE ? OR s.scale_code ILIKE ? OR "+
				"EXISTS (SELECT 1 FROM barcode_aliases ba WHERE ba.sku_id = s.id AND ba.code ILIKE ?) OR "+
				"EXISTS (SELECT 1 FROM scale_code_aliases sca WHERE sca.sku_id = s.id AND sca.code ILIKE ?)"+
				")"+
				")",
			pattern, pattern, pattern, *filter.Search, fuzzySearchSimilarityThreshold, pattern, pattern, pattern, pattern, pattern,
		)
	}
	if filter.UpdatedAfter != nil {
		q = q.Where("updated_at > ?", *filter.UpdatedAfter)
	}
	return q, pattern, nil
}

func (r *ProductRepo) List(ctx context.Context, filter pmsDomain.ProductFilter) ([]pmsDomain.Product, int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, 0, err
	}
	q, pattern, err := applyProductFilter(tx.Model(&productModel{}), filter, true)
	if err != nil {
		return nil, 0, fmt.Errorf("productRepo.List: %w", err)
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
	// No search term: most-recently-created first, so a newly created or
	// duplicated product is immediately visible at the top of the list
	// instead of buried alphabetically. A search still ranks by relevance
	// (below) -- recency isn't a useful signal once the user has typed
	// something specific to find.
	orderedQuery := q.Order("created_at desc")
	if len(filter.OrderByIDs) > 0 {
		orderedQuery = q.Clauses(clause.OrderBy{Expression: clause.Expr{
			SQL: "array_position(?::uuid[], products.id), products.id", Vars: []any{"{" + strings.Join(filter.OrderByIDs, ",") + "}"}, WithoutParentheses: true,
		}})
	}
	if filter.Search != nil && *filter.Search != "" {
		// Barcode is the primary key a scan or a typed code is ever really
		// about -- a product whose SKU code/barcode/scale code actually
		// matches must outrank one that only fuzzy-matched by name, no
		// matter how high that name similarity scores (a name containing
		// the search term verbatim scores very high despite being
		// functionally unrelated to what was scanned). Tier 0 (code hit,
		// including a barcode/scale-code alias) always sorts before tier 1
		// (name-only hit); similarity is only the tiebreak within a tier.
		orderedQuery = q.Clauses(clause.OrderBy{
			Expression: clause.Expr{
				SQL: "CASE WHEN EXISTS (" +
					"SELECT 1 FROM skus s JOIN variants v ON v.id = s.variant_id " +
					"WHERE v.product_id = products.id AND (" +
					"s.sku_code ILIKE ? OR s.barcode ILIKE ? OR s.scale_code ILIKE ? OR " +
					"EXISTS (SELECT 1 FROM barcode_aliases ba WHERE ba.sku_id = s.id AND ba.code ILIKE ?) OR " +
					"EXISTS (SELECT 1 FROM scale_code_aliases sca WHERE sca.sku_id = s.id AND sca.code ILIKE ?)" +
					")" +
					") THEN 0 ELSE 1 END, " +
					"similarity(immutable_unaccent(name), immutable_unaccent(?)) DESC, name asc",
				Vars: []interface{}{pattern, pattern, pattern, pattern, pattern, *filter.Search},
			},
		})
	}
	if err := orderedQuery.Limit(limit).Offset(filter.Offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("productRepo.List: %w", err)
	}
	products := make([]pmsDomain.Product, len(rows))
	for i, row := range rows {
		products[i] = *toProductDomain(row)
	}
	return products, total, nil
}

type productCategoryCountRow struct {
	CategoryID *string
	Count      int64
}

// CountByCategory groups filter's matching products by category_id --
// AndCategoryID is deliberately ignored (see applyProductFilter's doc
// comment) so the counts reflect every category a shopper could switch to,
// not just the one currently selected. A nil CategoryID (uncategorized)
// comes back under the empty-string key.
// ListLinks: see pmsDomain.ProductLinkLister.
func (r *ProductRepo) ListLinks(ctx context.Context, filter pmsDomain.ProductFilter) ([]pmsDomain.ProductLink, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	q, _, err := applyProductFilter(tx.Model(&productModel{}), filter, true)
	if err != nil {
		return nil, fmt.Errorf("productRepo.ListLinks: %w", err)
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 200
	}
	var rows []struct {
		Slug      string
		UpdatedAt time.Time
	}
	if err := q.Select("slug", "updated_at").Order("updated_at desc, id").Limit(limit).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("productRepo.ListLinks: %w", err)
	}
	links := make([]pmsDomain.ProductLink, len(rows))
	for i, row := range rows {
		links[i] = pmsDomain.ProductLink{Slug: row.Slug, UpdatedAt: row.UpdatedAt}
	}
	return links, nil
}

func (r *ProductRepo) CountByCategory(ctx context.Context, filter pmsDomain.ProductFilter) (map[string]int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	q, _, err := applyProductFilter(tx.Model(&productModel{}), filter, false)
	if err != nil {
		return nil, fmt.Errorf("productRepo.CountByCategory: %w", err)
	}
	var rows []productCategoryCountRow
	if err := q.Select("category_id, COUNT(*) as count").Group("category_id").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("productRepo.CountByCategory: %w", err)
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		if row.CategoryID != nil {
			out[*row.CategoryID] = row.Count
		} else {
			out[""] = row.Count
		}
	}
	return out, nil
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
			"slug":                            p.Slug,
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

// SetTranslations replaces a product's translations, touching nothing else.
func (r *ProductRepo) SetTranslations(ctx context.Context, productID string, tr map[locale.Locale]pmsDomain.ProductTexts) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Model(&productModel{}).Where("id = ?", productID).Update("translations", toProductTranslations(tr))
	if res.Error != nil {
		return fmt.Errorf("productRepo.SetTranslations: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return pmsDomain.ErrProductNotFound
	}
	return nil
}

func (r *ProductRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&productModel{}).Error; err != nil {
		if historyErr := skuHistoryViolation(err); historyErr != nil {
			return historyErr
		}
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
		Translations:                m.Translations.toDomain(),
		CreatedAt:                   m.CreatedAt,
		UpdatedAt:                   m.UpdatedAt,
	}
}

type productTextsJSON struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// productTranslations is the JSONB translations column.
type productTranslations map[locale.Locale]productTextsJSON

func toProductTranslations(tr map[locale.Locale]pmsDomain.ProductTexts) productTranslations {
	out := productTranslations{}
	for l, t := range tr {
		out[l] = productTextsJSON{Name: t.Name, Description: t.Description}
	}
	return out
}

func (p productTranslations) toDomain() map[locale.Locale]pmsDomain.ProductTexts {
	if len(p) == 0 {
		return nil
	}
	out := make(map[locale.Locale]pmsDomain.ProductTexts, len(p))
	for l, t := range p {
		out[l] = pmsDomain.ProductTexts{Name: t.Name, Description: t.Description}
	}
	return out
}

func (p productTranslations) Value() (driver.Value, error) {
	if len(p) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(map[locale.Locale]productTextsJSON(p))
	return string(b), err
}

func (p *productTranslations) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*p = nil
		return nil
	case []byte:
		return json.Unmarshal(v, (*map[locale.Locale]productTextsJSON)(p))
	case string:
		return json.Unmarshal([]byte(v), (*map[locale.Locale]productTextsJSON)(p))
	default:
		return fmt.Errorf("productTranslations.Scan: unsupported type %T", value)
	}
}
