package channels_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/channels"
)

func TestResolveStorefrontMode_emptyDefaultsToHeadless(t *testing.T) {
	mode, err := channels.ResolveStorefrontMode(channels.ChannelTypeInternal, "")

	require.NoError(t, err)
	assert.Equal(t, channels.StorefrontModeHeadless, mode)
}

func TestResolveStorefrontMode_internalChannelMayUseTheme(t *testing.T) {
	mode, err := channels.ResolveStorefrontMode(channels.ChannelTypeInternal, channels.StorefrontModeTheme)

	require.NoError(t, err)
	assert.Equal(t, channels.StorefrontModeTheme, mode)
}

func TestResolveStorefrontMode_rejectsUnknownMode(t *testing.T) {
	_, err := channels.ResolveStorefrontMode(channels.ChannelTypeInternal, channels.StorefrontMode("spa"))

	assert.ErrorIs(t, err, channels.ErrInvalidStorefrontMode)
}

func TestResolveStorefrontMode_onlyInternalChannelsHostAThemedStorefront(t *testing.T) {
	for _, ct := range []channels.ChannelType{channels.ChannelTypeWooCommerce, channels.ChannelTypeBol, channels.ChannelTypeWhatsApp} {
		_, err := channels.ResolveStorefrontMode(ct, channels.StorefrontModeTheme)
		assert.ErrorIs(t, err, channels.ErrThemedStorefrontRequiresInternal, ct)

		mode, err := channels.ResolveStorefrontMode(ct, channels.StorefrontModeHeadless)
		require.NoError(t, err)
		assert.Equal(t, channels.StorefrontModeHeadless, mode)
	}
}

func TestSalesChannel_HostsThemedStorefront(t *testing.T) {
	assert.True(t, channels.SalesChannel{Type: channels.ChannelTypeInternal, StorefrontMode: channels.StorefrontModeTheme}.HostsThemedStorefront())
	assert.False(t, channels.SalesChannel{Type: channels.ChannelTypeInternal, StorefrontMode: channels.StorefrontModeHeadless}.HostsThemedStorefront())
	assert.False(t, channels.SalesChannel{Type: channels.ChannelTypeWooCommerce, StorefrontMode: channels.StorefrontModeTheme}.HostsThemedStorefront())
}
