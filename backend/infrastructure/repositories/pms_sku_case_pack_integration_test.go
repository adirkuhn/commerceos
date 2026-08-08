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

// TestSKURepo_Integration_CaseOfSKURoundTrips is the "box of a product"
// catalog relationship (migration 043): a case SKU declares itself a case
// of a unit SKU, both columns round-tripping through Create/FindByID/Update
// against real Postgres, including clearing the relationship back to nil.
func TestSKURepo_Integration_CaseOfSKURoundTrips(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)

	var unitSKUID, caseSKUID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		product := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Rapadurinha de Minas 25g", Slug: "rapadurinha-25g",
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
		unitSKU := &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "RAPA-25G",
			PriceMinorUnits: 100, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := skuRepo.Create(ctx, unitSKU); err != nil {
			return err
		}
		unitSKUID = unitSKU.ID

		caseSKU := &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "RAPA-CASE-36",
			PriceMinorUnits: 3200, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
			CaseOfSKUID: &unitSKUID, UnitsPerCase: int64Ptr(36),
			CreatedAt: now, UpdatedAt: now,
		}
		if err := skuRepo.Create(ctx, caseSKU); err != nil {
			return err
		}
		caseSKUID = caseSKU.ID
		return nil
	})
	require.NoError(t, err)

	var found *pmsDomain.SKU
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = skuRepo.FindByID(ctx, caseSKUID)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, found.CaseOfSKUID)
	require.Equal(t, unitSKUID, *found.CaseOfSKUID)
	require.NotNil(t, found.UnitsPerCase)
	require.Equal(t, int64(36), *found.UnitsPerCase)

	// Clearing the relationship back to nil must persist as nil, not linger.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found.CaseOfSKUID = nil
		found.UnitsPerCase = nil
		found.UpdatedAt = time.Now()
		return skuRepo.Update(ctx, found)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = skuRepo.FindByID(ctx, caseSKUID)
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found.CaseOfSKUID)
	require.Nil(t, found.UnitsPerCase)
}

// TestSKURepo_Integration_DeletingCaseOfSKUTargetClearsTheReference is the
// migration's ON DELETE SET NULL: deleting the unit SKU a case SKU points
// at must clear case_of_sku_id (and leave units_per_case dangling, since
// the DB constraint only nulls the FK column -- CreateSKUUseCase/
// UpdateSKUUseCase require both set together, but that's an app-layer
// invariant, not enforceable by the FK itself).
func TestSKURepo_Integration_DeletingCaseOfSKUTargetClearsTheReference(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()

	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)

	var caseSKUID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		product := &pmsDomain.Product{
			TenantID: tenant.IDFromContext(ctx), Name: "Rapadurinha de Minas 25g", Slug: "rapadurinha-25g-2",
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
		unitSKU := &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "RAPA-25G-2",
			PriceMinorUnits: 100, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := skuRepo.Create(ctx, unitSKU); err != nil {
			return err
		}
		unitSKUID := unitSKU.ID

		caseSKU := &pmsDomain.SKU{
			TenantID: tenant.IDFromContext(ctx), VariantID: variant.ID, SKUCode: "RAPA-CASE-36-2",
			PriceMinorUnits: 3200, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
			CaseOfSKUID: &unitSKUID, UnitsPerCase: int64Ptr(36),
			CreatedAt: now, UpdatedAt: now,
		}
		if err := skuRepo.Create(ctx, caseSKU); err != nil {
			return err
		}
		caseSKUID = caseSKU.ID
		return skuRepo.Delete(ctx, unitSKUID)
	})
	require.NoError(t, err)

	var found *pmsDomain.SKU
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = skuRepo.FindByID(ctx, caseSKUID)
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found.CaseOfSKUID, "ON DELETE SET NULL must clear the reference, not block or cascade-delete the case SKU")
}
