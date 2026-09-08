package theme_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appTheme "github.com/adirkuhn/commerceos/application/theme"
	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
)

func TestGetThemePage_returnsSavedPage(t *testing.T) {
	repo := &fakeThemeRepo{byChannelSlug: map[string]themeDomain.ThemePage{
		themeKey("channel-1", themeDomain.PageSlugHome): {
			SalesChannelID: "channel-1", Slug: themeDomain.PageSlugHome, CanvasWidth: 1200,
			Blocks: []themeDomain.Block{{ID: "b1", Type: themeDomain.BlockTypeHero}},
		},
	}}
	uc := appTheme.NewGetThemePageUseCase(newMockGormDB(t, 1), repo)

	page, err := uc.Execute(context.Background(), "channel-1", themeDomain.PageSlugHome)

	require.NoError(t, err)
	require.Equal(t, 1200, page.CanvasWidth)
	require.Len(t, page.Blocks, 1)
}

func TestGetThemePage_returnsDefaultEmptyPageWhenNeverSaved(t *testing.T) {
	uc := appTheme.NewGetThemePageUseCase(newMockGormDB(t, 1), &fakeThemeRepo{})

	page, err := uc.Execute(context.Background(), "channel-1", themeDomain.PageSlugCheckout)

	require.NoError(t, err)
	require.Equal(t, "channel-1", page.SalesChannelID)
	require.Equal(t, themeDomain.PageSlugCheckout, page.Slug)
	require.Equal(t, 1440, page.CanvasWidth)
	require.Empty(t, page.Blocks)
	require.Empty(t, page.CustomCSS)
}
