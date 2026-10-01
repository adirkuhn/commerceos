package theme_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apptheme "github.com/adirkuhn/commerceos/application/theme"
	"github.com/adirkuhn/commerceos/domain/theme"
)

// fakePresetImages serves a tiny real PNG for every preset picture.
type fakePresetImages struct{ reads int }

func (f *fakePresetImages) Read(_ string, _ theme.PresetImage) ([]byte, error) {
	f.reads++
	var buf bytes.Buffer
	_ = png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	return buf.Bytes(), nil
}

func createFresh(t *testing.T, w *world) (theme.Theme, error) {
	t.Helper()
	th, err := apptheme.NewCreateThemeUseCase(w.deps).Execute(tenantCtx(), apptheme.CreateThemeInput{
		SalesChannelID: "chan-1", Name: "Verde", PresetKey: theme.PresetFresh, UserID: "user-1",
	})
	if err != nil {
		return theme.Theme{}, err
	}
	return *th, nil
}

func assetNamed(t *testing.T, w *world, filename string) theme.Asset {
	t.Helper()
	for _, a := range w.assets.byID {
		if a.Filename == filename {
			return a
		}
	}
	t.Fatalf("no library asset %q", filename)
	return theme.Asset{}
}

func TestCreateTheme_addsThePresetsPicturesToTheLibraryAndUsesThem(t *testing.T) {
	w := newWorld(t)
	w.deps.PresetImages = &fakePresetImages{}
	w.deps.DB = newMockGormDB(t, 40) // one transaction per stored picture

	th, err := createFresh(t, w)
	require.NoError(t, err)

	assert.Len(t, w.storage.saved, len(theme.PresetImages(theme.PresetFresh)))
	hero := assetNamed(t, w, "supermarket-hero.jpg")
	assert.Equal(t, "Smiling shopper with a bag of fresh vegetables", hero.Alt)
	draft := w.draft(t, th.ID)
	home, err := w.docs.Find(context.Background(), draft.ID, theme.KindTemplate, "home")
	require.NoError(t, err)
	assert.Contains(t, fmt.Sprintf("%v", home.Root), "asset:"+hero.ID)
	assert.NotContains(t, fmt.Sprintf("%v", home.Root), "preset-image:")
	require.NotNil(t, draft.Settings.Brand.LogoAssetID)
	assert.Equal(t, assetNamed(t, w, "supermarket-logo.png").ID, *draft.Settings.Brand.LogoAssetID)

	_, err = createFresh(t, w)
	require.NoError(t, err)
	assert.Len(t, w.storage.saved, len(theme.PresetImages(theme.PresetFresh)), "a second theme reuses the pictures already in the library")
}

func TestCreateTheme_withoutPicturesStillCreatesTheTheme(t *testing.T) {
	w := newWorld(t)

	th, err := createFresh(t, w)
	require.NoError(t, err)

	home, err := w.docs.Find(context.Background(), w.draft(t, th.ID).ID, theme.KindTemplate, "home")
	require.NoError(t, err)
	assert.False(t, strings.Contains(fmt.Sprintf("%v", home.Root), "preset-image:"))
}

func TestCreateTheme_failsWhenAPictureCannotBeStored(t *testing.T) {
	w := newWorld(t)
	w.deps.PresetImages = &fakePresetImages{}
	w.storage.err = errors.New("disk full")

	_, err := createFresh(t, w)

	assert.Error(t, err)
	assert.Empty(t, w.themes.byID)
}
