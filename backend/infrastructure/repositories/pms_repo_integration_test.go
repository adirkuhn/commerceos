//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
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
// migration 047's tracks_weight/portioned_from_sku_id columns exist,
// round-trip through GORM's Create/Update, and that the self-referential
// FK is ON DELETE SET NULL: deleting the bulk SKU must not delete the
// portioned SKU, just clear its link to the (now-gone) pool.
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
			CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, TracksWeight: true, CreatedAt: now, UpdatedAt: now,
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
			CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
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
