package theme_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appTheme "github.com/adirkuhn/commerceos/application/theme"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
)

func TestSaveThemePage_savesAndReturnsTheSavedPage(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": {ID: "channel-1", Type: channelsDomain.ChannelTypeInternal}}}
	repo := &fakeThemeRepo{}
	uc := appTheme.NewSaveThemePageUseCase(newMockGormDB(t, 1), repo, channelRepo)

	page, err := uc.Execute(context.Background(), appTheme.SaveThemePageInput{
		SalesChannelID: "channel-1", Slug: themeDomain.PageSlugHome, CanvasWidth: 1440,
		Blocks: []themeDomain.Block{{ID: "b1", Type: themeDomain.BlockTypeHero}},
	})

	require.NoError(t, err)
	require.Len(t, page.Blocks, 1)
	require.Equal(t, "channel-1", repo.upserted.SalesChannelID)
}

func TestSaveThemePage_savesCustomCSS(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": {ID: "channel-1", Type: channelsDomain.ChannelTypeInternal}}}
	repo := &fakeThemeRepo{}
	uc := appTheme.NewSaveThemePageUseCase(newMockGormDB(t, 1), repo, channelRepo)

	page, err := uc.Execute(context.Background(), appTheme.SaveThemePageInput{
		SalesChannelID: "channel-1", Slug: themeDomain.PageSlugHome, CanvasWidth: 1440,
		CustomCSS: ".hero { color: red; }",
	})

	require.NoError(t, err)
	require.Equal(t, ".hero { color: red; }", page.CustomCSS)
	require.Equal(t, ".hero { color: red; }", repo.upserted.CustomCSS)
}

func TestSaveThemePage_allowsCheckoutFormBlockOnCheckoutPage(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": {ID: "channel-1", Type: channelsDomain.ChannelTypeInternal}}}
	repo := &fakeThemeRepo{}
	uc := appTheme.NewSaveThemePageUseCase(newMockGormDB(t, 1), repo, channelRepo)

	page, err := uc.Execute(context.Background(), appTheme.SaveThemePageInput{
		SalesChannelID: "channel-1", Slug: themeDomain.PageSlugCheckout, CanvasWidth: 1440,
		Blocks: []themeDomain.Block{{ID: "b1", Type: themeDomain.BlockTypeCheckoutForm}},
	})

	require.NoError(t, err)
	require.True(t, page.HasRequiredBlock())
}

func TestSaveThemePage_rejectsInvalidPageSlug(t *testing.T) {
	uc := appTheme.NewSaveThemePageUseCase(newMockGormDB(t, 0), &fakeThemeRepo{}, &fakeChannelRepo{})

	_, err := uc.Execute(context.Background(), appTheme.SaveThemePageInput{
		SalesChannelID: "channel-1", Slug: "bogus-page",
	})

	require.ErrorIs(t, err, themeDomain.ErrInvalidPageSlug)
}

func TestSaveThemePage_rejectsInvalidBlockType(t *testing.T) {
	uc := appTheme.NewSaveThemePageUseCase(newMockGormDB(t, 0), &fakeThemeRepo{}, &fakeChannelRepo{})

	_, err := uc.Execute(context.Background(), appTheme.SaveThemePageInput{
		SalesChannelID: "channel-1", Slug: themeDomain.PageSlugHome,
		Blocks: []themeDomain.Block{{ID: "b1", Type: "not-a-real-block"}},
	})

	require.ErrorIs(t, err, themeDomain.ErrInvalidBlockType)
}

func TestSaveThemePage_defaultsToFreeformEditorMode(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": {ID: "channel-1", Type: channelsDomain.ChannelTypeInternal}}}
	repo := &fakeThemeRepo{}
	uc := appTheme.NewSaveThemePageUseCase(newMockGormDB(t, 1), repo, channelRepo)

	page, err := uc.Execute(context.Background(), appTheme.SaveThemePageInput{
		SalesChannelID: "channel-1", Slug: themeDomain.PageSlugHome, CanvasWidth: 1440,
	})

	require.NoError(t, err)
	require.Equal(t, themeDomain.EditorModeFreeform, page.EditorMode)
}

func TestSaveThemePage_savesPuckModeAndData(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": {ID: "channel-1", Type: channelsDomain.ChannelTypeInternal}}}
	repo := &fakeThemeRepo{}
	uc := appTheme.NewSaveThemePageUseCase(newMockGormDB(t, 1), repo, channelRepo)

	page, err := uc.Execute(context.Background(), appTheme.SaveThemePageInput{
		SalesChannelID: "channel-1", Slug: themeDomain.PageSlugHome, CanvasWidth: 1440,
		EditorMode: themeDomain.EditorModePuck, PuckData: map[string]any{"content": []any{"x"}},
	})

	require.NoError(t, err)
	require.Equal(t, themeDomain.EditorModePuck, page.EditorMode)
	require.Equal(t, map[string]any{"content": []any{"x"}}, page.PuckData)
}

func TestSaveThemePage_rejectsInvalidEditorMode(t *testing.T) {
	uc := appTheme.NewSaveThemePageUseCase(newMockGormDB(t, 0), &fakeThemeRepo{}, &fakeChannelRepo{})

	_, err := uc.Execute(context.Background(), appTheme.SaveThemePageInput{
		SalesChannelID: "channel-1", Slug: themeDomain.PageSlugHome, EditorMode: "bogus-mode",
	})

	require.ErrorIs(t, err, themeDomain.ErrInvalidEditorMode)
}

func TestSaveThemePage_propagatesUnknownChannel(t *testing.T) {
	uc := appTheme.NewSaveThemePageUseCase(newMockGormDB(t, 1), &fakeThemeRepo{}, &fakeChannelRepo{})

	_, err := uc.Execute(context.Background(), appTheme.SaveThemePageInput{
		SalesChannelID: "missing-channel", Slug: themeDomain.PageSlugHome,
	})

	require.ErrorIs(t, err, channelsDomain.ErrSalesChannelNotFound)
}
