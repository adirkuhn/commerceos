package theme_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func editSession() *theme.Session {
	return &theme.Session{ThemeID: themeID, TenantID: "tenant-1", UserID: "user-1", Purpose: theme.PurposeEdit, Locale: locale.Dutch}
}

// ── editor ──

func TestEditor_rejectsMissingOrInvalidSession(t *testing.T) {
	r := newRouter(t, &stub{session: editSession()})

	assert.Equal(t, http.StatusUnauthorized, do(r, http.MethodPost, "/api/v1/theme-editor/draft", "", "").Code)
	assert.Equal(t, http.StatusUnauthorized, do(r, http.MethodPost, "/api/v1/theme-editor/draft", "", "wrong").Code)
}

func TestEditor_openDraftRunsAsTheSessionTenant(t *testing.T) {
	s := &stub{session: editSession()}
	w := do(newRouter(t, s), http.MethodPost, "/api/v1/theme-editor/draft", "", "good")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	data := decode(t, w)["data"].(map[string]any)
	assert.Equal(t, channelID, data["sales_channel_id"])
	assert.EqualValues(t, 3, data["version"].(map[string]any)["revision"])
	assert.Equal(t, "nl", data["locale"], "the editor speaks the language its session was opened in")
	assert.Equal(t, "tenant-1", tenant.IDFromContext(s.gotCtx))
}

func TestEditor_saveDocumentMapsTreeAndRevision(t *testing.T) {
	s := &stub{session: editSession()}
	body := `{"expected_revision":3,"root":{"id":"root","type":"root","children":[{"id":"h","type":"heading","props":{"text":"Hi","level":1},"translations":{"nl":{"text":"Hoi"}},"style":{"mobile":{"font_size":"xl"}}}]}}`

	w := do(newRouter(t, s), http.MethodPut, "/api/v1/theme-editor/draft/documents/template/home", body, "good")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 3, s.gotSave.ExpectedRevision)
	assert.Equal(t, theme.KindTemplate, s.gotSave.Kind)
	h := s.gotSave.Root.Children[0]
	assert.Equal(t, "Hi", h.Props["text"])
	assert.Equal(t, "Hoi", h.Translations[locale.Dutch]["text"])
	assert.Equal(t, "xl", h.Style.Mobile["font_size"])
	assert.EqualValues(t, 4, decode(t, w)["data"].(map[string]any)["revision"])
}

func TestEditor_invalidDocumentReportsNodeID(t *testing.T) {
	s := &stub{session: editSession(), err: &theme.DocumentError{NodeID: "h", Reason: "unknown component"}}
	w := do(newRouter(t, s), http.MethodPut, "/api/v1/theme-editor/draft/documents/template/home", `{"expected_revision":3,"root":{"id":"root","type":"root"}}`, "good")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	e := decode(t, w)["error"].(map[string]any)
	assert.Equal(t, "VALIDATION_ERROR", e["code"])
	assert.Equal(t, "h", e["node_id"])
}

func TestEditor_revisionConflictIs409(t *testing.T) {
	s := &stub{session: editSession(), err: theme.ErrRevisionConflict}
	w := do(newRouter(t, s), http.MethodPut, "/api/v1/theme-editor/draft/settings", `{"expected_revision":1,"settings":{}}`, "good")
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestEditor_deletePageReadsExpectedRevision(t *testing.T) {
	s := &stub{session: editSession()}
	w := do(newRouter(t, s), http.MethodDelete, "/api/v1/theme-editor/draft/pages/about?expected_revision=5", "", "good")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 5, s.gotRevision)
	assert.EqualValues(t, 6, decode(t, w)["data"].(map[string]any)["revision"])
}

func TestEditor_previewTokenCannotEdit(t *testing.T) {
	sess := editSession()
	sess.Purpose = theme.PurposePreview
	w := do(newRouter(t, &stub{session: sess}), http.MethodPost, "/api/v1/theme-editor/draft", "", "good")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ── public ──

func channelKey() apikeysDomain.ResolvedAPIKey {
	ch := channelID
	return apikeysDomain.ResolvedAPIKey{TenantID: "tenant-1", SalesChannelID: &ch}
}

func TestPublic_requiresChannelScopedKey(t *testing.T) {
	r := newRouter(t, &stub{key: apikeysDomain.ResolvedAPIKey{TenantID: "tenant-1"}})
	assert.Equal(t, http.StatusUnauthorized, do(r, http.MethodGet, "/api/v1/storefront/v1/theme", "", "").Code)
	assert.Equal(t, http.StatusUnauthorized, do(r, http.MethodGet, "/api/v1/storefront/v1/theme", "", "shop-key").Code, "tenant-wide key")
}

func TestPublic_liveThemeIsCacheableWithETag(t *testing.T) {
	s := &stub{key: channelKey()}
	r := newRouter(t, s)

	w := do(r, http.MethodGet, "/api/v1/storefront/v1/theme", "", "shop-key")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	data := decode(t, w)["data"].(map[string]any)
	assert.Equal(t, versionID, data["version_id"])
	assert.Equal(t, false, data["preview"])
	etag := w.Header().Get("ETag")
	assert.NotEmpty(t, etag)
	assert.Contains(t, w.Header().Get("Cache-Control"), "max-age=")
	assert.Equal(t, "tenant-1", tenant.IDFromContext(s.gotCtx))

	assert.Equal(t, http.StatusNotModified, do(r, http.MethodGet, "/api/v1/storefront/v1/theme", "", "shop-key", "If-None-Match", etag).Code)
}

func TestPublic_headlessChannelIs404(t *testing.T) {
	w := do(newRouter(t, &stub{key: channelKey(), err: theme.ErrStorefrontNotEnabled}), http.MethodGet, "/api/v1/storefront/v1/theme", "", "shop-key")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPublic_publishedDocumentIsImmutable(t *testing.T) {
	w := do(newRouter(t, &stub{key: channelKey()}), http.MethodGet, "/api/v1/storefront/v1/theme/versions/"+versionID+"/documents/template/home", "", "shop-key")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Header().Get("Cache-Control"), "immutable")
	root := decode(t, w)["data"].(map[string]any)["root"].(map[string]any)
	assert.Equal(t, "root", root["type"])
}

func TestPublic_previewTokenShowsDraftUncached(t *testing.T) {
	sess := editSession()
	sess.Purpose = theme.PurposePreview
	s := &stub{key: channelKey(), session: sess}

	w := do(newRouter(t, s), http.MethodGet, "/api/v1/storefront/v1/theme", "", "shop-key", "X-Theme-Preview", "good")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, true, decode(t, w)["data"].(map[string]any)["preview"])
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	assert.Equal(t, channelID, s.gotPreviewCh)

	w = do(newRouter(t, s), http.MethodGet, "/api/v1/storefront/v1/theme/preview/documents/template/home", "", "shop-key", "X-Theme-Preview", "good")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

func TestPublic_previewTokenOfAnotherTenantIsRejected(t *testing.T) {
	sess := editSession()
	sess.Purpose = theme.PurposePreview
	sess.TenantID = "tenant-2"

	w := do(newRouter(t, &stub{key: channelKey(), session: sess}), http.MethodGet, "/api/v1/storefront/v1/theme", "", "shop-key", "X-Theme-Preview", "good")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
