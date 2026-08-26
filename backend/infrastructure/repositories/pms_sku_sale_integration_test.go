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

func int64Ptr(v int64) *int64 { return &v }

func TestSKURepo_Integration_ListWithExpiredSale(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctxA := newTestTenant()
	_, ctxB := newTestTenant()

	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)

	makeSKU := func(ctx context.Context, code string, salePrice *int64, saleEndDate *time.Time) string {
		var skuID string
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			now := time.Now()
			product := &pmsDomain.Product{
				TenantID: tenant.IDFromContext(ctx), Name: "Product " + code, Slug: "product-" + code,
				Status: pmsDomain.ProductStatusActive, CreatedAt: now, UpdatedAt: now,
			}
			if err := productRepo.Create(ctx, product); err != nil {
				return err
			}
			variant := &pmsDomain.Variant{
				TenantID: tenant.IDFromContext(ctx), ProductID: product.ID, Name: "Default", CreatedAt: now, UpdatedAt: now,
			}
			if err := variantRepo.Create(ctx, variant); err != nil {
				return err
			}
			sku := &pmsDomain.SKU{
				TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: code,
				PriceMinorUnits: 1000, SalePriceMinorUnits: salePrice, SaleEndDate: saleEndDate,
				CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now,
			}
			if err := skuRepo.Create(ctx, sku); err != nil {
				return err
			}
			skuID = sku.ID
			return nil
		})
		require.NoError(t, err)
		return skuID
	}

	// .UTC() matters: sale_end_date is a DATE column, and pgx's DATE encoder
	// takes Year/Month/Day straight from time.Time's own Location without
	// normalizing first. CURRENT_DATE (used by ListWithExpiredSale) is
	// evaluated in the session's UTC timezone -- a local-zone time.Now()
	// near midnight lands on the wrong calendar day relative to it.
	yesterday := time.Now().UTC().Add(-24 * time.Hour)
	tomorrow := time.Now().UTC().Add(24 * time.Hour)

	expiredA := makeSKU(ctxA, "EXPIRED-A", int64Ptr(800), &yesterday)
	makeSKU(ctxA, "FUTURE-A", int64Ptr(800), &tomorrow) // not expired yet -- excluded
	makeSKU(ctxA, "NO-SALE-A", nil, nil)                // no sale at all -- excluded
	expiredB := makeSKU(ctxB, "EXPIRED-B", int64Ptr(500), &yesterday)

	var found []pmsDomain.SKU
	err := tenant.WithTenant(tenant.WithSuperAdmin(context.Background()), db, func(ctx context.Context) error {
		var err error
		found, err = skuRepo.ListWithExpiredSale(ctx)
		return err
	})
	require.NoError(t, err)

	foundIDs := make(map[string]bool, len(found))
	for _, s := range found {
		foundIDs[s.ID] = true
	}
	require.True(t, foundIDs[expiredA], "expired SKU in tenant A must be found")
	require.True(t, foundIDs[expiredB], "expired SKU in tenant B must be found -- this is a cross-tenant query")
	require.Len(t, found, 2, "future and no-sale SKUs must be excluded")
}
