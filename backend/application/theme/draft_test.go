package theme_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apptheme "github.com/adirkuhn/commerceos/application/theme"
	"github.com/adirkuhn/commerceos/domain/theme"
)

func headingTree(text string) theme.Node {
	return theme.Node{ID: "root", Type: theme.RootType, Children: []theme.Node{{
		ID: "s1", Type: "section", Children: []theme.Node{{ID: "h1", Type: "heading", Props: map[string]any{"text": text}}},
	}}}
}

func TestOpenDraft_returnsExistingDraftWithDocumentIndex(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)

	view, err := apptheme.NewOpenDraftUseCase(w.deps).Execute(tenantCtx(), th.ID, "user-1")

	require.NoError(t, err)
	assert.Equal(t, w.draft(t, th.ID).ID, view.Version.ID)
	assert.NotEmpty(t, view.Documents)
	assert.Len(t, w.versions.byID, 1, "no new version when a draft exists")
}

func TestOpenDraft_startsNextDraftFromPublishedVersion(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	published := w.publish(t, th.ID)

	view, err := apptheme.NewOpenDraftUseCase(w.deps).Execute(tenantCtx(), th.ID, "user-2")

	require.NoError(t, err)
	assert.Equal(t, 2, view.Version.Number)
	assert.True(t, view.Version.IsDraft())
	assert.Equal(t, "user-2", view.Version.CreatedBy)
	assert.Len(t, w.docs.forVersion(view.Version.ID), len(w.docs.forVersion(published.ID)))
}

func TestSaveDraftSettings(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	uc := apptheme.NewSaveDraftSettingsUseCase(w.deps)
	s := theme.DefaultSettings()
	s.Colors.Primary = "#000000"

	v, err := uc.Execute(tenantCtx(), apptheme.SaveDraftSettingsInput{ThemeID: th.ID, ExpectedRevision: 1, Settings: s})
	require.NoError(t, err)
	assert.Equal(t, 2, v.Revision)
	assert.Equal(t, "#000000", w.draft(t, th.ID).Settings.Colors.Primary)

	_, err = uc.Execute(tenantCtx(), apptheme.SaveDraftSettingsInput{ThemeID: th.ID, ExpectedRevision: 1, Settings: s})
	assert.ErrorIs(t, err, theme.ErrRevisionConflict)

	s.Colors.Primary = "black"
	_, err = uc.Execute(tenantCtx(), apptheme.SaveDraftSettingsInput{ThemeID: th.ID, ExpectedRevision: 2, Settings: s})
	assert.ErrorIs(t, err, theme.ErrInvalidSettings)
}

func TestSaveDraftSettings_noDraft(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	w.publish(t, th.ID)

	_, err := apptheme.NewSaveDraftSettingsUseCase(w.deps).Execute(tenantCtx(), apptheme.SaveDraftSettingsInput{
		ThemeID: th.ID, ExpectedRevision: 1, Settings: theme.DefaultSettings(),
	})
	assert.ErrorIs(t, err, theme.ErrNoDraft)
}

func TestSaveDraftDocument_validatesTreeAndBumpsRevision(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	uc := apptheme.NewSaveDraftDocumentUseCase(w.deps)

	res, err := uc.Execute(tenantCtx(), apptheme.SaveDraftDocumentInput{
		ThemeID: th.ID, ExpectedRevision: 1, Kind: theme.KindTemplate, Key: "home", Root: headingTree("Hi"),
	})
	require.NoError(t, err)
	assert.Equal(t, 2, res.Revision)

	got, err := apptheme.NewGetDraftDocumentUseCase(w.deps).Execute(tenantCtx(), th.ID, theme.KindTemplate, "home")
	require.NoError(t, err)
	assert.Equal(t, headingTree("Hi"), got.Root)

	bad := headingTree("Hi")
	bad.Children[0].Type = "marquee"
	_, err = uc.Execute(tenantCtx(), apptheme.SaveDraftDocumentInput{ThemeID: th.ID, ExpectedRevision: 2, Kind: theme.KindTemplate, Key: "home", Root: bad})
	assert.ErrorIs(t, err, theme.ErrInvalidDocument)
	assert.Equal(t, 2, w.draft(t, th.ID).Revision, "a rejected save leaves the revision alone")

	_, err = uc.Execute(tenantCtx(), apptheme.SaveDraftDocumentInput{ThemeID: th.ID, ExpectedRevision: 2, Kind: theme.KindPage, Key: "nope", Root: headingTree("x")})
	assert.ErrorIs(t, err, theme.ErrDocumentNotFound)
}

func TestPages_createSaveAndDelete(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)

	created, err := apptheme.NewCreatePageUseCase(w.deps).Execute(tenantCtx(), apptheme.CreatePageInput{
		ThemeID: th.ID, ExpectedRevision: 1, Key: "about", Title: "About us",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, created.Revision)
	assert.Equal(t, "About us", created.Document.Title)

	_, err = apptheme.NewCreatePageUseCase(w.deps).Execute(tenantCtx(), apptheme.CreatePageInput{
		ThemeID: th.ID, ExpectedRevision: 2, Key: "about", Title: "Again",
	})
	assert.ErrorIs(t, err, theme.ErrDocumentExists)

	saved, err := apptheme.NewSaveDraftDocumentUseCase(w.deps).Execute(tenantCtx(), apptheme.SaveDraftDocumentInput{
		ThemeID: th.ID, ExpectedRevision: 2, Kind: theme.KindPage, Key: "about", Title: " ", Root: headingTree("x"),
	})
	assert.ErrorIs(t, err, theme.ErrInvalidDocumentTitle, "pages keep a title")
	assert.Nil(t, saved)

	rev, err := apptheme.NewDeletePageUseCase(w.deps).Execute(tenantCtx(), th.ID, 2, "about")
	require.NoError(t, err)
	assert.Equal(t, 3, rev)
	_, err = w.docs.Find(tenantCtx(), w.draft(t, th.ID).ID, theme.KindPage, "about")
	assert.ErrorIs(t, err, theme.ErrDocumentNotFound)
}
