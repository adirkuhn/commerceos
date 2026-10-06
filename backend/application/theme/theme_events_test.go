package theme_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	apptheme "github.com/adirkuhn/commerceos/application/theme"
	"github.com/adirkuhn/commerceos/domain/theme"
)

// recording swaps the world's outbox for one that keeps payloads, after
// setup (creating/publishing a theme) has raised its own events.
func recording(w *world) *eventrecordtest.Outbox {
	o := &eventrecordtest.Outbox{}
	w.deps.Outbox = o
	return o
}

func onlyEvent(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

func TestRenameTheme_RecordsRenamed(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	o := recording(w)

	_, err := apptheme.NewRenameThemeUseCase(w.deps).Execute(tenantCtx(), th.ID, "Winter")

	require.NoError(t, err)
	e := onlyEvent(t, o)
	assert.Equal(t, "storefront.theme.renamed", e.Subject)
	assert.Equal(t, th.ID, e.Payload["entity_id"])
	assert.Equal(t, map[string][2]any{"name": {"Autumn", "Winter"}}, e.Changes())
}

func TestOpenDraft_RecordsDraftStartedOnlyWhenANewDraftIsMade(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	published := w.publish(t, th.ID)
	o := recording(w)

	view, err := apptheme.NewOpenDraftUseCase(w.deps).Execute(tenantCtx(), th.ID, "user-2")
	require.NoError(t, err)
	e := onlyEvent(t, o)
	assert.Equal(t, "storefront.theme.draft_started", e.Subject)
	assert.Equal(t, th.ID, e.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"version_id": view.Version.ID, "version_number": float64(2), "based_on_version_id": published.ID,
	}, e.Data())

	_, err = apptheme.NewOpenDraftUseCase(w.deps).Execute(tenantCtx(), th.ID, "user-2")
	require.NoError(t, err)
	assert.Len(t, o.Events, 1, "reopening the existing draft changes nothing")
}

func TestSaveDraftSettings_RecordsSettingsUpdated(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	o := recording(w)

	v, err := apptheme.NewSaveDraftSettingsUseCase(w.deps).Execute(tenantCtx(), apptheme.SaveDraftSettingsInput{
		ThemeID: th.ID, ExpectedRevision: 1, Settings: theme.DefaultSettings(),
	})

	require.NoError(t, err)
	e := onlyEvent(t, o)
	assert.Equal(t, "storefront.theme.settings_updated", e.Subject)
	assert.Equal(t, th.ID, e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"version_id": v.ID, "revision": float64(2)}, e.Data())
}

func TestSaveDraftDocument_RecordsDocumentUpdated(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	o := recording(w)

	res, err := apptheme.NewSaveDraftDocumentUseCase(w.deps).Execute(tenantCtx(), apptheme.SaveDraftDocumentInput{
		ThemeID: th.ID, ExpectedRevision: 1, Kind: theme.KindTemplate, Key: "home", Root: headingTree("Hi"),
	})

	require.NoError(t, err)
	e := onlyEvent(t, o)
	assert.Equal(t, "storefront.theme_document.updated", e.Subject)
	assert.Equal(t, res.Document.ID, e.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"theme_id": th.ID, "version_id": res.Document.VersionID, "kind": "template", "key": "home", "revision": float64(2),
	}, e.Data())
}

func TestPages_RecordCreatedAndDeleted(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	o := recording(w)

	created, err := apptheme.NewCreatePageUseCase(w.deps).Execute(tenantCtx(), apptheme.CreatePageInput{
		ThemeID: th.ID, ExpectedRevision: 1, Key: "about", Title: "About us",
	})
	require.NoError(t, err)
	_, err = apptheme.NewDeletePageUseCase(w.deps).Execute(tenantCtx(), th.ID, 2, "about")
	require.NoError(t, err)

	require.Equal(t, []string{"storefront.theme_page.created", "storefront.theme_page.deleted"}, o.Subjects())
	assert.Equal(t, created.Document.ID, o.Events[0].Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"theme_id": th.ID, "version_id": created.Document.VersionID, "key": "about", "title": "About us", "revision": float64(2),
	}, o.Events[0].Data())
	assert.Equal(t, created.Document.ID, o.Events[1].Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"theme_id": th.ID, "version_id": created.Document.VersionID, "key": "about", "revision": float64(3),
	}, o.Events[1].Data())
}

func TestCreateSession_RecordsCreatedWithoutTheToken(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	o := recording(w)

	res, err := apptheme.NewCreateSessionUseCase(w.deps).Execute(tenantCtx(), apptheme.CreateSessionInput{
		ThemeID: th.ID, UserID: "user-1", Role: "customer_admin", Purpose: theme.PurposeEdit,
	})

	require.NoError(t, err)
	e := onlyEvent(t, o)
	assert.Equal(t, "storefront.theme_session.created", e.Subject)
	assert.Equal(t, map[string]any{
		"theme_id": th.ID, "purpose": string(theme.PurposeEdit), "user_id": "user-1", "role": "customer_admin", "locale": "en",
	}, e.Data())
	raw, err := json.Marshal(e.Payload)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), res.Token)
	assert.NotContains(t, string(raw), theme.HashToken(res.Token))
	stored := w.sessions.byHash[theme.HashToken(res.Token)]
	assert.Equal(t, stored.ID, e.Payload["entity_id"])
}

func TestAssets_RecordUploadedUpdatedDeleted(t *testing.T) {
	w := newWorld(t)
	o := recording(w)

	a, err := upload(w, pngBytes(t, 4, 3), "logo.png")
	require.NoError(t, err)
	_, err = apptheme.NewUpdateAssetUseCase(w.deps).Execute(tenantCtx(), a.ID, "New alt")
	require.NoError(t, err)
	require.NoError(t, apptheme.NewDeleteAssetUseCase(w.deps).Execute(tenantCtx(), a.ID))

	require.Equal(t, []string{"storefront.theme_asset.uploaded", "storefront.theme_asset.updated", "storefront.theme_asset.deleted"}, o.Subjects())
	assert.Equal(t, a.ID, o.Events[0].Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"filename": "logo.png", "content_type": "image/png", "bytes": float64(a.Bytes), "width": float64(4), "height": float64(3),
	}, o.Events[0].Data())
	assert.Equal(t, map[string][2]any{"alt": {"Logo", "New alt"}}, o.Events[1].Changes())
	assert.Equal(t, map[string]any{"filename": "logo.png"}, o.Events[2].Data())
}

func TestThemeUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	w := newWorld(t)
	th := w.createTheme(t)
	w.deps.Outbox = &eventrecordtest.Outbox{Err: boom}

	_, err := apptheme.NewRenameThemeUseCase(w.deps).Execute(tenantCtx(), th.ID, "X")
	assert.ErrorIs(t, err, boom, "rename")
	_, err = apptheme.NewSaveDraftSettingsUseCase(w.deps).Execute(tenantCtx(), apptheme.SaveDraftSettingsInput{
		ThemeID: th.ID, ExpectedRevision: 1, Settings: theme.DefaultSettings(),
	})
	assert.ErrorIs(t, err, boom, "settings")
	_, err = apptheme.NewUploadAssetUseCase(w.deps).Execute(tenantCtx(), apptheme.UploadAssetInput{
		UserID: "u", Filename: "a.png", Data: bytes.NewReader(pngBytes(t, 1, 1)),
	})
	assert.ErrorIs(t, err, boom, "upload")
}
