//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestThemeAssetRepo_Integration_libraryHidesDeletedButResolutionKeepsThem(t *testing.T) {
	_, dsn := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, dsn)
	_, ctx := newTestTenant()
	_, other := newTestTenant()
	repo := repositories.NewThemeAssetRepo(db)

	var kept, gone theme.Asset
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		if kept, err = theme.NewAsset(tenant.IDFromContext(ctx), "https://cdn/a.png", "a.png", "image/png", 10, 4, 3, "A", "u", time.Now()); err != nil {
			return err
		}
		if err := repo.Create(ctx, &kept); err != nil {
			return err
		}
		gone, _ = theme.NewAsset(tenant.IDFromContext(ctx), "https://cdn/b.png", "b.png", "image/png", 10, 0, 0, "", "u", time.Now().Add(time.Second))
		if err := repo.Create(ctx, &gone); err != nil {
			return err
		}
		now := time.Now()
		gone.DeletedAt = &now
		return repo.Update(ctx, &gone)
	}))

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		list, total, err := repo.ListActive(ctx, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		require.Len(t, list, 1)
		assert.Equal(t, kept.ID, list[0].ID)
		assert.Equal(t, 4, list[0].Width)

		resolved, err := repo.FindByIDs(ctx, []string{kept.ID, gone.ID})
		require.NoError(t, err)
		assert.Len(t, resolved, 2)
		return nil
	}))

	require.NoError(t, tenant.WithTenant(other, db, func(ctx context.Context) error {
		_, err := repo.FindByID(ctx, kept.ID)
		assert.ErrorIs(t, err, theme.ErrAssetNotFound, "RLS hides another tenant's assets")
		resolved, err := repo.FindByIDs(ctx, []string{kept.ID})
		require.NoError(t, err)
		assert.Empty(t, resolved)
		return nil
	}))
}
