package theme_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apptheme "github.com/adirkuhn/commerceos/application/theme"
	"github.com/adirkuhn/commerceos/domain/theme"
)

// publish publishes the theme's current draft.
func (w *world) publish(t *testing.T, themeID string) theme.Version {
	t.Helper()
	d := w.draft(t, themeID)
	v, err := apptheme.NewPublishThemeUseCase(w.deps).Execute(tenantCtx(), apptheme.PublishThemeInput{
		ThemeID: themeID, ExpectedRevision: d.Revision, Note: "",
	})
	require.NoError(t, err)
	return *v
}

func TestPublishTheme_sealsDraftGoesLiveAndRaisesEvent(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)

	v, err := apptheme.NewPublishThemeUseCase(w.deps).Execute(tenantCtx(), apptheme.PublishThemeInput{
		ThemeID: th.ID, ExpectedRevision: 1, Note: "launch",
	})

	require.NoError(t, err)
	assert.Equal(t, theme.VersionStateSealed, v.State)
	assert.Equal(t, "launch", v.Note)
	live := w.themes.byID[th.ID]
	assert.True(t, live.IsLive)
	assert.Equal(t, v.ID, *live.PublishedVersionID)
	assert.Contains(t, w.outbox.subjects, "storefront.theme.published")
}

func TestPublishTheme_takesOtherThemeOffLive(t *testing.T) {
	w := newWorld(t)
	first := w.createTheme(t)
	w.publish(t, first.ID)
	second := w.createTheme(t)

	w.publish(t, second.ID)

	assert.False(t, w.themes.byID[first.ID].IsLive)
	assert.True(t, w.themes.byID[second.ID].IsLive)
}

func TestPublishTheme_staleRevisionAndNoDraft(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	uc := apptheme.NewPublishThemeUseCase(w.deps)

	_, err := uc.Execute(tenantCtx(), apptheme.PublishThemeInput{ThemeID: th.ID, ExpectedRevision: 7})
	assert.ErrorIs(t, err, theme.ErrRevisionConflict)
	assert.False(t, w.themes.byID[th.ID].IsLive)
	assert.NotContains(t, w.outbox.subjects, "storefront.theme.published")

	w.publish(t, th.ID)
	_, err = uc.Execute(tenantCtx(), apptheme.PublishThemeInput{ThemeID: th.ID, ExpectedRevision: 1})
	assert.ErrorIs(t, err, theme.ErrNoDraft)
}

func TestPublishTheme_revalidatesEveryDocument(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	d := w.draft(t, th.ID)
	home, _ := w.docs.Find(tenantCtx(), d.ID, theme.KindTemplate, "home")
	home.Root.Children = []theme.Node{{ID: "x", Type: "retired-component"}}
	w.docs.byID[home.ID] = *home

	_, err := apptheme.NewPublishThemeUseCase(w.deps).Execute(tenantCtx(), apptheme.PublishThemeInput{ThemeID: th.ID, ExpectedRevision: 1})

	assert.ErrorIs(t, err, theme.ErrInvalidDocument)
	assert.True(t, w.draft(t, th.ID).IsDraft())
}

func TestPublishTheme_requiresThemedStorefront(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	ch := w.channels.byID["chan-1"]
	ch.StorefrontMode = "headless"
	w.channels.byID["chan-1"] = ch

	_, err := apptheme.NewPublishThemeUseCase(w.deps).Execute(tenantCtx(), apptheme.PublishThemeInput{ThemeID: th.ID, ExpectedRevision: 1})
	assert.ErrorIs(t, err, theme.ErrStorefrontNotEnabled)
}

func TestRollbackTheme(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	v1 := w.publish(t, th.ID)
	_, err := apptheme.NewOpenDraftUseCase(w.deps).Execute(tenantCtx(), th.ID, "user-1")
	require.NoError(t, err)
	v2 := w.publish(t, th.ID)
	require.Equal(t, v2.ID, *w.themes.byID[th.ID].PublishedVersionID)

	got, err := apptheme.NewRollbackThemeUseCase(w.deps).Execute(tenantCtx(), th.ID, v1.ID)

	require.NoError(t, err)
	assert.Equal(t, v1.ID, *got.PublishedVersionID)
	assert.Contains(t, w.outbox.subjects, "storefront.theme.rolled_back")

	_, err = apptheme.NewRollbackThemeUseCase(w.deps).Execute(tenantCtx(), th.ID, "missing")
	assert.ErrorIs(t, err, theme.ErrVersionNotFound)
}

func TestListVersions_newestFirst(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	w.publish(t, th.ID)
	_, _ = apptheme.NewOpenDraftUseCase(w.deps).Execute(tenantCtx(), th.ID, "u")

	list, total, err := apptheme.NewListVersionsUseCase(w.deps).Execute(tenantCtx(), th.ID, 1, 20)

	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, []int{2, 1}, []int{list[0].Number, list[1].Number})
}

// Publishing keeps the newest theme.KeepPublishedVersions sealed versions
// and drops older ones (with their documents) in the same transaction.
func TestPublishTheme_prunesOldPublishedVersions(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	draft := w.draft(t, th.ID)
	for n := 1; n <= theme.KeepPublishedVersions; n++ {
		old := theme.Version{ID: fmt.Sprintf("old-%d", n), ThemeID: th.ID, Number: n, State: theme.VersionStateSealed}
		w.versions.byID[old.ID] = old
	}
	draft.Number = theme.KeepPublishedVersions + 1
	w.versions.byID[draft.ID] = draft

	_, err := apptheme.NewPublishThemeUseCase(w.deps).Execute(tenantCtx(), apptheme.PublishThemeInput{ThemeID: th.ID, ExpectedRevision: draft.Revision})

	require.NoError(t, err)
	assert.Equal(t, []string{th.ID}, w.versions.prunedThemes)
	_, stillThere := w.versions.byID["old-1"]
	assert.False(t, stillThere, "oldest published version is dropped")
	_, kept := w.versions.byID["old-2"]
	assert.True(t, kept)
	assert.Contains(t, w.versions.byID, draft.ID)
}
