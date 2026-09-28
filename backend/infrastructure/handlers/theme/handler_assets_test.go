package theme_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/theme"
)

func multipartUpload(t *testing.T, filename string, data []byte, alt string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, _ = fw.Write(data)
	if alt != "" {
		require.NoError(t, w.WriteField("alt", alt))
	}
	require.NoError(t, w.Close())
	return &buf, w.FormDataContentType()
}

func TestAssets_uploadPassesFileAndSessionUser(t *testing.T) {
	s := &stub{session: editSession()}
	body, ct := multipartUpload(t, "logo.png", []byte("PNGDATA"), "Our logo")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/theme-editor/assets", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer good")
	w := httptest.NewRecorder()

	newRouter(t, s).ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, "logo.png", s.gotUpload.Filename)
	assert.Equal(t, "Our logo", s.gotUpload.Alt)
	assert.Equal(t, "user-1", s.gotUpload.UserID)
	assert.Equal(t, []byte("PNGDATA"), s.gotUploadBytes)
	assert.Equal(t, "https://cdn.test/a.png", decode(t, w)["data"].(map[string]any)["url"])
}

func TestAssets_uploadErrors(t *testing.T) {
	body, ct := multipartUpload(t, "x.svg", []byte("<svg/>"), "")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/theme-editor/assets", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer good")
	w := httptest.NewRecorder()
	newRouter(t, &stub{session: editSession(), err: theme.ErrUnsupportedAssetType}).ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	missing := do(newRouter(t, &stub{session: editSession()}), http.MethodPost, "/api/v1/theme-editor/assets", "", "good")
	assert.Equal(t, http.StatusUnprocessableEntity, missing.Code)

	unauth := do(newRouter(t, &stub{session: editSession()}), http.MethodGet, "/api/v1/theme-editor/assets", "", "")
	assert.Equal(t, http.StatusUnauthorized, unauth.Code)
}

func TestAssets_listUpdateDelete(t *testing.T) {
	r := newRouter(t, &stub{session: editSession()})

	list := do(r, http.MethodGet, "/api/v1/theme-editor/assets?page=1&per_page=10", "", "good")
	require.Equal(t, http.StatusOK, list.Code)
	assert.EqualValues(t, 1, decode(t, list)["total"])

	upd := do(r, http.MethodPatch, "/api/v1/theme-editor/assets/"+assetID, `{"alt":"New"}`, "good")
	require.Equal(t, http.StatusOK, upd.Code)
	assert.Equal(t, "New", decode(t, upd)["data"].(map[string]any)["alt"])

	assert.Equal(t, http.StatusOK, do(r, http.MethodDelete, "/api/v1/theme-editor/assets/"+assetID, "", "good").Code)
	assert.Equal(t, http.StatusNotFound, do(newRouter(t, &stub{session: editSession(), err: theme.ErrAssetNotFound}), http.MethodDelete, "/api/v1/theme-editor/assets/"+assetID, "", "good").Code)
}

func TestDocuments_includeResolvedAssets(t *testing.T) {
	w := do(newRouter(t, &stub{key: channelKey()}), http.MethodGet, "/api/v1/storefront/v1/theme/versions/"+versionID+"/documents/template/home", "", "shop-key")
	require.Equal(t, http.StatusOK, w.Code)
	assets := decode(t, w)["data"].(map[string]any)["assets"].(map[string]any)
	assert.Equal(t, "https://cdn.test/a.png", assets[assetID].(map[string]any)["url"])
}
