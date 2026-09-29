package channels_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/channels"
)

func TestResolveStorefrontLocale(t *testing.T) {
	got, err := channels.ResolveStorefrontLocale("")
	require.NoError(t, err)
	assert.Equal(t, channels.StorefrontLocaleEnglish, got, "blank defaults to English")

	for _, l := range []channels.StorefrontLocale{"en", "pt-BR", "nl"} {
		got, err := channels.ResolveStorefrontLocale(l)
		require.NoError(t, err)
		assert.Equal(t, l, got)
	}

	for _, bad := range []channels.StorefrontLocale{"pt", "EN", "de", "pt_BR"} {
		_, err := channels.ResolveStorefrontLocale(bad)
		assert.ErrorIs(t, err, channels.ErrInvalidStorefrontLocale, string(bad))
	}
}
