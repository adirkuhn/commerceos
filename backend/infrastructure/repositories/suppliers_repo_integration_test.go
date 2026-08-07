//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestSupplierRepo_Integration_CreateFindListUpdateDelete(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSupplierRepo(db)

	var created suppliersDomain.Supplier
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		created = suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx), Name: "Acme Foods", Email: "hi@acme.example",
			Status: suppliersDomain.SupplierStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var found *suppliersDomain.Supplier
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "Acme Foods", found.Name)

	var listed []suppliersDomain.Supplier
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		listed, err = repo.List(ctx)
		return err
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		found.Name = "Acme Foods Ltd"
		found.UpdatedAt = time.Now()
		return repo.Update(ctx, found)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "Acme Foods Ltd", found.Name)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return repo.Delete(ctx, created.ID)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := repo.FindByID(ctx, created.ID)
		return err
	})
	require.ErrorIs(t, err, suppliersDomain.ErrSupplierNotFound)
}

func TestSupplierRepo_Integration_FindByNameIsCaseInsensitiveAndTrimmed(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSupplierRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		s := suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx), Name: "Pedaços do Brasil, Lda.",
			Status: suppliersDomain.SupplierStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &s)
	})
	require.NoError(t, err)

	var found *suppliersDomain.Supplier
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByName(ctx, "  PEDAÇOS DO BRASIL, LDA.  ")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "Pedaços do Brasil, Lda.", found.Name)
}

func TestSupplierRepo_Integration_FindByNameNotFound(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSupplierRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := repo.FindByName(ctx, "Nobody Ever Registered This Supplier")
		return err
	})
	require.ErrorIs(t, err, suppliersDomain.ErrSupplierNotFound)
}

func TestSupplierRepo_Integration_FindByIDNotFound(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSupplierRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := repo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
		return err
	})
	require.ErrorIs(t, err, suppliersDomain.ErrSupplierNotFound)
}

// TestSupplierRepo_Integration_RLSHidesOtherTenantsSuppliers is routed
// through commerceos_app (not the migration-owning superuser) -- RLS is
// never enforced against a superuser regardless of policy, so this only
// means something exercised through the same role production connects as.
func TestSupplierRepo_Integration_RLSHidesOtherTenantsSuppliers(t *testing.T) {
	_, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	repo := repositories.NewSupplierRepo(db)

	err := tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		now := time.Now()
		s := suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx), Name: "Tenant A Supplier",
			Status: suppliersDomain.SupplierStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &s)
	})
	require.NoError(t, err)

	_, ctxB := newTestTenant()
	var listedForB []suppliersDomain.Supplier
	err = tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		var err error
		listedForB, err = repo.List(ctx)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, listedForB, "RLS must prevent tenant B from seeing tenant A's supplier")
}
