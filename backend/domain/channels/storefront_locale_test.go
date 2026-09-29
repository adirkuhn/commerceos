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

func TestResolveStorefrontLocales_defaultComesFirstAndEachLanguageOnce(t *testing.T) {
	got, err := channels.ResolveStorefrontLocales(channels.StorefrontLocalePortuguese,
		[]channels.StorefrontLocale{channels.StorefrontLocaleDutch, channels.StorefrontLocalePortuguese, channels.StorefrontLocaleDutch})

	require.NoError(t, err)
	require.Equal(t, []channels.StorefrontLocale{channels.StorefrontLocalePortuguese, channels.StorefrontLocaleDutch}, got)
}

func TestResolveStorefrontLocales_noOthersIsJustTheDefault(t *testing.T) {
	got, err := channels.ResolveStorefrontLocales(channels.StorefrontLocaleEnglish, nil)

	require.NoError(t, err)
	require.Equal(t, []channels.StorefrontLocale{channels.StorefrontLocaleEnglish}, got)
}

func TestResolveStorefrontLocales_rejectsUnsupportedLanguage(t *testing.T) {
	_, err := channels.ResolveStorefrontLocales(channels.StorefrontLocaleEnglish, []channels.StorefrontLocale{"de"})

	require.ErrorIs(t, err, channels.ErrInvalidStorefrontLocale)
}

func TestSalesChannel_OfferedStorefrontLocales(t *testing.T) {
	legacy := channels.SalesChannel{StorefrontLocale: channels.StorefrontLocaleDutch}
	require.Equal(t, []channels.StorefrontLocale{channels.StorefrontLocaleDutch}, legacy.OfferedStorefrontLocales(), "a channel saved before languages were a list offers its one language")

	multi := channels.SalesChannel{StorefrontLocale: channels.StorefrontLocaleEnglish,
		StorefrontLocales: []channels.StorefrontLocale{channels.StorefrontLocaleEnglish, channels.StorefrontLocalePortuguese}}
	require.Equal(t, []channels.StorefrontLocale{channels.StorefrontLocaleEnglish, channels.StorefrontLocalePortuguese}, multi.OfferedStorefrontLocales())

	require.Equal(t, []channels.StorefrontLocale{channels.StorefrontLocaleEnglish}, channels.SalesChannel{}.OfferedStorefrontLocales())
}
