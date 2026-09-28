package theme_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/theme"
)

var now = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

func TestNewTheme_trimsAndRequiresName(t *testing.T) {
	th, err := theme.NewTheme("tenant-1", "chan-1", "  Autumn  ", "default", now)
	require.NoError(t, err)
	assert.Equal(t, "Autumn", th.Name)
	assert.Equal(t, "default", th.PresetKey)
	assert.False(t, th.IsLive)
	assert.Nil(t, th.PublishedVersionID)

	_, err = theme.NewTheme("tenant-1", "chan-1", "   ", "default", now)
	assert.ErrorIs(t, err, theme.ErrInvalidThemeName)

	_, err = theme.NewTheme("tenant-1", "chan-1", strings.Repeat("x", 121), "default", now)
	assert.ErrorIs(t, err, theme.ErrInvalidThemeName)
}

func TestNewTheme_requiresKnownPreset(t *testing.T) {
	_, err := theme.NewTheme("tenant-1", "chan-1", "Autumn", "bogus", now)
	assert.ErrorIs(t, err, theme.ErrUnknownPreset)
}

func TestTheme_Rename(t *testing.T) {
	th, _ := theme.NewTheme("tenant-1", "chan-1", "Autumn", "default", now)
	later := now.Add(time.Hour)

	require.NoError(t, th.Rename(" Winter ", later))
	assert.Equal(t, "Winter", th.Name)
	assert.Equal(t, later, th.UpdatedAt)
	assert.ErrorIs(t, th.Rename("", later), theme.ErrInvalidThemeName)
}

func TestTheme_CanDelete_notWhileLive(t *testing.T) {
	th, _ := theme.NewTheme("tenant-1", "chan-1", "Autumn", "default", now)
	assert.NoError(t, th.CanDelete())

	th.IsLive = true
	assert.ErrorIs(t, th.CanDelete(), theme.ErrThemeIsLive)
}
