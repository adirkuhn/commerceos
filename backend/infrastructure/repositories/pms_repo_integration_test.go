//go:build integration

package repositories_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestProductRepo_Integration_SearchIsAccentInsensitive proves the
// unaccent()-based search (022_unaccent_extension.sql) actually works
// against real Postgres -- sqlmock-based tests can only assert the SQL
// string sent, not that unaccent() as a function actually exists and
// behaves as expected in the target database.
func TestProductRepo_Integration_SearchIsAccentInsensitive(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		for _, name := range []string{"Açaí na tigela", "Feijão preto", "Pão de queijo"} {
			p := &pmsDomain.Product{
				TenantID: tenant.IDFromContext(ctx), Name: name, Slug: name,
				Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now,
			}
			if err := productRepo.Create(ctx, p); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	for _, tc := range []struct {
		query string
		want  string
	}{
		{"acai", "Açaí na tigela"},
		{"AÇAÍ", "Açaí na tigela"},
		{"feijao", "Feijão preto"},
		{"pao", "Pão de queijo"},
	} {
		var products []pmsDomain.Product
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			search := tc.query
			products, _, err = productRepo.List(ctx, pmsDomain.ProductFilter{Search: &search})
			return err
		})
		require.NoError(t, err)
		require.Len(t, products, 1, "query %q should match exactly one product", tc.query)
		require.Equal(t, tc.want, products[0].Name, "query %q", tc.query)
	}
}

// TestProductRepo_Integration_CountByCategory proves the GROUP BY query
// actually runs against real Postgres and that AndCategoryID is ignored
// (see applyProductFilter's doc comment) -- both categories' counts come
// back even though the filter passed in is scoped to just one of them.
func TestProductRepo_Integration_CountByCategory(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	categoryRepo := repositories.NewCategoryRepo(db)

	var categoryAID, categoryBID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		tid := tenant.IDFromContext(ctx)
		categoryA := &pmsDomain.Category{TenantID: tid, Name: "Category A", Slug: "category-a-count-by-category"}
		if err := categoryRepo.Create(ctx, categoryA); err != nil {
			return err
		}
		categoryAID = categoryA.ID
		categoryB := &pmsDomain.Category{TenantID: tid, Name: "Category B", Slug: "category-b-count-by-category"}
		if err := categoryRepo.Create(ctx, categoryB); err != nil {
			return err
		}
		categoryBID = categoryB.ID

		now := time.Now()
		for i, catID := range []string{categoryAID, categoryAID, categoryBID} {
			id := catID
			p := &pmsDomain.Product{
				TenantID: tid, Name: fmt.Sprintf("Product %d", i), Slug: fmt.Sprintf("count-by-category-product-%d", i),
				CategoryID: &id, Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now,
			}
			if err := productRepo.Create(ctx, p); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	var counts map[string]int64
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		counts, err = productRepo.CountByCategory(ctx, pmsDomain.ProductFilter{AndCategoryID: &categoryAID})
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), counts[categoryAID])
	assert.Equal(t, int64(1), counts[categoryBID])
}

// TestProductRepo_Integration_UpdatedAfterFiltersByRealUpdatedAtColumn
// proves the updated_after filter -- the incremental-sync filter external
// integrations (e.g. a POS desktop) use to pull only what changed -- runs
// against the real updated_at column and picks up a real Update, not just
// CreatedAt.
func TestProductRepo_Integration_UpdatedAfterFiltersByRealUpdatedAtColumn(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)

	var recentProductID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		old := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Old Product", Slug: "old-product-updated-after",
			Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := productRepo.Create(ctx, old); err != nil {
			return err
		}
		recent := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Recent Product", Slug: "recent-product-updated-after",
			Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := productRepo.Create(ctx, recent); err != nil {
			return err
		}
		recentProductID = recent.ID
		return nil
	})
	require.NoError(t, err)

	cutoff := time.Now()

	// Only the recent product gets touched again, after the cutoff.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		recent, err := productRepo.FindByID(ctx, recentProductID)
		if err != nil {
			return err
		}
		recent.Name = "Recent Product (touched)"
		// ProductRepo.Update writes UpdatedAt as-is from the struct (bumping
		// it is UpdateProductUseCase's job, not the repo's) -- set it
		// explicitly here, same as that use case does.
		recent.UpdatedAt = time.Now()
		return productRepo.Update(ctx, recent)
	})
	require.NoError(t, err)

	var filtered []pmsDomain.Product
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		filtered, _, err = productRepo.List(ctx, pmsDomain.ProductFilter{UpdatedAfter: &cutoff})
		return err
	})
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	require.Equal(t, recentProductID, filtered[0].ID)
}

// TestProductRepo_Integration_DefaultOrderIsMostRecentlyCreatedFirst proves
// List's no-search default order is created_at desc, not alphabetical --
// so a just-created or just-duplicated product is immediately visible at
// the top of the list.
func TestProductRepo_Integration_DefaultOrderIsMostRecentlyCreatedFirst(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)

	var firstID, secondID, thirdID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		base := time.Now().Add(-time.Hour)
		// Named so alphabetical order would put this LAST -- proves the
		// default isn't "name asc" anymore.
		first := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Zzz First Created", Slug: "zzz-first-created-order-test",
			Status: pmsDomain.ProductStatusActive, CreatedAt: base, UpdatedAt: base,
		}
		if err := productRepo.Create(ctx, first); err != nil {
			return err
		}
		firstID = first.ID

		second := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Aaa Second Created", Slug: "aaa-second-created-order-test",
			Status: pmsDomain.ProductStatusActive, CreatedAt: base.Add(time.Minute), UpdatedAt: base.Add(time.Minute),
		}
		if err := productRepo.Create(ctx, second); err != nil {
			return err
		}
		secondID = second.ID

		third := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Mmm Third Created", Slug: "mmm-third-created-order-test",
			Status: pmsDomain.ProductStatusActive, CreatedAt: base.Add(2 * time.Minute), UpdatedAt: base.Add(2 * time.Minute),
		}
		if err := productRepo.Create(ctx, third); err != nil {
			return err
		}
		thirdID = third.ID
		return nil
	})
	require.NoError(t, err)

	var byID = map[string]bool{firstID: true, secondID: true, thirdID: true}
	var ordered []pmsDomain.Product
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		all, _, err := productRepo.List(ctx, pmsDomain.ProductFilter{Limit: 1000})
		if err != nil {
			return err
		}
		for _, p := range all {
			if byID[p.ID] {
				ordered = append(ordered, p)
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.Len(t, ordered, 3)
	require.Equal(t, []string{thirdID, secondID, firstID}, []string{ordered[0].ID, ordered[1].ID, ordered[2].ID})
}

// TestProductRepo_Integration_SearchMatchesSKUCodeAndBarcode proves the
// EXISTS-subquery join to variants/skus actually works against real
// Postgres, and doesn't return a product more than once for a product with
// several SKUs (a plain JOIN would duplicate rows here).
func TestProductRepo_Integration_SearchMatchesSKUCodeAndBarcode(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		p := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Widget Multipack", Slug: "widget-multipack-search",
			Status: pmsDomain.ProductStatusActive, HasVariants: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := productRepo.Create(ctx, p); err != nil {
			return err
		}
		v := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: p.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, v); err != nil {
			return err
		}
		// Two SKUs under the same product -- the search must still return
		// the product exactly once, matching on either one.
		for _, sku := range []*pmsDomain.SKU{
			{TenantID: tenant.IDFromContext(ctx), VariantID: v.ID, SKUCode: "WGT-SEARCH-1", Barcode: "0123456789012", CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now},
			{TenantID: tenant.IDFromContext(ctx), VariantID: v.ID, SKUCode: "WGT-SEARCH-2", CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now},
		} {
			if err := skuRepo.Create(ctx, sku); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	for _, query := range []string{"WGT-SEARCH-1", "wgt-search-2", "0123456789012"} {
		var products []pmsDomain.Product
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			search := query
			products, _, err = productRepo.List(ctx, pmsDomain.ProductFilter{Search: &search})
			return err
		})
		require.NoError(t, err, "query %q", query)
		require.Len(t, products, 1, "query %q should match the product exactly once, not once per matching sku", query)
		require.Equal(t, "Widget Multipack", products[0].Name, "query %q", query)
	}

	var noMatch []pmsDomain.Product
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		search := "no-such-code-or-name"
		noMatch, _, err = productRepo.List(ctx, pmsDomain.ProductFilter{Search: &search})
		return err
	})
	require.NoError(t, err)
	require.Empty(t, noMatch)
}

// TestProductRepo_Integration_SearchIsTypoTolerant proves the pg_trgm-based
// fuzzy fallback (077_pg_trgm_extension.sql) actually works against real
// Postgres -- a plain unit test with sqlmock can't verify the similarity()
// function/gin_trgm_ops index exist and behave as expected in the target
// database.
func TestProductRepo_Integration_SearchIsTypoTolerant(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		for _, name := range []string{"Widget", "Gadget", "Contraption"} {
			p := &pmsDomain.Product{
				TenantID: tenant.IDFromContext(ctx), Name: name, Slug: name,
				Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now,
			}
			if err := productRepo.Create(ctx, p); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	for _, tc := range []struct {
		query string
		want  string
	}{
		{"Wigdet", "Widget"},  // transposed letters
		{"Widgt", "Widget"},   // dropped letter
		{"Gadgett", "Gadget"}, // extra letter
	} {
		var products []pmsDomain.Product
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			search := tc.query
			products, _, err = productRepo.List(ctx, pmsDomain.ProductFilter{Search: &search})
			return err
		})
		require.NoError(t, err, "query %q", tc.query)
		require.NotEmpty(t, products, "query %q should fuzzy-match something", tc.query)
		require.Equal(t, tc.want, products[0].Name, "query %q should rank %q first", tc.query, tc.want)
	}

	// Something with no plausible relationship to any product name must
	// still come back empty -- the fuzzy fallback isn't a match-everything.
	var noMatch []pmsDomain.Product
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		search := "zzzznonexistentqqqq"
		noMatch, _, err = productRepo.List(ctx, pmsDomain.ProductFilter{Search: &search})
		return err
	})
	require.NoError(t, err)
	require.Empty(t, noMatch)
}

// TestProductRepo_Integration_SearchRanksBarcodeMatchAboveUnrelatedNameFuzzyMatch
// proves the "barcode first" priority: a product whose barcode/sku_code
// exactly matches the search term must outrank a product that only matches
// by name-similarity (pg_trgm), even when the name-similarity hit scores
// higher raw similarity (a name literally containing the search term as a
// substring scores very high). Without this ranking, a real barcode scan
// can get pushed past a small Limit by irrelevant fuzzy name noise -- see
// 077_pg_trgm_extension.sql's ORDER BY, which ranked by name similarity
// alone.
func TestProductRepo_Integration_SearchRanksBarcodeMatchAboveUnrelatedNameFuzzyMatch(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)

	const code = "SUPERCODE123"
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		// Matches by barcode only -- name is completely unrelated to the
		// search term, so its name-similarity score is near zero.
		barcodeMatch := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Random Gadget", Slug: "random-gadget-barcode-first",
			Status: pmsDomain.ProductStatusActive, HasVariants: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := productRepo.Create(ctx, barcodeMatch); err != nil {
			return err
		}
		v1 := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: barcodeMatch.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, v1); err != nil {
			return err
		}
		if err := skuRepo.Create(ctx, &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: v1.ID, SKUCode: "RG-1", Barcode: code,
			CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return err
		}

		// Matches by name only -- the name contains the search term
		// verbatim, so it scores a very high (likely higher) similarity,
		// but has nothing to do with the actual barcode being searched.
		nameMatch := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: code + " Deluxe Edition", Slug: "deluxe-edition-barcode-first",
			Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		return productRepo.Create(ctx, nameMatch)
	})
	require.NoError(t, err)

	var products []pmsDomain.Product
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		search := code
		products, _, err = productRepo.List(ctx, pmsDomain.ProductFilter{Search: &search, Limit: 1})
		return err
	})
	require.NoError(t, err)

	require.Len(t, products, 1)
	require.Equal(t, "Random Gadget", products[0].Name, "the barcode-matching product must rank first (and survive Limit:1) even though the other product's name scores higher fuzzy similarity")
}

// TestProductRepo_Integration_SyncVariantsAsWooVariationsRoundTrips proves
// migration 046's products.sync_variants_as_woo_variations column exists,
// defaults to false, and survives a real round trip through GORM's
// Create/Update -- a sqlmock test can only assert the SQL string sent, not
// that the column is actually there with the right default in the target
// schema.
func TestProductRepo_Integration_SyncVariantsAsWooVariationsRoundTrips(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)

	var productID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		p := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Carne Seca", Slug: "carne-seca-046",
			Status: pmsDomain.ProductStatusActive, HasVariants: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := productRepo.Create(ctx, p); err != nil {
			return err
		}
		productID = p.ID
		return nil
	})
	require.NoError(t, err)

	var fetched *pmsDomain.Product
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		fetched, err = productRepo.FindByID(ctx, productID)
		return err
	})
	require.NoError(t, err)
	require.False(t, fetched.SyncVariantsAsWooVariations, "defaults to false")

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		fetched.SyncVariantsAsWooVariations = true
		return productRepo.Update(ctx, fetched)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		fetched, err = productRepo.FindByID(ctx, productID)
		return err
	})
	require.NoError(t, err)
	require.True(t, fetched.SyncVariantsAsWooVariations)
}

// TestSKURepo_Integration_BulkPoolRoundTripsAndCascadesOnDelete proves
// migration 047/048's unit_type/portioned_from_sku_id columns exist,
// round-trip through GORM's Create/Update, and that the self-referential
// FK is ON DELETE SET NULL: deleting the bulk SKU must not delete the
// portioned SKU, just clear its link to the (now-gone) pool.
// TestProductRepo_Integration_SlugRoundTripsThroughUpdate proves Update's
// map-based Updates() call actually includes the slug column -- a field
// present in the map-literal Create path but missing from Update's map
// would silently drop every slug edit with no error, exactly the "slug not
// saved" bug this guards against.
func TestProductRepo_Integration_SlugRoundTripsThroughUpdate(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)

	var productID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		p := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Carne Seca", Slug: "carne-seca-original",
			Status: pmsDomain.ProductStatusActive, HasVariants: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := productRepo.Create(ctx, p); err != nil {
			return err
		}
		productID = p.ID
		return nil
	})
	require.NoError(t, err)

	var fetched *pmsDomain.Product
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		fetched, err = productRepo.FindByID(ctx, productID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "carne-seca-original", fetched.Slug)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		fetched.Slug = "carne-seca-renamed"
		return productRepo.Update(ctx, fetched)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		fetched, err = productRepo.FindByID(ctx, productID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "carne-seca-renamed", fetched.Slug)
}

func TestSKURepo_Integration_BulkPoolRoundTripsAndCascadesOnDelete(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)

	var bulkSKUID, portionedSKUID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		product := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Carne Seca 047", Slug: "carne-seca-047",
			Status: pmsDomain.ProductStatusActive, HasVariants: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		bulkVariant := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Bulk", CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, bulkVariant); err != nil {
			return err
		}
		bulkSKU := &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: bulkVariant.ID, SKUCode: "BULK-047",
			CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, UnitType: pmsDomain.SKUUnitTypeKilo, CreatedAt: now, UpdatedAt: now,
		}
		if err := skuRepo.Create(ctx, bulkSKU); err != nil {
			return err
		}
		bulkSKUID = bulkSKU.ID

		portionedVariant := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "378g", CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, portionedVariant); err != nil {
			return err
		}
		weight := int64(378)
		portionedSKU := &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: portionedVariant.ID, SKUCode: "PORTION-047",
			CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, UnitType: pmsDomain.SKUUnitTypeUnit,
			WeightGrams: &weight, PortionedFromSKUID: &bulkSKUID, CreatedAt: now, UpdatedAt: now,
		}
		if err := skuRepo.Create(ctx, portionedSKU); err != nil {
			return err
		}
		portionedSKUID = portionedSKU.ID
		return nil
	})
	require.NoError(t, err)

	var fetched *pmsDomain.SKU
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		fetched, err = skuRepo.FindByID(ctx, portionedSKUID)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, fetched.PortionedFromSKUID)
	require.Equal(t, bulkSKUID, *fetched.PortionedFromSKUID)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return skuRepo.Delete(ctx, bulkSKUID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		fetched, err = skuRepo.FindByID(ctx, portionedSKUID)
		return err
	})
	require.NoError(t, err, "the portioned SKU itself must survive the bulk SKU's deletion")
	require.Nil(t, fetched.PortionedFromSKUID, "ON DELETE SET NULL clears the now-dangling link")
}

// TestSKURepo_Integration_FindByExactProductName proves the real
// normalized-equality SQL (case/punctuation/whitespace-insensitive, same
// convention as suppliers.FindByName) actually resolves against Postgres --
// sqlmock can't prove the SQL expression itself is valid or behaves as
// expected.
func TestSKURepo_Integration_FindByExactProductName(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)

	var skuID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		product := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Carne Seca ~414g", Slug: "carne-seca-414g",
			Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		variant := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: product.Name, CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, variant); err != nil {
			return err
		}
		sku := &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "CARNE-SECA-414",
			CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, UnitType: pmsDomain.SKUUnitTypeUnit, CreatedAt: now, UpdatedAt: now,
		}
		if err := skuRepo.Create(ctx, sku); err != nil {
			return err
		}
		skuID = sku.ID
		return nil
	})
	require.NoError(t, err)

	for _, query := range []string{"Carne Seca ~414g", "carne seca ~414g", "  CARNE SECA ~414G  ", "Carne, Seca ~414g."} {
		var found *pmsDomain.SKU
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			found, err = skuRepo.FindByExactProductName(ctx, query)
			return err
		})
		require.NoError(t, err, "query %q", query)
		require.NotNil(t, found, "query %q", query)
		assert.Equal(t, skuID, found.ID, "query %q", query)
	}

	var notFound *pmsDomain.SKU
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		notFound, err = skuRepo.FindByExactProductName(ctx, "Some Other Product")
		return err
	})
	require.ErrorIs(t, err, pmsDomain.ErrSKUNotFound)
	assert.Nil(t, notFound)
}

// TestProductRepo_Integration_DeleteBlockedByReceivingHistory proves
// ProductRepo.Delete maps the booking_scans_sku_id_fkey violation (ON
// DELETE RESTRICT -- see 031_scanner.sql) to the typed
// pmsDomain.ErrProductHasReceivingHistory instead of leaking the raw
// Postgres error, when a product's SKU was scanned into a purchase-order
// booking.
func TestProductRepo_Integration_DeleteBlockedByReceivingHistory(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	bookingRepo := repositories.NewBookingRepo(db)
	scanRepo := repositories.NewBookingScanRepo(db)

	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	var productID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		sku, err := skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		variant, err := variantRepo.FindByID(ctx, sku.VariantID)
		if err != nil {
			return err
		}
		productID = variant.ProductID

		now := time.Now()
		supplier := &suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx), Name: "Acme Foods", Status: suppliersDomain.SupplierStatusActive,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := supplierRepo.Create(ctx, supplier); err != nil {
			return err
		}
		po := procurementDomain.PurchaseOrder{
			TenantID: tenant.IDFromContext(ctx), SupplierID: &supplier.ID, WarehouseID: &warehouseID,
			Status: procurementDomain.PurchaseOrderStatusSent, CurrencyCode: "USD", CreatedAt: now, UpdatedAt: now,
		}
		if err := poRepo.Create(ctx, &po); err != nil {
			return err
		}
		booking := procurementDomain.Booking{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: po.ID,
			Status: procurementDomain.BookingStatusInProgress, StartedAt: now,
		}
		if err := bookingRepo.Create(ctx, &booking); err != nil {
			return err
		}
		scan := procurementDomain.BookingScan{
			TenantID: tenant.IDFromContext(ctx), BookingID: booking.ID, SKUID: skuID,
			Quantity: 5, ScannedAt: now, ScannedBy: "user-1",
		}
		return scanRepo.Create(ctx, &scan)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return productRepo.Delete(ctx, productID)
	})
	require.ErrorIs(t, err, pmsDomain.ErrProductHasReceivingHistory)
}

// TestProductRepo_Integration_DeleteBlockedBySalesOrderHistory proves
// ProductRepo.Delete maps the sales_order_line_items_sku_id_fkey violation
// (ON DELETE RESTRICT -- see 052_sales_orders.sql) to the typed
// pmsDomain.ErrProductHasSalesOrderHistory instead of leaking the raw
// Postgres error, when a product's SKU was sold on a sales order.
func TestProductRepo_Integration_DeleteBlockedBySalesOrderHistory(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)

	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)

	var productID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		sku, err := skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		variant, err := variantRepo.FindByID(ctx, sku.VariantID)
		if err != nil {
			return err
		}
		productID = variant.ProductID

		now := time.Now()
		order := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourcePOS,
			ExternalOrderID: "pos-delete-history-1", WarehouseID: warehouseID, CurrencyCode: "USD",
			OccurredAt: now, CreatedAt: now,
		}
		if err := salesOrderRepo.Create(ctx, &order); err != nil {
			return err
		}
		lineItem := omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: order.ID, SKUID: &skuID,
			Quantity: 1, PriceMinorUnits: 500, CreatedAt: now,
		}
		return lineItemRepo.Create(ctx, &lineItem)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return productRepo.Delete(ctx, productID)
	})
	require.ErrorIs(t, err, pmsDomain.ErrProductHasSalesOrderHistory)
}

// TestSKURepo_Integration_UpdateCostPriceIsIndependentOfUpdate proves the
// deliberate split between the narrow UpdateCostPrice (the cron's sole
// writer) and the ordinary Update (the product-edit save path): saving a
// SKU through Update must never wipe out a cost price the cron already
// computed.
func TestSKURepo_Integration_UpdateCostPriceIsIndependentOfUpdate(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)

	var skuID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		product := &pmsDomain.Product{TenantID: tenant.IDFromContext(ctx), Name: "Widget", Slug: "widget-cost-price", Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		variant := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, variant); err != nil {
			return err
		}
		sku := &pmsDomain.SKU{TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "WID-COST-1", PriceMinorUnits: 1000, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := skuRepo.Create(ctx, sku); err != nil {
			return err
		}
		skuID = sku.ID
		return skuRepo.UpdateCostPrice(ctx, skuID, 700)
	})
	require.NoError(t, err)

	var fetched *pmsDomain.SKU
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		fetched, err = skuRepo.FindByID(ctx, skuID)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, fetched.CostPriceMinorUnits)
	require.Equal(t, int64(700), *fetched.CostPriceMinorUnits)

	// An ordinary product-edit save (Update) must not touch cost_price_minor_units.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		fetched.PriceMinorUnits = 1200
		return skuRepo.Update(ctx, fetched)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		fetched, err = skuRepo.FindByID(ctx, skuID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(1200), fetched.PriceMinorUnits)
	require.NotNil(t, fetched.CostPriceMinorUnits, "Update must not clear a cost price only UpdateCostPrice should touch")
	require.Equal(t, int64(700), *fetched.CostPriceMinorUnits)
}

// TestSKURepo_Integration_UpdateRenamesSKUCodeAndEnforcesUniqueness proves
// SKUCode round-trips through Update against real Postgres (the repo's
// column map used to omit it entirely), and that the DB's own
// (tenant_id, sku_code) unique constraint still backstops the app-level
// check -- surfacing as ErrDuplicateSKUCode, not a raw Postgres error.
func TestSKURepo_Integration_UpdateRenamesSKUCodeAndEnforcesUniqueness(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)

	var skuAID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		product := &pmsDomain.Product{TenantID: tenant.IDFromContext(ctx), Name: "Widget", Slug: "widget-sku-rename", Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		variant := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, variant); err != nil {
			return err
		}
		skuA := &pmsDomain.SKU{TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "WID-RENAME-A", PriceMinorUnits: 1000, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := skuRepo.Create(ctx, skuA); err != nil {
			return err
		}
		skuAID = skuA.ID
		skuB := &pmsDomain.SKU{TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "WID-RENAME-B", PriceMinorUnits: 1000, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now}
		return skuRepo.Create(ctx, skuB)
	})
	require.NoError(t, err)

	// Rename actually persists.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		skuA, err := skuRepo.FindByID(ctx, skuAID)
		if err != nil {
			return err
		}
		skuA.SKUCode = "WID-RENAME-A2"
		return skuRepo.Update(ctx, skuA)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		fetched, err := skuRepo.FindByID(ctx, skuAID)
		if err != nil {
			return err
		}
		require.Equal(t, "WID-RENAME-A2", fetched.SKUCode)
		return nil
	})
	require.NoError(t, err)

	// The DB constraint itself rejects a collision that somehow bypassed
	// the app-level check, mapped to the typed domain error.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		skuA, err := skuRepo.FindByID(ctx, skuAID)
		if err != nil {
			return err
		}
		skuA.SKUCode = "WID-RENAME-B"
		return skuRepo.Update(ctx, skuA)
	})
	require.ErrorIs(t, err, pmsDomain.ErrDuplicateSKUCode)
}

// TestCategoryRepo_Integration_TargetMarkupBasisPointsRoundTripsAndClears
// proves the category markup override persists and can be cleared back to
// nil (falls back to the tenant default -- see
// application/pms.EffectiveTargetMarkupBasisPoints).
func TestCategoryRepo_Integration_TargetMarkupBasisPointsRoundTripsAndClears(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	categoryRepo := repositories.NewCategoryRepo(db)

	var categoryID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		markup := 3500
		category := &pmsDomain.Category{TenantID: tenant.IDFromContext(ctx), Name: "Beverages", Slug: "beverages-markup-test", TargetMarkupBasisPoints: &markup, CreatedAt: now, UpdatedAt: now}
		if err := categoryRepo.Create(ctx, category); err != nil {
			return err
		}
		categoryID = category.ID
		return nil
	})
	require.NoError(t, err)

	var fetched *pmsDomain.Category
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		fetched, err = categoryRepo.FindByID(ctx, categoryID)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, fetched.TargetMarkupBasisPoints)
	require.Equal(t, 3500, *fetched.TargetMarkupBasisPoints)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		fetched.TargetMarkupBasisPoints = nil
		fetched.UpdatedAt = time.Now()
		return categoryRepo.Update(ctx, fetched)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		fetched, err = categoryRepo.FindByID(ctx, categoryID)
		return err
	})
	require.NoError(t, err)
	require.Nil(t, fetched.TargetMarkupBasisPoints)
}

// TestSKUCostPriceHistoryRepo_Integration_UpsertIsIdempotentPerDay proves
// the same-day cron rerun the daily cost-price cron relies on: a second
// Upsert for the same (sku_id, occurred_on) overwrites rather than
// duplicating.
func TestSKUCostPriceHistoryRepo_Integration_UpsertIsIdempotentPerDay(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	historyRepo := repositories.NewSKUCostPriceHistoryRepo(db)

	var skuID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		product := &pmsDomain.Product{TenantID: tenant.IDFromContext(ctx), Name: "Widget", Slug: "widget-history-test", Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		variant := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, variant); err != nil {
			return err
		}
		sku := &pmsDomain.SKU{TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "WID-HIST-1", PriceMinorUnits: 1000, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := skuRepo.Create(ctx, sku); err != nil {
			return err
		}
		skuID = sku.ID
		return nil
	})
	require.NoError(t, err)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return historyRepo.Upsert(ctx, &pmsDomain.SKUCostPriceHistory{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID,
			CostPriceMinorUnits: 700, RecommendedPriceMinorUnits: 1000, OccurredOn: today, CreatedAt: time.Now(),
		})
	})
	require.NoError(t, err)
	// Rerun the same day with a different cost -- must overwrite, not add a row.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return historyRepo.Upsert(ctx, &pmsDomain.SKUCostPriceHistory{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID,
			CostPriceMinorUnits: 750, RecommendedPriceMinorUnits: 1071, OccurredOn: today, CreatedAt: time.Now(),
		})
	})
	require.NoError(t, err)

	var history []pmsDomain.SKUCostPriceHistory
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		history, err = historyRepo.ListBySKU(ctx, skuID, 0)
		return err
	})
	require.NoError(t, err)
	require.Len(t, history, 1, "same-day rerun must overwrite, not duplicate")
	require.Equal(t, int64(750), history[0].CostPriceMinorUnits)
}

// TestSKUCostPriceHistoryRepo_Integration_RLSIsolatesTenants proves the new
// table's tenant_isolation policy actually enforces isolation (not just
// that the SQL runs) -- routed through commerceos_app, not the
// migration-owning superuser, same reasoning as
// TestBatchRepo_Integration_RLSHidesOtherTenantsBatches: RLS is never
// enforced against a superuser/BYPASSRLS role regardless of policy, so
// this test only means something over the same non-superuser connection
// production actually uses.
func TestSKUCostPriceHistoryRepo_Integration_RLSIsolatesTenants(t *testing.T) {
	superDB, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	productRepo := repositories.NewProductRepo(superDB)
	variantRepo := repositories.NewVariantRepo(superDB)
	skuRepo := repositories.NewSKURepo(superDB)
	historyRepo := repositories.NewSKUCostPriceHistoryRepo(db)

	var skuID string
	err := tenant.WithTenant(ctxA, superDB, func(ctx context.Context) error {
		now := time.Now()
		product := &pmsDomain.Product{TenantID: tenant.IDFromContext(ctx), Name: "Widget", Slug: "widget-rls-test", Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		variant := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, variant); err != nil {
			return err
		}
		sku := &pmsDomain.SKU{TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "WID-RLS-1", PriceMinorUnits: 1000, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := skuRepo.Create(ctx, sku); err != nil {
			return err
		}
		skuID = sku.ID
		return nil
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		return historyRepo.Upsert(ctx, &pmsDomain.SKUCostPriceHistory{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID,
			CostPriceMinorUnits: 700, RecommendedPriceMinorUnits: 1000, OccurredOn: time.Now().UTC().Truncate(24 * time.Hour), CreatedAt: time.Now(),
		})
	})
	require.NoError(t, err)

	_, ctxB := newTestTenant()
	var historyFromOtherTenant []pmsDomain.SKUCostPriceHistory
	err = tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		var err error
		historyFromOtherTenant, err = historyRepo.ListBySKU(ctx, skuID, 0)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, historyFromOtherTenant, "tenant B must not see tenant A's cost-price history")
}

// TestSKUPriceHistoryRepo_Integration_CloseOpenEndsThePreviousIntervalOnly
// proves CloseOpen only closes the currently-open row for that SKU (leaving
// older, already-closed intervals untouched) and that a subsequent Create
// opens a fresh interval -- the sequence UpdateSKUUseCase/
// ExpireSKUSalePriceUseCase rely on.
func TestSKUPriceHistoryRepo_Integration_CloseOpenEndsThePreviousIntervalOnly(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	historyRepo := repositories.NewSKUPriceHistoryRepo(db)

	var skuID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		product := &pmsDomain.Product{TenantID: tenant.IDFromContext(ctx), Name: "Acai", Slug: "acai-price-history-test", Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		variant := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, variant); err != nil {
			return err
		}
		sku := &pmsDomain.SKU{TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "ACAI-HIST-1", PriceMinorUnits: 999, CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := skuRepo.Create(ctx, sku); err != nil {
			return err
		}
		skuID = sku.ID
		return nil
	})
	require.NoError(t, err)

	// First interval: opened at creation.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return historyRepo.Create(ctx, &pmsDomain.SKUPriceHistory{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID,
			PriceMinorUnits: 999, CurrencyCode: "EUR",
			ValidFrom: time.Now(), Reason: pmsDomain.PriceChangeReasonCreated, CreatedAt: time.Now(),
		})
	})
	require.NoError(t, err)

	// Price change: close the open interval, open a new one.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := historyRepo.CloseOpen(ctx, skuID, time.Now()); err != nil {
			return err
		}
		return historyRepo.Create(ctx, &pmsDomain.SKUPriceHistory{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID,
			PriceMinorUnits: 1049, CurrencyCode: "EUR",
			ValidFrom: time.Now(), Reason: pmsDomain.PriceChangeReasonManualUpdate, CreatedAt: time.Now(),
		})
	})
	require.NoError(t, err)

	var history []pmsDomain.SKUPriceHistory
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		history, err = historyRepo.ListBySKU(ctx, skuID, 0)
		return err
	})
	require.NoError(t, err)
	require.Len(t, history, 2)
	// Most recent (valid_from desc) first: the new open interval.
	require.Equal(t, int64(1049), history[0].PriceMinorUnits)
	require.Nil(t, history[0].ValidUntil)
	require.Equal(t, int64(999), history[1].PriceMinorUnits)
	require.NotNil(t, history[1].ValidUntil, "the superseded interval must be closed")

	// A second CloseOpen (nothing open right now) must be a no-op, not an error.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return historyRepo.CloseOpen(ctx, skuID, time.Now())
	})
	require.NoError(t, err)
}

// TestSKUPriceHistoryRepo_Integration_RLSIsolatesTenants proves the new
// table's tenant_isolation policy actually enforces isolation over the
// non-superuser app role, same reasoning as
// TestSKUCostPriceHistoryRepo_Integration_RLSIsolatesTenants.
func TestSKUPriceHistoryRepo_Integration_RLSIsolatesTenants(t *testing.T) {
	superDB, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	productRepo := repositories.NewProductRepo(superDB)
	variantRepo := repositories.NewVariantRepo(superDB)
	skuRepo := repositories.NewSKURepo(superDB)
	historyRepo := repositories.NewSKUPriceHistoryRepo(db)

	var skuID string
	err := tenant.WithTenant(ctxA, superDB, func(ctx context.Context) error {
		now := time.Now()
		product := &pmsDomain.Product{TenantID: tenant.IDFromContext(ctx), Name: "Acai", Slug: "acai-price-history-rls-test", Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		variant := &pmsDomain.Variant{TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Default", CreatedAt: now, UpdatedAt: now}
		if err := variantRepo.Create(ctx, variant); err != nil {
			return err
		}
		sku := &pmsDomain.SKU{TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "ACAI-RLS-1", PriceMinorUnits: 999, CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := skuRepo.Create(ctx, sku); err != nil {
			return err
		}
		skuID = sku.ID
		return nil
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		return historyRepo.Create(ctx, &pmsDomain.SKUPriceHistory{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID,
			PriceMinorUnits: 999, CurrencyCode: "EUR",
			ValidFrom: time.Now(), Reason: pmsDomain.PriceChangeReasonCreated, CreatedAt: time.Now(),
		})
	})
	require.NoError(t, err)

	_, ctxB := newTestTenant()
	var historyFromOtherTenant []pmsDomain.SKUPriceHistory
	err = tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		var err error
		historyFromOtherTenant, err = historyRepo.ListBySKU(ctx, skuID, 0)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, historyFromOtherTenant, "tenant B must not see tenant A's price history")
}

func TestProductHistoryRepo_Integration_ListByEntityOrdersMostRecentFirst(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	historyRepo := repositories.NewProductHistoryRepo(db)

	var productID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		product := &pmsDomain.Product{TenantID: tenant.IDFromContext(ctx), Name: "Acai", Slug: "acai-history-test", Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		productID = product.ID
		return nil
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := historyRepo.Create(ctx, &pmsDomain.ProductHistory{
			TenantID: tenant.IDFromContext(ctx), EntityType: pmsDomain.ProductHistoryEntityProduct, EntityID: productID,
			Field: pmsDomain.ProductHistoryFieldName, OldValue: "Acai", NewValue: "Acai Berry", ChangedAt: time.Now(),
		}); err != nil {
			return err
		}
		time.Sleep(2 * time.Millisecond)
		return historyRepo.Create(ctx, &pmsDomain.ProductHistory{
			TenantID: tenant.IDFromContext(ctx), EntityType: pmsDomain.ProductHistoryEntityProduct, EntityID: productID,
			Field: pmsDomain.ProductHistoryFieldStatus, OldValue: "draft", NewValue: "active", ChangedAt: time.Now(),
		})
	})
	require.NoError(t, err)

	var history []pmsDomain.ProductHistory
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		history, err = historyRepo.ListByEntity(ctx, pmsDomain.ProductHistoryEntityProduct, productID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, history, 2)
	require.Equal(t, pmsDomain.ProductHistoryFieldStatus, history[0].Field, "most recent change first")
	require.Equal(t, pmsDomain.ProductHistoryFieldName, history[1].Field)
	require.Equal(t, "Acai", history[1].OldValue)
	require.Equal(t, "Acai Berry", history[1].NewValue)
}

func TestProductHistoryRepo_Integration_RLSIsolatesTenants(t *testing.T) {
	superDB, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	productRepo := repositories.NewProductRepo(superDB)
	historyRepo := repositories.NewProductHistoryRepo(db)

	var productID string
	err := tenant.WithTenant(ctxA, superDB, func(ctx context.Context) error {
		now := time.Now()
		product := &pmsDomain.Product{TenantID: tenant.IDFromContext(ctx), Name: "Acai", Slug: "acai-history-rls-test", Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := productRepo.Create(ctx, product); err != nil {
			return err
		}
		productID = product.ID
		return nil
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		return historyRepo.Create(ctx, &pmsDomain.ProductHistory{
			TenantID: tenant.IDFromContext(ctx), EntityType: pmsDomain.ProductHistoryEntityProduct, EntityID: productID,
			Field: pmsDomain.ProductHistoryFieldName, OldValue: "Acai", NewValue: "Acai Berry", ChangedAt: time.Now(),
		})
	})
	require.NoError(t, err)

	_, ctxB := newTestTenant()
	var historyFromOtherTenant []pmsDomain.ProductHistory
	err = tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		var err error
		historyFromOtherTenant, err = historyRepo.ListByEntity(ctx, pmsDomain.ProductHistoryEntityProduct, productID)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, historyFromOtherTenant, "tenant B must not see tenant A's product history")
}

// seedProductForFilterTest creates one active product with the given
// category/tags, returning its id.
func seedProductForFilterTest(t *testing.T, ctx context.Context, repo *repositories.ProductRepo, name string, categoryID *string, tags []string) string {
	t.Helper()
	now := time.Now()
	p := &pmsDomain.Product{
		TenantID: tenant.IDFromContext(ctx), Name: name, Slug: name, CategoryID: categoryID,
		Tags: tags, Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, repo.Create(ctx, p))
	return p.ID
}

// TestProductRepo_Integration_ListByTagUsesJSONBContainment proves the
// tags @> ?::jsonb clause actually matches real Postgres JSONB rows, not
// just parses -- sqlmock can't catch a wrong containment operator/cast.
func TestProductRepo_Integration_ListByTagUsesJSONBContainment(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)

	var summerID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		summerID = seedProductForFilterTest(t, ctx, productRepo, "Sunhat", nil, []string{"summer", "sale"})
		seedProductForFilterTest(t, ctx, productRepo, "Scarf", nil, []string{"winter"})
		return nil
	})
	require.NoError(t, err)

	var results []pmsDomain.Product
	tag := "summer"
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		results, _, err = productRepo.List(ctx, pmsDomain.ProductFilter{Tag: &tag})
		return err
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, summerID, results[0].ID)
}

// TestProductRepo_Integration_ListCombinesCategoryOrIncludeIDsThenExcludes
// proves the OR-group-then-AND-NOT-exclude SQL this filter builds actually
// behaves as documented on ProductFilter -- exactly the "category X's
// products, plus these manually-included ones, minus these excluded ones"
// shape application/storefront's enrollment-aware listing depends on for
// exact pagination.
func TestProductRepo_Integration_ListCombinesCategoryOrIncludeIDsThenExcludes(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	categoryRepo := repositories.NewCategoryRepo(db)

	var categoryID string
	var inCategoryID, manuallyIncludedID, excludedID, unrelatedID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		cat := &pmsDomain.Category{TenantID: tenant.IDFromContext(ctx), Name: "Shoes", Slug: "shoes-filter-test", CreatedAt: now, UpdatedAt: now}
		if err := categoryRepo.Create(ctx, cat); err != nil {
			return err
		}
		categoryID = cat.ID
		inCategoryID = seedProductForFilterTest(t, ctx, productRepo, "In category", &categoryID, nil)
		manuallyIncludedID = seedProductForFilterTest(t, ctx, productRepo, "Manually included", nil, nil)
		excludedID = seedProductForFilterTest(t, ctx, productRepo, "In category but excluded", &categoryID, nil)
		unrelatedID = seedProductForFilterTest(t, ctx, productRepo, "Unrelated", nil, nil)
		return nil
	})
	require.NoError(t, err)
	_ = unrelatedID

	includeIDs := []string{manuallyIncludedID}
	var results []pmsDomain.Product
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		results, _, err = productRepo.List(ctx, pmsDomain.ProductFilter{
			CategoryID: &categoryID, IncludeIDs: &includeIDs, ExcludeIDs: []string{excludedID},
		})
		return err
	})
	require.NoError(t, err)

	gotIDs := make([]string, len(results))
	for i, p := range results {
		gotIDs[i] = p.ID
	}
	require.ElementsMatch(t, []string{inCategoryID, manuallyIncludedID}, gotIDs)
}

// TestProductRepo_Integration_ListManualModeWithNoIncludesMatchesNothing
// proves a non-nil-but-empty IncludeIDs (CategoryID/Tag both nil) matches
// zero rows rather than silently falling through to "no constraint" --
// this is the empty-manual-enrollment case application/storefront's
// listing hits for a channel with EnrollmentModeManual and no products
// added to it yet.
func TestProductRepo_Integration_ListManualModeWithNoIncludesMatchesNothing(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		seedProductForFilterTest(t, ctx, productRepo, "Anything", nil, nil)
		return nil
	})
	require.NoError(t, err)

	emptyIncludes := []string{}
	var results []pmsDomain.Product
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		results, _, err = productRepo.List(ctx, pmsDomain.ProductFilter{IncludeIDs: &emptyIncludes})
		return err
	})
	require.NoError(t, err)
	require.Empty(t, results)
}

// TestProductRepo_Integration_AndIncludeIDsNarrowsRegardlessOfCategory
// proves AndIncludeIDs' independent "AND id IN (...)" behaves like
// AndCategoryID's own test above -- a manually-curated Featured Products
// block (application/storefront.ListProductsInput.ProductIDs) narrows to
// exactly its picked list even when a product sits in an unrelated
// category, and a non-nil-but-empty list matches nothing rather than
// falling through to "unconstrained".
func TestProductRepo_Integration_AndIncludeIDsNarrowsRegardlessOfCategory(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	productRepo := repositories.NewProductRepo(db)
	categoryRepo := repositories.NewCategoryRepo(db)

	var categoryID string
	var pickedID, unpickedInCategoryID, unrelatedID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		cat := &pmsDomain.Category{TenantID: tenant.IDFromContext(ctx), Name: "Beverages", Slug: "beverages-and-include-ids-test", CreatedAt: now, UpdatedAt: now}
		if err := categoryRepo.Create(ctx, cat); err != nil {
			return err
		}
		categoryID = cat.ID
		unpickedInCategoryID = seedProductForFilterTest(t, ctx, productRepo, "In category, not picked", &categoryID, nil)
		pickedID = seedProductForFilterTest(t, ctx, productRepo, "Manually picked, different category", nil, nil)
		unrelatedID = seedProductForFilterTest(t, ctx, productRepo, "Neither", nil, nil)
		return nil
	})
	require.NoError(t, err)
	_ = unpickedInCategoryID
	_ = unrelatedID

	pickedIDs := []string{pickedID}
	var results []pmsDomain.Product
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		results, _, err = productRepo.List(ctx, pmsDomain.ProductFilter{AndIncludeIDs: &pickedIDs})
		return err
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, pickedID, results[0].ID)

	emptyPicked := []string{}
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		results, _, err = productRepo.List(ctx, pmsDomain.ProductFilter{AndIncludeIDs: &emptyPicked})
		return err
	})
	require.NoError(t, err)
	require.Empty(t, results)
}
