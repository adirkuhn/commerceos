//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestTaxClassRepo_Integration_CreateFindListUpdateDelete(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewTaxClassRepo(db)

	var classID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		c := taxDomain.TaxClass{TenantID: tenant.IDFromContext(ctx), Name: "Standard", CreatedAt: now, UpdatedAt: now}
		if err := repo.Create(ctx, &c); err != nil {
			return err
		}
		classID = c.ID
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, classID)

	var found *taxDomain.TaxClass
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, classID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "Standard", found.Name)
	require.False(t, found.IsDefault)

	var listed []taxDomain.TaxClass
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, err = repo.List(ctx)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found.Name = "Standard rate"
		found.IsDefault = true
		return repo.Update(ctx, found)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, classID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "Standard rate", found.Name)
	require.True(t, found.IsDefault)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Delete(ctx, classID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := repo.FindByID(ctx, classID)
		return err
	})
	require.ErrorIs(t, err, taxDomain.ErrTaxClassNotFound)
}

func TestTaxClassRepo_Integration_RLSHidesOtherTenantsClasses(t *testing.T) {
	// Deliberately routed through commerceos_app, not the migration-owning
	// superuser: RLS is never enforced against a superuser (or any
	// BYPASSRLS role) regardless of policy, so this only means something
	// through the same role backend/relay/scheduler actually connect as.
	_, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	tenantA, ctxA := newTestTenant()
	_, ctxB := newTestTenant()
	repo := repositories.NewTaxClassRepo(db)

	err := tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		now := time.Now()
		c := taxDomain.TaxClass{TenantID: tenantA, Name: "Tenant A class", CreatedAt: now, UpdatedAt: now}
		return repo.Create(ctx, &c)
	})
	require.NoError(t, err)

	var listedByB []taxDomain.TaxClass
	err = tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		var err error
		listedByB, err = repo.List(ctx)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, listedByB)
}

func TestTaxRateRepo_Integration_UpsertFindListDelete(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	classRepo := repositories.NewTaxClassRepo(db)
	rateRepo := repositories.NewTaxRateRepo(db)

	var classID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		c := taxDomain.TaxClass{TenantID: tenant.IDFromContext(ctx), Name: "Standard", CreatedAt: now, UpdatedAt: now}
		if err := classRepo.Create(ctx, &c); err != nil {
			return err
		}
		classID = c.ID
		return nil
	})
	require.NoError(t, err)

	var rateID string
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		r := taxDomain.TaxRate{
			TenantID: tenantID, TaxClassID: classID, Country: "NL", RateBasisPoints: 2100,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := rateRepo.Upsert(ctx, &r); err != nil {
			return err
		}
		rateID = r.ID
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, rateID)

	var found *taxDomain.TaxRate
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = rateRepo.FindByID(ctx, rateID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int32(2100), found.RateBasisPoints)

	// Upserting the same (tenant, class, country) again overwrites the rate
	// in place -- same row ID, not a duplicate.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		r := taxDomain.TaxRate{
			TenantID: tenantID, TaxClassID: classID, Country: "NL", RateBasisPoints: 900,
			CreatedAt: now, UpdatedAt: now,
		}
		return rateRepo.Upsert(ctx, &r)
	})
	require.NoError(t, err)

	var listed []taxDomain.TaxRate
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, err = rateRepo.List(ctx)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, rateID, listed[0].ID)
	require.Equal(t, int32(900), listed[0].RateBasisPoints)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return rateRepo.Delete(ctx, rateID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := rateRepo.FindByID(ctx, rateID)
		return err
	})
	require.ErrorIs(t, err, taxDomain.ErrTaxRateNotFound)
}

func TestExternalTaxClassLinkRepo_Integration_UpsertFindListDelete(t *testing.T) {
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	classRepo := repositories.NewTaxClassRepo(db)
	linkRepo := repositories.NewExternalTaxClassLinkRepo(db)

	var classID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		c := taxDomain.TaxClass{TenantID: tenant.IDFromContext(ctx), Name: "Reduced rate", CreatedAt: now, UpdatedAt: now}
		if err := classRepo.Create(ctx, &c); err != nil {
			return err
		}
		classID = c.ID
		return nil
	})
	require.NoError(t, err)

	var linkID string
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		l := taxDomain.ExternalTaxClassLink{
			TenantID: tenantID, Provider: "woocommerce", ExternalSlug: "reduced-rate", TaxClassID: classID,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := linkRepo.Upsert(ctx, &l); err != nil {
			return err
		}
		linkID = l.ID
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, linkID)

	var found *taxDomain.ExternalTaxClassLink
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = linkRepo.FindByProviderAndSlug(ctx, "woocommerce", "reduced-rate")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, classID, found.TaxClassID)

	// Upserting the same (tenant, provider, slug) again repoints the
	// existing row -- same row ID, not a duplicate.
	var otherClassID string
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		c := taxDomain.TaxClass{TenantID: tenant.IDFromContext(ctx), Name: "Zero rate", CreatedAt: now, UpdatedAt: now}
		if err := classRepo.Create(ctx, &c); err != nil {
			return err
		}
		otherClassID = c.ID
		now2 := time.Now()
		l := taxDomain.ExternalTaxClassLink{
			TenantID: tenantID, Provider: "woocommerce", ExternalSlug: "reduced-rate", TaxClassID: otherClassID,
			CreatedAt: now2, UpdatedAt: now2,
		}
		return linkRepo.Upsert(ctx, &l)
	})
	require.NoError(t, err)

	var listed []taxDomain.ExternalTaxClassLink
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, err = linkRepo.List(ctx)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, linkID, listed[0].ID)
	require.Equal(t, otherClassID, listed[0].TaxClassID)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return linkRepo.Delete(ctx, linkID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := linkRepo.FindByProviderAndSlug(ctx, "woocommerce", "reduced-rate")
		return err
	})
	require.ErrorIs(t, err, taxDomain.ErrExternalTaxClassLinkNotFound)
}

func TestExternalTaxClassLinkRepo_Integration_FindByProviderAndSlug_emptySlugIsValid(t *testing.T) {
	// "" represents WooCommerce's own Standard rate -- must be a distinct,
	// findable row, not treated as "no slug"/skipped.
	db := newIntegrationDB(t)
	tenantID, ctx := newTestTenant()
	classRepo := repositories.NewTaxClassRepo(db)
	linkRepo := repositories.NewExternalTaxClassLinkRepo(db)

	var classID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		c := taxDomain.TaxClass{TenantID: tenant.IDFromContext(ctx), Name: "Standard", IsDefault: true, CreatedAt: now, UpdatedAt: now}
		if err := classRepo.Create(ctx, &c); err != nil {
			return err
		}
		classID = c.ID
		return nil
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		l := taxDomain.ExternalTaxClassLink{TenantID: tenantID, Provider: "woocommerce", ExternalSlug: "", TaxClassID: classID, CreatedAt: now, UpdatedAt: now}
		return linkRepo.Upsert(ctx, &l)
	})
	require.NoError(t, err)

	var found *taxDomain.ExternalTaxClassLink
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = linkRepo.FindByProviderAndSlug(ctx, "woocommerce", "")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, classID, found.TaxClassID)
}
