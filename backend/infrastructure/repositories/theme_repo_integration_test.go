//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type themeRepos struct {
	themes   *repositories.ThemeRepo
	versions *repositories.ThemeVersionRepo
	docs     *repositories.ThemeDocumentRepo
	sessions *repositories.ThemeSessionRepo
}

func newThemeRepos(db *gorm.DB) themeRepos {
	return themeRepos{
		themes: repositories.NewThemeRepo(db), versions: repositories.NewThemeVersionRepo(db),
		docs: repositories.NewThemeDocumentRepo(db), sessions: repositories.NewThemeSessionRepo(db),
	}
}

func createInternalChannel(t *testing.T, ctx context.Context, db *gorm.DB) string {
	t.Helper()
	ch := channelsDomain.SalesChannel{
		Name: "Shop", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		StorefrontMode: channelsDomain.StorefrontModeTheme, Active: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		ch.TenantID = tenant.IDFromContext(ctx)
		return repositories.NewSalesChannelRepo(db).Create(ctx, &ch)
	}))
	return ch.ID
}

// seedTheme creates a theme with a v1 draft holding one home template.
func seedTheme(t *testing.T, ctx context.Context, db *gorm.DB, r themeRepos, channelID string) (theme.Theme, theme.Version, theme.Document) {
	t.Helper()
	var th theme.Theme
	var v theme.Version
	var d theme.Document
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		th, err = theme.NewTheme(tenant.IDFromContext(ctx), channelID, "Autumn", theme.PresetDefault, time.Now())
		if err != nil {
			return err
		}
		if err := r.themes.Create(ctx, &th); err != nil {
			return err
		}
		v = theme.NewDraftVersion(th, 1, theme.DefaultSettings(), "user-1", time.Now())
		if err := r.versions.Create(ctx, &v); err != nil {
			return err
		}
		d = theme.Document{TenantID: th.TenantID, VersionID: v.ID, Kind: theme.KindTemplate, Key: "home", Root: sampleTree()}
		return r.docs.Create(ctx, &d)
	}))
	return th, v, d
}

func sampleTree() theme.Node {
	return theme.Node{ID: "root", Type: theme.RootType, Children: []theme.Node{{
		ID: "s1", Type: "section",
		Style:    theme.NodeStyle{Base: map[string]string{"padding_y": "xl"}, Mobile: map[string]string{"padding_y": "md"}},
		Children: []theme.Node{{ID: "h1", Type: "heading", Props: map[string]any{"text": "Hello", "level": float64(1)}}},
	}}}
}

func TestThemeRepos_Integration_roundTripThemeVersionAndDocument(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	r := newThemeRepos(db)
	th, v, d := seedTheme(t, ctx, db, r, createInternalChannel(t, ctx, db))

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		gotTheme, err := r.themes.FindByID(ctx, th.ID)
		require.NoError(t, err)
		assert.Equal(t, "Autumn", gotTheme.Name)
		assert.False(t, gotTheme.IsLive)

		draft, err := r.versions.FindDraft(ctx, th.ID, true)
		require.NoError(t, err)
		require.NotNil(t, draft)
		assert.Equal(t, v.ID, draft.ID)
		assert.Equal(t, theme.DefaultSettings(), draft.Settings)

		doc, err := r.docs.Find(ctx, v.ID, theme.KindTemplate, "home")
		require.NoError(t, err)
		assert.Equal(t, d.ID, doc.ID)
		assert.Equal(t, sampleTree(), doc.Root)

		list, err := r.docs.List(ctx, v.ID)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Empty(t, list[0].Root.Children, "List leaves trees out")

		n, err := r.versions.NextNumber(ctx, th.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, n)
		return nil
	}))
}

func TestThemeRepos_Integration_publishSealsAndDatabaseKeepsSealedImmutable(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	r := newThemeRepos(db)
	th, v, d := seedTheme(t, ctx, db, r, createInternalChannel(t, ctx, db))

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := theme.Publish(&th, &v, v.Revision, "launch", time.Now()); err != nil {
			return err
		}
		if err := r.versions.Update(ctx, &v); err != nil {
			return err
		}
		return r.themes.Update(ctx, &th)
	}))

	// A sealed version can't be changed even by bypassing the domain.
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		v.Note = "sneaky"
		return r.versions.Update(ctx, &v)
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is published and cannot be changed")

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		d.Title = "sneaky"
		return r.docs.Update(ctx, &d)
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is published and cannot be changed")

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		live, err := r.themes.FindLiveByChannel(ctx, th.SalesChannelID)
		require.NoError(t, err)
		require.NotNil(t, live)
		assert.Equal(t, v.ID, *live.PublishedVersionID)

		draft, err := r.versions.FindDraft(ctx, th.ID, false)
		require.NoError(t, err)
		assert.Nil(t, draft, "publishing consumed the draft")
		return nil
	}))
}

func TestThemeRepos_Integration_copyAllClonesDocumentsIntoNextDraft(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	r := newThemeRepos(db)
	th, v, _ := seedTheme(t, ctx, db, r, createInternalChannel(t, ctx, db))

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		if err := theme.Publish(&th, &v, v.Revision, "", time.Now()); err != nil {
			return err
		}
		if err := r.versions.Update(ctx, &v); err != nil {
			return err
		}
		next := theme.NextDraftFrom(v, 2, "user-2", time.Now())
		if err := r.versions.Create(ctx, &next); err != nil {
			return err
		}
		if err := r.docs.CopyAll(ctx, v.ID, next.ID); err != nil {
			return err
		}
		copied, err := r.docs.Find(ctx, next.ID, theme.KindTemplate, "home")
		require.NoError(t, err)
		assert.Equal(t, sampleTree(), copied.Root)

		copied.Root.Children = nil
		require.NoError(t, r.docs.Update(ctx, copied), "the new draft's copy is editable")
		return nil
	}))
}

func TestThemeRepos_Integration_oneDraftAndOneLiveThemeEnforced(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	r := newThemeRepos(db)
	channelID := createInternalChannel(t, ctx, db)
	th, v, _ := seedTheme(t, ctx, db, r, channelID)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		second := theme.NewDraftVersion(th, 2, theme.DefaultSettings(), "u", time.Now())
		return r.versions.Create(ctx, &second)
	})
	require.Error(t, err, "second draft for one theme")
	assert.Contains(t, err.Error(), "theme_versions_one_draft")

	other, ov, _ := seedTheme(t, ctx, db, r, channelID)
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for _, p := range []struct {
			th *theme.Theme
			v  *theme.Version
		}{{&th, &v}, {&other, &ov}} {
			if err := theme.Publish(p.th, p.v, p.v.Revision, "", time.Now()); err != nil {
				return err
			}
			if err := r.versions.Update(ctx, p.v); err != nil {
				return err
			}
			if err := r.themes.ClearLive(ctx, channelID, p.th.ID); err != nil {
				return err
			}
			if err := r.themes.Update(ctx, p.th); err != nil {
				return err
			}
		}
		live, err := r.themes.FindLiveByChannel(ctx, channelID)
		require.NoError(t, err)
		assert.Equal(t, other.ID, live.ID)

		list, total, err := r.themes.ListByChannel(ctx, channelID, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, list, 2)
		return nil
	}))
}

func TestThemeRepos_Integration_sessionsResolveByHashAcrossTenants(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	r := newThemeRepos(db)
	th, _, _ := seedTheme(t, ctx, db, r, createInternalChannel(t, ctx, db))

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		s, err := theme.NewSession(th.TenantID, th.ID, "user-1", "customer_admin", theme.PurposeEdit, locale.Dutch, "raw", time.Now())
		if err != nil {
			return err
		}
		return r.sessions.Create(ctx, &s)
	}))

	require.NoError(t, tenant.WithTenant(tenant.WithSuperAdmin(context.Background()), db, func(ctx context.Context) error {
		s, err := r.sessions.FindByTokenHash(ctx, theme.HashToken("raw"))
		require.NoError(t, err)
		require.NotNil(t, s)
		assert.Equal(t, th.ID, s.ThemeID)
		assert.Equal(t, locale.Dutch, s.Locale)

		none, err := r.sessions.FindByTokenHash(ctx, theme.HashToken("other"))
		require.NoError(t, err)
		assert.Nil(t, none)
		return nil
	}))
}

func TestThemeRepos_Integration_rlsIsolatesTenants(t *testing.T) {
	superDB, dsn := newIntegrationDBAndDSN(t)
	appDB := newAppRoleDB(t, dsn)
	_, ctxA := newTestTenant()
	_, ctxB := newTestTenant()
	th, v, _ := seedTheme(t, ctxA, appDB, newThemeRepos(appDB), createInternalChannel(t, ctxA, superDB))
	r := newThemeRepos(appDB)

	require.NoError(t, tenant.WithTenant(ctxB, appDB, func(ctx context.Context) error {
		_, err := r.themes.FindByID(ctx, th.ID)
		assert.ErrorIs(t, err, theme.ErrThemeNotFound)
		_, err = r.versions.FindByID(ctx, v.ID)
		assert.ErrorIs(t, err, theme.ErrVersionNotFound)
		_, err = r.docs.Find(ctx, v.ID, theme.KindTemplate, "home")
		assert.ErrorIs(t, err, theme.ErrDocumentNotFound)
		return nil
	}))
}

func TestThemeRepos_Integration_pruneSealedKeepsNewestAndProtected(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	r := newThemeRepos(db)
	th, v1, _ := seedTheme(t, ctx, db, r, createInternalChannel(t, ctx, db))

	// Seal v1 (it has a document), then add sealed v2..v4 and a v5 draft.
	var ids []string
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		if err := theme.Publish(&th, &v1, v1.Revision, "", now); err != nil {
			return err
		}
		if err := r.versions.Update(ctx, &v1); err != nil {
			return err
		}
		ids = append(ids, v1.ID)
		for n := 2; n <= 5; n++ {
			v := theme.NewDraftVersion(th, n, theme.DefaultSettings(), "user-1", now)
			if err := r.versions.Create(ctx, &v); err != nil {
				return err
			}
			if n < 5 {
				if err := theme.Publish(&th, &v, v.Revision, "", now); err != nil {
					return err
				}
				if err := r.versions.Update(ctx, &v); err != nil {
					return err
				}
			}
			ids = append(ids, v.ID)
		}
		return nil
	}))

	var pruned int64
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		// Keep the newest 2 sealed (v4, v3) and protect v1.
		pruned, err = r.versions.PruneSealed(ctx, th.ID, 2, ids[0])
		return err
	}))
	assert.Equal(t, int64(1), pruned, "only v2 goes")

	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		list, _, err := r.versions.ListByTheme(ctx, th.ID, 10, 0)
		require.NoError(t, err)
		var numbers []int
		for _, v := range list {
			numbers = append(numbers, v.Number)
		}
		assert.Equal(t, []int{5, 4, 3, 1}, numbers, "draft, newest sealed and the protected one stay")
		docs, err := r.docs.List(ctx, ids[0])
		require.NoError(t, err)
		assert.Len(t, docs, 1, "the protected version keeps its documents")
		return nil
	}))
}
