package theme_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apptheme "github.com/adirkuhn/commerceos/application/theme"
	"github.com/adirkuhn/commerceos/domain/theme"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func upload(w *world, data []byte, name string) (*theme.Asset, error) {
	return apptheme.NewUploadAssetUseCase(w.deps).Execute(tenantCtx(), apptheme.UploadAssetInput{
		UserID: "user-1", Filename: name, Data: bytes.NewReader(data), Alt: "Logo",
	})
}

func TestUploadAsset_detectsTypeAndSizeFromTheBytes(t *testing.T) {
	w := newWorld(t)

	a, err := upload(w, pngBytes(t, 40, 20), "logo.jpg") // wrong extension on purpose

	require.NoError(t, err)
	assert.Equal(t, "image/png", a.ContentType)
	assert.Equal(t, 40, a.Width)
	assert.Equal(t, 20, a.Height)
	assert.Equal(t, "tenant-1", a.TenantID)
	assert.Equal(t, "Logo", a.Alt)
	assert.True(t, strings.HasSuffix(a.URL, "asset.png"), "stored with the detected type's extension: %s", a.URL)
	assert.Equal(t, "logo.jpg", a.Filename)
	assert.Contains(t, w.assets.byID, a.ID)
}

func TestUploadAsset_rejectsNonImagesEvenWithAnImageName(t *testing.T) {
	w := newWorld(t)

	_, err := upload(w, []byte("<svg onload=alert(1)></svg>"), "evil.png")

	assert.ErrorIs(t, err, theme.ErrUnsupportedAssetType)
	assert.Empty(t, w.storage.saved, "nothing stored")
}

func TestUploadAsset_rejectsOversizedFiles(t *testing.T) {
	w := newWorld(t)
	big := append(pngBytes(t, 1, 1), make([]byte, theme.MaxAssetBytes)...)

	_, err := upload(w, big, "big.png")

	assert.ErrorIs(t, err, theme.ErrAssetTooLarge)
	assert.Empty(t, w.storage.saved)
}

func TestUploadAsset_storageFailure(t *testing.T) {
	w := newWorld(t)
	w.storage.err = errors.New("s3 down")

	_, err := upload(w, pngBytes(t, 1, 1), "a.png")

	assert.Error(t, err)
	assert.Empty(t, w.assets.byID)
}

func TestAssetLibrary_listUpdateDelete(t *testing.T) {
	w := newWorld(t)
	a, err := upload(w, pngBytes(t, 1, 1), "a.png")
	require.NoError(t, err)

	updated, err := apptheme.NewUpdateAssetUseCase(w.deps).Execute(tenantCtx(), a.ID, "New alt")
	require.NoError(t, err)
	assert.Equal(t, "New alt", updated.Alt)

	list, total, err := apptheme.NewListAssetsUseCase(w.deps).Execute(tenantCtx(), 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)

	require.NoError(t, apptheme.NewDeleteAssetUseCase(w.deps).Execute(tenantCtx(), a.ID))
	_, total, _ = apptheme.NewListAssetsUseCase(w.deps).Execute(tenantCtx(), 1, 20)
	assert.Equal(t, int64(0), total)
	assert.NotNil(t, w.assets.byID[a.ID].DeletedAt, "soft deleted: published versions may still show it")

	assert.ErrorIs(t, apptheme.NewDeleteAssetUseCase(w.deps).Execute(tenantCtx(), "missing"), theme.ErrAssetNotFound)
}

func TestDocumentReadsResolveReferencedAssets(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	a, err := upload(w, pngBytes(t, 2, 2), "hero.png")
	require.NoError(t, err)
	root := theme.Node{ID: "root", Type: theme.RootType, Children: []theme.Node{{
		ID: "s1", Type: "section", Style: theme.NodeStyle{Base: map[string]string{"background_image": "asset:" + a.ID}},
		Children: []theme.Node{{ID: "i", Type: "image", Props: map[string]any{"image": "asset:" + a.ID}}},
	}}}
	_, err = apptheme.NewSaveDraftDocumentUseCase(w.deps).Execute(tenantCtx(), apptheme.SaveDraftDocumentInput{
		ThemeID: th.ID, ExpectedRevision: 1, Kind: theme.KindTemplate, Key: "home", Root: root,
	})
	require.NoError(t, err)

	draftDoc, err := apptheme.NewGetDraftDocumentUseCase(w.deps).Execute(tenantCtx(), th.ID, theme.KindTemplate, "home")
	require.NoError(t, err)
	assert.Equal(t, a.URL, draftDoc.Assets[a.ID].URL)

	v := w.publish(t, th.ID)
	pub, err := apptheme.NewGetPublishedDocumentUseCase(w.deps).Execute(tenantCtx(), "chan-1", v.ID, theme.KindTemplate, "home")
	require.NoError(t, err)
	assert.Len(t, pub.Assets, 1)
}

func TestThemeReadsResolveSettingsAssets(t *testing.T) {
	w := newWorld(t)
	th := w.createTheme(t)
	a, err := upload(w, pngBytes(t, 2, 2), "logo.png")
	require.NoError(t, err)
	s := theme.DefaultSettings()
	s.Brand.LogoAssetID = &a.ID
	_, err = apptheme.NewSaveDraftSettingsUseCase(w.deps).Execute(tenantCtx(), apptheme.SaveDraftSettingsInput{ThemeID: th.ID, ExpectedRevision: 1, Settings: s})
	require.NoError(t, err)

	view, err := apptheme.NewOpenDraftUseCase(w.deps).Execute(tenantCtx(), th.ID, "u")
	require.NoError(t, err)
	assert.Equal(t, a.URL, view.Assets[a.ID].URL)

	w.publish(t, th.ID)
	live, err := apptheme.NewGetLiveThemeUseCase(w.deps).Execute(tenantCtx(), "chan-1")
	require.NoError(t, err)
	assert.Contains(t, live.Assets, a.ID)
}

func TestResolveAssetURLs(t *testing.T) {
	w := newWorld(t)
	a, err := upload(w, pngBytes(t, 1, 1), "a.png")
	require.NoError(t, err)

	urls, err := apptheme.NewResolveAssetURLsUseCase(w.deps).Execute(tenantCtx(), []string{a.ID, "missing"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{a.ID: a.URL}, urls)

	empty, err := apptheme.NewResolveAssetURLsUseCase(w.deps).Execute(tenantCtx(), nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}
