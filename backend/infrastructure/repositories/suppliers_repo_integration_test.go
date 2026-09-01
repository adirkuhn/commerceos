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

// TestSupplierRepo_Integration_SearchIsAccentInsensitiveAndTypoTolerant
// proves the same unaccent()/pg_trgm search Products already has
// (127_search_trigram_indexes.sql) actually works against real Postgres
// for suppliers too -- powers global search (Cmd+K).
func TestSupplierRepo_Integration_SearchIsAccentInsensitiveAndTypoTolerant(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSupplierRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		for _, name := range []string{"Açaí Distribuidora", "Feijão & Companhia"} {
			s := &suppliersDomain.Supplier{
				TenantID: tenant.IDFromContext(ctx), Name: name,
				Status: suppliersDomain.SupplierStatusActive, CreatedAt: now, UpdatedAt: now,
			}
			if err := repo.Create(ctx, s); err != nil {
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
		{"acai", "Açaí Distribuidora"},
		{"AÇAÍ", "Açaí Distribuidora"},
		{"Distribudora", "Açaí Distribuidora"}, // typo, missing an 'i'
		{"feijao", "Feijão & Companhia"},
	} {
		var results []suppliersDomain.Supplier
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			var err error
			results, err = repo.Search(ctx, tc.query, 10)
			return err
		})
		require.NoError(t, err)
		require.Len(t, results, 1, "query %q should match exactly one supplier", tc.query)
		require.Equal(t, tc.want, results[0].Name, "query %q", tc.query)
	}
}

func TestSupplierRepo_Integration_SearchEmptyQuery_returnsNil(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSupplierRepo(db)

	var results []suppliersDomain.Supplier
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		results, err = repo.Search(ctx, "", 10)
		return err
	})
	require.NoError(t, err)
	require.Empty(t, results)
}

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

// Reproduces a real report: AI-extracted text included a trailing period
// on the legal-suffix abbreviation ("LDA.") that the manually-typed
// supplier record never had ("LDA") -- must still match.
func TestSupplierRepo_Integration_FindByNameTruncatesTrailingPunctuationDifferences(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSupplierRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		s := suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx), Name: "PEDAÇOS DO BRASIL, LDA",
			Status: suppliersDomain.SupplierStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &s)
	})
	require.NoError(t, err)

	var found *suppliersDomain.Supplier
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByName(ctx, "PEDAÇOS DO BRASIL, LDA.")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "PEDAÇOS DO BRASIL, LDA", found.Name)
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
