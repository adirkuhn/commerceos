package theme_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apptheme "github.com/adirkuhn/commerceos/application/theme"
	"github.com/adirkuhn/commerceos/domain/theme"
)

func TestGetLiveTheme(t *testing.T) {
	w := newWorld(t)
	uc := apptheme.NewGetLiveThemeUseCase(w.deps)

	_, err := uc.Execute(tenantCtx(), "chan-1")
	assert.ErrorIs(t, err, theme.ErrNoLiveTheme)

	th := w.createTheme(t)
	v := w.publish(t, th.ID)

	live, err := uc.Execute(tenantCtx(), "chan-1")
	require.NoError(t, err)
	assert.Equal(t, v.ID, live.Version.ID)
	assert.NotEmpty(t, live.Documents)
}

func TestGetLiveTheme_headlessChannelHasNoStorefront(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	w.publish(t, th.ID)
	ch := w.channels.byID["chan-1"]
	ch.StorefrontMode = "headless"
	w.channels.byID["chan-1"] = ch

	_, err := apptheme.NewGetLiveThemeUseCase(w.deps).Execute(tenantCtx(), "chan-1")
	assert.ErrorIs(t, err, theme.ErrStorefrontNotEnabled)
}

func TestGetPublishedDocument_onlyServesSealedVersionsOfTheChannel(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	draft := w.draft(t, th.ID)
	uc := apptheme.NewGetPublishedDocumentUseCase(w.deps)

	_, err := uc.Execute(tenantCtx(), "chan-1", draft.ID, theme.KindTemplate, "home")
	assert.ErrorIs(t, err, theme.ErrVersionNotFound, "drafts are never public")

	v := w.publish(t, th.ID)
	doc, err := uc.Execute(tenantCtx(), "chan-1", v.ID, theme.KindTemplate, "home")
	require.NoError(t, err)
	assert.Equal(t, "home", doc.Key)

	w.channels.byID["chan-2"] = themedChannel("chan-2")
	_, err = uc.Execute(tenantCtx(), "chan-2", v.ID, theme.KindTemplate, "home")
	assert.ErrorIs(t, err, theme.ErrVersionNotFound, "another channel's version")
}

func TestPreview_readsDraftWithoutTouchingLive(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	published := w.publish(t, th.ID)
	view, err := apptheme.NewOpenDraftUseCase(w.deps).Execute(tenantCtx(), th.ID, "u")
	require.NoError(t, err)
	_, err = apptheme.NewSaveDraftDocumentUseCase(w.deps).Execute(tenantCtx(), apptheme.SaveDraftDocumentInput{
		ThemeID: th.ID, ExpectedRevision: view.Version.Revision, Kind: theme.KindTemplate, Key: "home", Root: headingTree("Draft"),
	})
	require.NoError(t, err)

	pv, err := apptheme.NewGetPreviewThemeUseCase(w.deps).Execute(tenantCtx(), th.ID)
	require.NoError(t, err)
	assert.Equal(t, view.Version.ID, pv.Version.ID)
	doc, err := apptheme.NewGetPreviewDocumentUseCase(w.deps).Execute(tenantCtx(), th.ID, theme.KindTemplate, "home")
	require.NoError(t, err)
	assert.Equal(t, headingTree("Draft"), doc.Root)

	live, err := apptheme.NewGetPublishedDocumentUseCase(w.deps).Execute(tenantCtx(), "chan-1", published.ID, theme.KindTemplate, "home")
	require.NoError(t, err)
	assert.Empty(t, live.Root.Children, "live storefront unchanged")
}

func TestPreview_fallsBackToPublishedWhenNoDraft(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	v := w.publish(t, th.ID)

	pv, err := apptheme.NewGetPreviewThemeUseCase(w.deps).Execute(tenantCtx(), th.ID)

	require.NoError(t, err)
	assert.Equal(t, v.ID, pv.Version.ID)
}
