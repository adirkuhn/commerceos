//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/collections"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type catalogFixture struct {
	shoes, boots, hats  string // category ids (boots is a subcategory of shoes)
	red, blue, cap, old string // product ids
	draft               string // inactive product
}

// seedCatalog: red (shoes, tag summer, 30.00 on sale), blue (boots, 50.00),
// cap (hats, tag summer, 10.00), old (hats, 20.00, created 90 days ago),
// draft (shoes, draft status).
func seedCatalog(t *testing.T, ctx context.Context, db *gorm.DB) catalogFixture {
	t.Helper()
	var f catalogFixture
	products, variants, skus, cats := repositories.NewProductRepo(db), repositories.NewVariantRepo(db), repositories.NewSKURepo(db), repositories.NewCategoryRepo(db)
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		tid := tenant.IDFromContext(ctx)
		now := time.Now()
		mkCat := func(name string, parent *string) string {
			c := &pmsDomain.Category{TenantID: tid, Name: name, Slug: name, ParentID: parent, CreatedAt: now, UpdatedAt: now}
			require.NoError(t, cats.Create(ctx, c))
			return c.ID
		}
		f.shoes = mkCat("shoes", nil)
		f.boots = mkCat("boots", &f.shoes)
		f.hats = mkCat("hats", nil)
		mk := func(name string, cat string, tags []string, price int64, sale *int64, created time.Time, status pmsDomain.ProductStatus) string {
			p := &pmsDomain.Product{TenantID: tid, Name: name, Slug: name, CategoryID: &cat, Tags: tags, Status: status, CreatedAt: created, UpdatedAt: created}
			require.NoError(t, products.Create(ctx, p))
			v := &pmsDomain.Variant{TenantID: tid, ProductID: p.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
			require.NoError(t, variants.Create(ctx, v))
			require.NoError(t, skus.Create(ctx, &pmsDomain.SKU{TenantID: tid, VariantID: v.ID, SKUCode: "SKU-" + name, CurrencyCode: "EUR",
				PriceMinorUnits: price, SalePriceMinorUnits: sale, Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now}))
			return p.ID
		}
		sale := int64(2500)
		f.red = mk("red", f.shoes, []string{"summer"}, 3000, &sale, now.Add(-time.Hour), pmsDomain.ProductStatusActive)
		f.blue = mk("blue", f.boots, nil, 5000, nil, now.Add(-2*time.Hour), pmsDomain.ProductStatusActive)
		f.cap = mk("cap", f.hats, []string{"summer"}, 1000, nil, now.Add(-3*time.Hour), pmsDomain.ProductStatusActive)
		f.old = mk("old", f.hats, nil, 2000, nil, now.Add(-90*24*time.Hour), pmsDomain.ProductStatusActive)
		f.draft = mk("draft", f.shoes, []string{"summer"}, 100, nil, now, pmsDomain.ProductStatusDraft)
		return nil
	}))
	return f
}

func resolve(t *testing.T, ctx context.Context, db *gorm.DB, c collections.Collection) []string {
	t.Helper()
	var ids []string
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		ids, err = repositories.NewCollectionRepo(db).ResolveProductIDs(ctx, c, 100)
		return err
	}))
	return ids
}

func smart(match collections.Match, sort collections.Sort, rules ...collections.Rule) collections.Collection {
	return collections.Collection{Draft: collections.Draft{Type: collections.TypeSmart, Match: match, Sort: sort, Rules: rules}}
}

func TestCollectionRepo_Integration_smartRules(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	f := seedCatalog(t, ctx, db)
	i := func(v int64) *int64 { return &v }

	cases := map[string]struct {
		c    collections.Collection
		want []string
	}{
		"category only":        {smart(collections.MatchAll, collections.SortNameAsc, collections.Rule{Field: collections.FieldCategory, Value: f.shoes}), []string{f.red}},
		"category + subs":      {smart(collections.MatchAll, collections.SortNameAsc, collections.Rule{Field: collections.FieldCategory, Value: f.shoes, IncludeSubcategories: true}), []string{f.blue, f.red}},
		"tag skips inactive":   {smart(collections.MatchAll, collections.SortNameAsc, collections.Rule{Field: collections.FieldTag, Value: "summer"}), []string{f.cap, f.red}},
		"price range":          {smart(collections.MatchAll, collections.SortPriceAsc, collections.Rule{Field: collections.FieldPrice, MinMinorUnits: i(1500), MaxMinorUnits: i(3500)}), []string{f.old, f.red}},
		"on sale":              {smart(collections.MatchAll, collections.SortNewest, collections.Rule{Field: collections.FieldOnSale}), []string{f.red}},
		"added within 30 days": {smart(collections.MatchAll, collections.SortNewest, collections.Rule{Field: collections.FieldAddedWithinDays, Days: 30}), []string{f.red, f.blue, f.cap}},
		"all = and": {smart(collections.MatchAll, collections.SortNameAsc,
			collections.Rule{Field: collections.FieldTag, Value: "summer"}, collections.Rule{Field: collections.FieldCategory, Value: f.hats}), []string{f.cap}},
		"any = or": {smart(collections.MatchAny, collections.SortPriceDesc,
			collections.Rule{Field: collections.FieldOnSale}, collections.Rule{Field: collections.FieldCategory, Value: f.hats}), []string{f.red, f.old, f.cap}},
	}
	for name, tc := range cases {
		assert.Equal(t, tc.want, resolve(t, ctx, db, tc.c), name)
	}
}

func TestCollectionRepo_Integration_manualCrudAndOrder(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	f := seedCatalog(t, ctx, db)
	repo := repositories.NewCollectionRepo(db)

	var c collections.Collection
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		c, err = collections.New(tenant.IDFromContext(ctx), collections.Draft{Name: "Picks", Slug: "picks", Type: collections.TypeManual, Sort: collections.SortManual}, time.Now())
		require.NoError(t, err)
		require.NoError(t, repo.Create(ctx, &c))
		return repo.SetManualProducts(ctx, c.ID, []string{f.cap, f.draft, f.red})
	}))
	// A failed insert aborts its transaction, so the duplicate gets its own.
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		dup := c
		dup.ID = ""
		return repo.Create(ctx, &dup)
	})
	assert.ErrorIs(t, err, collections.ErrSlugTaken)

	assert.Equal(t, []string{f.cap, f.red}, resolve(t, ctx, db, c), "manual order, inactive left out")
	c.Sort = collections.SortPriceDesc
	assert.Equal(t, []string{f.red, f.cap}, resolve(t, ctx, db, c))

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		got, err := repo.FindBySlug(ctx, "picks")
		require.NoError(t, err)
		assert.Equal(t, "Picks", got.Name)
		ids, err := repo.ManualProducts(ctx, c.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{f.cap, f.draft, f.red}, ids)

		got.Published = true
		require.NoError(t, repo.Update(ctx, got))
		pub, total, err := repo.List(ctx, true, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.True(t, pub[0].Published)

		require.NoError(t, repo.Delete(ctx, c.ID))
		_, err = repo.FindByID(ctx, c.ID)
		assert.ErrorIs(t, err, collections.ErrCollectionNotFound)
		return nil
	}))
}

func TestCollectionRepo_Integration_rlsIsolatesTenants(t *testing.T) {
	_, dsn := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, dsn)
	_, ctxA := newTestTenant()
	_, ctxB := newTestTenant()
	repo := repositories.NewCollectionRepo(db)
	var id string
	require.NoError(t, tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		c, _ := collections.New(tenant.IDFromContext(ctx), collections.Draft{Name: "A", Slug: "a", Type: collections.TypeManual, Sort: collections.SortManual}, time.Now())
		err := repo.Create(ctx, &c)
		id = c.ID
		return err
	}))
	require.NoError(t, tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		_, err := repo.FindByID(ctx, id)
		assert.ErrorIs(t, err, collections.ErrCollectionNotFound)
		c, _ := collections.New(tenant.IDFromContext(ctx), collections.Draft{Name: "A", Slug: "a", Type: collections.TypeManual, Sort: collections.SortManual}, time.Now())
		assert.NoError(t, repo.Create(ctx, &c), "slugs are unique per tenant, not globally")
		return nil
	}))
}

func TestProductRepo_Integration_orderByIDs(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	f := seedCatalog(t, ctx, db)
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		ids := []string{f.old, f.red, f.cap}
		list, total, err := repositories.NewProductRepo(db).List(ctx, pmsDomain.ProductFilter{AndIncludeIDs: &ids, OrderByIDs: ids, Limit: 2, Offset: 1})
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		require.Len(t, list, 2)
		assert.Equal(t, []string{f.red, f.cap}, []string{list[0].ID, list[1].ID})
		return nil
	}))
}
