//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

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
