package storefront_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
)

type stubGetThemePage struct {
	out *themeDomain.ThemePage
	err error

	gotChannelID string
	gotSlug      themeDomain.PageSlug
}

func (s *stubGetThemePage) Execute(_ context.Context, channelID string, slug themeDomain.PageSlug) (*themeDomain.ThemePage, error) {
	s.gotChannelID, s.gotSlug = channelID, slug
	return s.out, s.err
}

func TestGetStorefrontTheme_returnsThePage(t *testing.T) {
	get := &stubGetThemePage{out: &themeDomain.ThemePage{SalesChannelID: "chan-1", Slug: themeDomain.PageSlugHome, CanvasWidth: 1440}}
	uc := appstorefront.NewGetStorefrontThemeUseCase(get)

	page, err := uc.Execute(context.Background(), "chan-1", "home")

	require.NoError(t, err)
	require.Equal(t, "chan-1", get.gotChannelID)
	require.Equal(t, themeDomain.PageSlugHome, get.gotSlug)
	require.Equal(t, 1440, page.CanvasWidth)
}

func TestGetStorefrontTheme_rejectsInvalidSlugBeforeCallingUseCase(t *testing.T) {
	get := &stubGetThemePage{}
	uc := appstorefront.NewGetStorefrontThemeUseCase(get)

	_, err := uc.Execute(context.Background(), "chan-1", "not-a-real-page")

	require.ErrorIs(t, err, themeDomain.ErrInvalidPageSlug)
	require.Empty(t, get.gotChannelID)
}

func TestGetStorefrontTheme_propagatesUnderlyingError(t *testing.T) {
	get := &stubGetThemePage{err: errors.New("db down")}
	uc := appstorefront.NewGetStorefrontThemeUseCase(get)

	_, err := uc.Execute(context.Background(), "chan-1", "home")

	require.Error(t, err)
}
