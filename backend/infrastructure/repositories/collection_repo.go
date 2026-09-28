package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/collections"
)

type collectionRuleJSON struct {
	Field                string `json:"field"`
	Value                string `json:"value,omitempty"`
	IncludeSubcategories bool   `json:"include_subcategories,omitempty"`
	MinMinorUnits        *int64 `json:"min_minor_units,omitempty"`
	MaxMinorUnits        *int64 `json:"max_minor_units,omitempty"`
	Days                 int    `json:"days,omitempty"`
}

type collectionModel struct {
	ID             string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID       string `gorm:"type:uuid"`
	Name           string
	Slug           string
	Description    string
	ImageAssetID   *string `gorm:"type:uuid"`
	SEOTitle       string  `gorm:"column:seo_title"`
	SEODescription string  `gorm:"column:seo_description"`
	Type           string
	Match          string
	Rules          jsonb[[]collectionRuleJSON] `gorm:"type:jsonb"`
	Sort           string
	Published      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (collectionModel) TableName() string { return "collections" }

func toCollectionModel(c collections.Collection) collectionModel {
	rules := make([]collectionRuleJSON, len(c.Rules))
	for i, r := range c.Rules {
		rules[i] = collectionRuleJSON{Field: string(r.Field), Value: r.Value, IncludeSubcategories: r.IncludeSubcategories,
			MinMinorUnits: r.MinMinorUnits, MaxMinorUnits: r.MaxMinorUnits, Days: r.Days}
	}
	match := string(c.Match)
	if match == "" {
		match = string(collections.MatchAll)
	}
	return collectionModel{
		ID: c.ID, TenantID: c.TenantID, Name: c.Name, Slug: c.Slug, Description: c.Description, ImageAssetID: c.ImageAssetID,
		SEOTitle: c.SEOTitle, SEODescription: c.SEODescription, Type: string(c.Type), Match: match,
		Rules: jsonb[[]collectionRuleJSON]{V: rules}, Sort: string(c.Sort), Published: c.Published,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func (m collectionModel) toDomain() collections.Collection {
	rules := make([]collections.Rule, len(m.Rules.V))
	for i, r := range m.Rules.V {
		rules[i] = collections.Rule{Field: collections.RuleField(r.Field), Value: r.Value, IncludeSubcategories: r.IncludeSubcategories,
			MinMinorUnits: r.MinMinorUnits, MaxMinorUnits: r.MaxMinorUnits, Days: r.Days}
	}
	c := collections.Collection{ID: m.ID, TenantID: m.TenantID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
	c.Draft = collections.Draft{
		Name: m.Name, Slug: m.Slug, Description: m.Description, ImageAssetID: m.ImageAssetID, SEOTitle: m.SEOTitle,
		SEODescription: m.SEODescription, Type: collections.Type(m.Type), Match: collections.Match(m.Match), Rules: rules,
		Sort: collections.Sort(m.Sort), Published: m.Published,
	}
	if c.Type == collections.TypeManual {
		c.Rules = nil
	}
	return c
}

type CollectionRepo struct{ db *gorm.DB }

func NewCollectionRepo(db *gorm.DB) *CollectionRepo { return &CollectionRepo{db: db} }

var _ collections.Repository = (*CollectionRepo)(nil)

func slugConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "collections_tenant_slug_unique" {
		return collections.ErrSlugTaken
	}
	return err
}

func (r *CollectionRepo) Create(ctx context.Context, c *collections.Collection) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	m := toCollectionModel(*c)
	m.ID = ""
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("collectionRepo.Create: %w", slugConflict(err))
	}
	c.ID = m.ID
	return nil
}

func (r *CollectionRepo) find(ctx context.Context, where string, arg string) (*collections.Collection, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, err
	}
	var m collectionModel
	if err := tx.Where(where, arg).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, collections.ErrCollectionNotFound
		}
		return nil, fmt.Errorf("collectionRepo.find: %w", err)
	}
	c := m.toDomain()
	return &c, nil
}

func (r *CollectionRepo) FindByID(ctx context.Context, id string) (*collections.Collection, error) {
	return r.find(ctx, "id = ?", id)
}

func (r *CollectionRepo) FindBySlug(ctx context.Context, slug string) (*collections.Collection, error) {
	return r.find(ctx, "slug = ?", slug)
}

func (r *CollectionRepo) List(ctx context.Context, publishedOnly bool, limit, offset int) ([]collections.Collection, int64, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, 0, err
	}
	q := tx.Model(&collectionModel{})
	if publishedOnly {
		q = q.Where("published")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("collectionRepo.List count: %w", err)
	}
	var rows []collectionModel
	if err := q.Order("name, id").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("collectionRepo.List: %w", err)
	}
	out := make([]collections.Collection, len(rows))
	for i, m := range rows {
		out[i] = m.toDomain()
	}
	return out, total, nil
}

func (r *CollectionRepo) Update(ctx context.Context, c *collections.Collection) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	m := toCollectionModel(*c)
	if err := tx.Model(&collectionModel{}).Where("id = ?", c.ID).Updates(map[string]any{
		"name": m.Name, "slug": m.Slug, "description": m.Description, "image_asset_id": m.ImageAssetID,
		"seo_title": m.SEOTitle, "seo_description": m.SEODescription, "type": m.Type, "match": m.Match,
		"rules": m.Rules, "sort": m.Sort, "published": m.Published, "updated_at": m.UpdatedAt,
	}).Error; err != nil {
		return fmt.Errorf("collectionRepo.Update: %w", slugConflict(err))
	}
	return nil
}

func (r *CollectionRepo) Delete(ctx context.Context, id string) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Where("id = ?", id).Delete(&collectionModel{}).Error; err != nil {
		return fmt.Errorf("collectionRepo.Delete: %w", err)
	}
	return nil
}

func (r *CollectionRepo) ManualProducts(ctx context.Context, collectionID string) ([]string, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, err
	}
	var ids []string
	if err := tx.Raw(`SELECT product_id FROM collection_products WHERE collection_id = ? ORDER BY position`, collectionID).
		Scan(&ids).Error; err != nil {
		return nil, fmt.Errorf("collectionRepo.ManualProducts: %w", err)
	}
	return ids, nil
}

func (r *CollectionRepo) SetManualProducts(ctx context.Context, collectionID string, productIDs []string) error {
	tx, err := themeTx(ctx)
	if err != nil {
		return err
	}
	if err := tx.Exec(`DELETE FROM collection_products WHERE collection_id = ?`, collectionID).Error; err != nil {
		return fmt.Errorf("collectionRepo.SetManualProducts: %w", err)
	}
	if len(productIDs) == 0 {
		return nil
	}
	// One statement: unnest keeps the order as the position; products the
	// tenant can't see (RLS) or that don't exist are simply skipped.
	if err := tx.Exec(`INSERT INTO collection_products (tenant_id, collection_id, product_id, position)
		SELECT p.tenant_id, ?, p.id, ids.ord
		FROM unnest(?::uuid[]) WITH ORDINALITY AS ids(id, ord)
		JOIN products p ON p.id = ids.id`, collectionID, "{"+strings.Join(productIDs, ",")+"}").Error; err != nil {
		return fmt.Errorf("collectionRepo.SetManualProducts: %w", err)
	}
	return nil
}

const minPriceSQL = `(SELECT MIN(s.price_minor_units) FROM skus s JOIN variants v ON v.id = s.variant_id WHERE v.product_id = p.id)`

var sortSQL = map[collections.Sort]string{
	collections.SortNameAsc:   "p.name, p.id",
	collections.SortPriceAsc:  minPriceSQL + " ASC NULLS LAST, p.id",
	collections.SortPriceDesc: minPriceSQL + " DESC NULLS LAST, p.id",
	collections.SortNewest:    "p.created_at DESC, p.id",
}

// ruleSQL turns one rule into a SQL condition over products p. Only
// placeholders carry values -- nothing from the rule is spliced into SQL.
func ruleSQL(r collections.Rule) (string, []any) {
	switch r.Field {
	case collections.FieldCategory:
		if r.IncludeSubcategories {
			return `p.category_id IN (WITH RECURSIVE sub AS (
				SELECT id FROM categories WHERE id = ?::uuid
				UNION ALL SELECT c.id FROM categories c JOIN sub ON c.parent_id = sub.id
			) SELECT id FROM sub)`, []any{r.Value}
		}
		return `p.category_id = ?::uuid`, []any{r.Value}
	case collections.FieldTag:
		return `p.tags @> jsonb_build_array(?::text)`, []any{r.Value}
	case collections.FieldPrice:
		cond, args := []string{"v.product_id = p.id"}, []any{}
		if r.MinMinorUnits != nil {
			cond, args = append(cond, "s.price_minor_units >= ?"), append(args, *r.MinMinorUnits)
		}
		if r.MaxMinorUnits != nil {
			cond, args = append(cond, "s.price_minor_units <= ?"), append(args, *r.MaxMinorUnits)
		}
		return `EXISTS (SELECT 1 FROM skus s JOIN variants v ON v.id = s.variant_id WHERE ` + strings.Join(cond, " AND ") + `)`, args
	case collections.FieldOnSale:
		return `EXISTS (SELECT 1 FROM skus s JOIN variants v ON v.id = s.variant_id WHERE v.product_id = p.id
			AND s.sale_price_minor_units IS NOT NULL AND s.sale_price_minor_units < s.price_minor_units
			AND (s.sale_end_date IS NULL OR s.sale_end_date > NOW()))`, nil
	case collections.FieldAddedWithinDays:
		return `p.created_at >= NOW() - make_interval(days => ?)`, []any{r.Days}
	}
	return "FALSE", nil
}

func (r *CollectionRepo) ResolveProductIDs(ctx context.Context, c collections.Collection, limit int) ([]string, error) {
	tx, err := themeTx(ctx)
	if err != nil {
		return nil, err
	}
	var from, order string
	var args []any
	where := []string{"p.status = 'active'"}
	if c.Type == collections.TypeManual {
		from = `products p JOIN collection_products cp ON cp.product_id = p.id AND cp.collection_id = ?`
		args = append(args, c.ID)
		order = "cp.position"
	} else {
		from = "products p"
		var conds []string
		for _, rule := range c.Rules {
			sql, a := ruleSQL(rule)
			conds, args = append(conds, "("+sql+")"), append(args, a...)
		}
		join := " AND "
		if c.Match == collections.MatchAny {
			join = " OR "
		}
		if len(conds) == 0 {
			conds = []string{"FALSE"}
		}
		where = append(where, "("+strings.Join(conds, join)+")")
	}
	if s, ok := sortSQL[c.Sort]; ok {
		order = s
	}
	if order == "" {
		order = "p.name, p.id"
	}
	var ids []string
	query := `SELECT p.id FROM ` + from + ` WHERE ` + strings.Join(where, " AND ") + ` ORDER BY ` + order + ` LIMIT ?`
	if err := tx.Raw(query, append(args, limit)...).Scan(&ids).Error; err != nil {
		return nil, fmt.Errorf("collectionRepo.ResolveProductIDs: %w", err)
	}
	return ids, nil
}
