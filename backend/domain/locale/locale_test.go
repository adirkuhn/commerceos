package locale_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/locale"
)

func TestResolve(t *testing.T) {
	got, err := locale.Resolve("")
	require.NoError(t, err)
	assert.Equal(t, locale.English, got)

	for _, l := range []locale.Locale{"en", "pt-BR", "nl"} {
		got, err := locale.Resolve(l)
		require.NoError(t, err)
		assert.Equal(t, l, got)
	}
	for _, bad := range []locale.Locale{"pt", "EN", "de", "pt_BR"} {
		_, err := locale.Resolve(bad)
		assert.ErrorIs(t, err, locale.ErrUnsupported, string(bad))
	}
}
