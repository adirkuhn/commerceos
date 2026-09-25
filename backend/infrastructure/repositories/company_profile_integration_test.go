//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestCompanyProfileRepo_GetBeforeSaveIsEmpty(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewCompanyProfileRepo(db)
	_, ctx := newTestTenant()

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		p, err := repo.Get(ctx)
		require.NoError(t, err)
		assert.Equal(t, "", p.TenantID)
		assert.Equal(t, "", p.LegalName)
		return nil
	})
	require.NoError(t, err)
}

func TestCompanyProfileRepo_UpsertThenGet(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewCompanyProfileRepo(db)
	tenantID, ctx := newTestTenant()
	now := time.Now().UTC().Truncate(time.Microsecond)

	first := &settingsDomain.CompanyProfile{
		TenantID: tenantID, LegalName: "First B.V.", VATNumber: "NL123456789B01",
		Address: settingsDomain.CompanyAddress{Street: "Damrak", HouseNumber: "1", PostalCode: "1012 LG", City: "Amsterdam", Country: "NL"},
		IBAN:    "NL91ABNA0417164300", UpdatedAt: now,
	}
	second := *first
	second.LegalName = "Second B.V."
	second.Email = "info@example.nl"

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		require.NoError(t, repo.Upsert(ctx, first))
		require.NoError(t, repo.Upsert(ctx, &second))
		got, err := repo.Get(ctx)
		require.NoError(t, err)
		assert.True(t, second.UpdatedAt.Equal(got.UpdatedAt), "updated_at round-trips")
		got.UpdatedAt = second.UpdatedAt
		assert.Equal(t, second, *got)
		return nil
	})
	require.NoError(t, err)
}

func TestCompanyProfileRepo_TenantsCannotSeeEachOther(t *testing.T) {
	_, dsn := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, dsn)
	repo := repositories.NewCompanyProfileRepo(db)
	tenantA, ctxA := newTestTenant()
	_, ctxB := newTestTenant()

	require.NoError(t, tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		return repo.Upsert(ctx, &settingsDomain.CompanyProfile{TenantID: tenantA, LegalName: "Tenant A B.V.", UpdatedAt: time.Now()})
	}))

	require.NoError(t, tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		got, err := repo.Get(ctx)
		require.NoError(t, err)
		assert.Equal(t, "", got.LegalName, "RLS must hide another tenant's profile")
		return nil
	}))
}

func TestCompanyProfileRepo_RequiresTransaction(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewCompanyProfileRepo(db)
	_, err := repo.Get(context.Background())
	require.Error(t, err)
}
