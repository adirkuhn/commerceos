package channels_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/channels"
)

func TestResolveSignInURL_acceptsHTTPSPagesOnInternalChannels(t *testing.T) {
	got, err := channels.ResolveSignInURL(channels.ChannelTypeInternal, " https://shop.example.com/account/verify?lang=nl ")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "https://shop.example.com/account/verify?lang=nl", *got)
}

func TestResolveSignInURL_emptyClearsIt(t *testing.T) {
	got, err := channels.ResolveSignInURL(channels.ChannelTypeInternal, "  ")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestResolveSignInURL_allowsPlainHTTPOnlyForLocalDevelopment(t *testing.T) {
	for _, ok := range []string{"http://localhost:3000/verify", "http://127.0.0.1/verify"} {
		_, err := channels.ResolveSignInURL(channels.ChannelTypeInternal, ok)
		assert.NoError(t, err, ok)
	}
	_, err := channels.ResolveSignInURL(channels.ChannelTypeInternal, "http://shop.example.com/verify")
	assert.ErrorIs(t, err, channels.ErrInvalidSignInURL)
}

func TestResolveSignInURL_rejectsAnythingThatIsNotAPlainWebPage(t *testing.T) {
	for _, bad := range []string{
		"javascript:alert(1)", "/relative/verify", "//shop.example.com/verify", "https://", "ftp://shop.example.com/x",
		"https://user:pass@shop.example.com/verify", "https://shop.example.com/verify#frag", "not a url",
	} {
		_, err := channels.ResolveSignInURL(channels.ChannelTypeInternal, bad)
		assert.ErrorIs(t, err, channels.ErrInvalidSignInURL, bad)
	}
}

func TestResolveSignInURL_onlyInternalChannelsHaveShopperSignIn(t *testing.T) {
	_, err := channels.ResolveSignInURL(channels.ChannelTypeWooCommerce, "https://shop.example.com/verify")
	assert.ErrorIs(t, err, channels.ErrSignInURLRequiresInternal)
}

func internalChannel(mode channels.StorefrontMode, signIn *string) channels.SalesChannel {
	return channels.SalesChannel{ID: "0b0c7d1e-6f7a-4b8c-9d0e-1f2a3b4c5d6e", Type: channels.ChannelTypeInternal, StorefrontMode: mode, SignInURL: signIn}
}

func TestSignInLink_themedStoreUsesItsOwnLoginPage(t *testing.T) {
	evil := "https://evil.example/steal"
	link, err := internalChannel(channels.StorefrontModeTheme, &evil).SignInLink("https://shops.example.com/", "tok-1")
	require.NoError(t, err)
	assert.Equal(t, "https://shops.example.com/store/0b0c7d1e-6f7a-4b8c-9d0e-1f2a3b4c5d6e/login?token=tok-1", link)
}

func TestSignInLink_headlessStoreUsesTheMerchantsPage(t *testing.T) {
	page := "https://shop.example.com/verify?lang=nl"
	link, err := internalChannel(channels.StorefrontModeHeadless, &page).SignInLink("https://shops.example.com", "tok 1")
	require.NoError(t, err)
	assert.Equal(t, "https://shop.example.com/verify?lang=nl&token=tok+1", link)
}

func TestSignInLink_headlessStoreWithoutAPageCannotSignIn(t *testing.T) {
	_, err := internalChannel(channels.StorefrontModeHeadless, nil).SignInLink("https://shops.example.com", "tok")
	assert.ErrorIs(t, err, channels.ErrSignInNotConfigured)
}
