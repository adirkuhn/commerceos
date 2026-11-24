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

func TestAppearanceRepo_GetBeforeSaveIsNil(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewAppearanceRepo(db)
	_, ctx := newTestTenant()

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		a, err := repo.Get(ctx)
		require.NoError(t, err)
		assert.Nil(t, a)
		return nil
	})
	require.NoError(t, err)
}

func TestAppearanceRepo_UpsertThenGet(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewAppearanceRepo(db)
	tenantID, ctx := newTestTenant()
	now := time.Now().UTC().Truncate(time.Microsecond)

	first := settingsDomain.DefaultAppearance(tenantID)
	first.UpdatedAt = now
	second := settingsDomain.Appearance{
		TenantID: tenantID, DefaultLook: settingsDomain.LookGraphite,
		Corners: settingsDomain.CornersSharp, TextSize: settingsDomain.TextSizeLarger, Density: settingsDomain.DensityCompact,
		UpdatedAt: now.Add(time.Minute),
	}

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		require.NoError(t, repo.Upsert(ctx, &first))
		require.NoError(t, repo.Upsert(ctx, &second))
		got, err := repo.Get(ctx)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.True(t, second.UpdatedAt.Equal(got.UpdatedAt), "updated_at round-trips")
		got.UpdatedAt = second.UpdatedAt
		assert.Equal(t, second, *got)
		return nil
	})
	require.NoError(t, err)
}

func TestAppearanceRepo_TenantsCannotSeeEachOther(t *testing.T) {
	_, dsn := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, dsn)
	repo := repositories.NewAppearanceRepo(db)
	tenantA, ctxA := newTestTenant()
	_, ctxB := newTestTenant()

	require.NoError(t, tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		a := settingsDomain.DefaultAppearance(tenantA)
		a.UpdatedAt = time.Now()
		return repo.Upsert(ctx, &a)
	}))

	require.NoError(t, tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		got, err := repo.Get(ctx)
		require.NoError(t, err)
		assert.Nil(t, got, "RLS must hide another tenant's appearance")
		return nil
	}))
}

func TestAppearanceRepo_RequiresTransaction(t *testing.T) {
	db := newIntegrationDB(t)
	repo := repositories.NewAppearanceRepo(db)

	_, err := repo.Get(context.Background())
	require.Error(t, err)
	a := settingsDomain.DefaultAppearance("00000000-0000-0000-0000-000000000001")
	require.Error(t, repo.Upsert(context.Background(), &a))
}
