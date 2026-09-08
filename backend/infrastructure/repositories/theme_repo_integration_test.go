//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestThemePageRepo_Integration_FindBySlugNotFoundReturnsNilNotError proves
// a channel with no saved theme yet gets nil, not an error -- see
// theme.Repository.FindBySlug's doc comment.
func TestThemePageRepo_Integration_FindBySlugNotFoundReturnsNilNotError(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewThemePageRepo(db)

	var found *themeDomain.ThemePage
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindBySlug(ctx, "00000000-0000-0000-0000-000000000000", "home")
		return err
	})
	require.NoError(t, err)
	require.Nil(t, found)
}

// TestThemePageRepo_Integration_UpsertInsertsThenReplacesWholesale proves
// the JSONB blocks column round-trips (nested Layout/Props survive) through
// real Postgres, and that a second Upsert for the same (channel, slug)
// replaces the whole document rather than merging -- sqlmock can't verify
// either.
func TestThemePageRepo_Integration_UpsertInsertsThenReplacesWholesale(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	channelRepo := repositories.NewSalesChannelRepo(db)
	repo := repositories.NewThemePageRepo(db)

	var channelID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		channel := channelsDomain.SalesChannel{
			TenantID: tenant.IDFromContext(ctx), Name: "B2C Store", Type: channelsDomain.ChannelTypeInternal,
			EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := channelRepo.Create(ctx, &channel); err != nil {
			return err
		}
		channelID = channel.ID
		return nil
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		page := themeDomain.ThemePage{
			TenantID: tenant.IDFromContext(ctx), SalesChannelID: channelID, Slug: "home", CanvasWidth: 1440,
			Blocks: []themeDomain.Block{{
				ID: "block-1", Type: themeDomain.BlockTypeHero,
				Layout: themeDomain.BlockLayout{Desktop: themeDomain.Rect{X: 0, Y: 0, Width: 1440, Height: 400}},
				Props:  map[string]any{"heading": "Welcome"},
			}},
			CustomCSS: ".hero { color: red; }",
			CreatedAt: now, UpdatedAt: now,
		}
		return repo.Upsert(ctx, &page)
	})
	require.NoError(t, err)

	var found *themeDomain.ThemePage
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindBySlug(ctx, channelID, "home")
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Len(t, found.Blocks, 1)
	require.Equal(t, themeDomain.BlockTypeHero, found.Blocks[0].Type)
	require.Equal(t, 1440, found.Blocks[0].Layout.Desktop.Width)
	require.Equal(t, "Welcome", found.Blocks[0].Props["heading"])
	require.Equal(t, ".hero { color: red; }", found.CustomCSS)

	// Second Upsert replaces the whole document -- two blocks now, first
	// one gone entirely, not merged.
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		page := themeDomain.ThemePage{
			TenantID: tenant.IDFromContext(ctx), SalesChannelID: channelID, Slug: "home", CanvasWidth: 1200,
			Blocks: []themeDomain.Block{
				{ID: "block-a", Type: themeDomain.BlockTypeRichText, Layout: themeDomain.BlockLayout{Desktop: themeDomain.Rect{X: 0, Y: 0, Width: 600, Height: 100}}, Props: map[string]any{"text": "A"}},
				{ID: "block-b", Type: themeDomain.BlockTypeProductGrid, Layout: themeDomain.BlockLayout{Desktop: themeDomain.Rect{X: 0, Y: 100, Width: 1200, Height: 600}}, Props: map[string]any{}},
			},
			CreatedAt: now, UpdatedAt: now,
		}
		return repo.Upsert(ctx, &page)
	})
	require.NoError(t, err)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindBySlug(ctx, channelID, "home")
		return err
	})
	require.NoError(t, err)
	require.Equal(t, 1200, found.CanvasWidth)
	require.Len(t, found.Blocks, 2)
	require.Equal(t, "block-a", found.Blocks[0].ID)
	require.Equal(t, "block-b", found.Blocks[1].ID)
	require.Empty(t, found.CustomCSS) // whole-document replace: the second Upsert didn't resend it
}

// TestThemePageRepo_Integration_TenantIsolation proves a page created
// under one tenant is invisible to another -- RLS enforcement (must run as
// commerceos_app, see newAppRoleDB's doc comment -- the superuser
// connection newIntegrationDB returns bypasses RLS unconditionally).
func TestThemePageRepo_Integration_TenantIsolation(t *testing.T) {
	superDB, superDSN := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, superDSN)
	_, ctxA := newTestTenant()
	_, ctxB := newTestTenant()

	channelRepo := repositories.NewSalesChannelRepo(superDB)
	var channelID string
	err := tenant.WithTenant(ctxA, superDB, func(ctx context.Context) error {
		now := time.Now()
		channel := channelsDomain.SalesChannel{
			TenantID: tenant.IDFromContext(ctx), Name: "B2C Store", Type: channelsDomain.ChannelTypeInternal,
			EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := channelRepo.Create(ctx, &channel); err != nil {
			return err
		}
		channelID = channel.ID
		return nil
	})
	require.NoError(t, err)

	repo := repositories.NewThemePageRepo(db)
	err = tenant.WithTenant(ctxA, db, func(ctx context.Context) error {
		now := time.Now()
		page := themeDomain.ThemePage{
			TenantID: tenant.IDFromContext(ctx), SalesChannelID: channelID, Slug: "home", CanvasWidth: 1440,
			CreatedAt: now, UpdatedAt: now,
		}
		return repo.Upsert(ctx, &page)
	})
	require.NoError(t, err)

	var foundFromB *themeDomain.ThemePage
	err = tenant.WithTenant(ctxB, db, func(ctx context.Context) error {
		var err error
		foundFromB, err = repo.FindBySlug(ctx, channelID, "home")
		return err
	})
	require.NoError(t, err)
	require.Nil(t, foundFromB, "tenant B must never see tenant A's theme page")
}
