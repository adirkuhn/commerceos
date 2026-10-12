package settings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/settings"
)

func TestNewExpiryAlert_SortsAndDedupesWindows(t *testing.T) {
	a, err := settings.NewExpiryAlert(true, []int{7, 90, 30, 7})
	require.NoError(t, err)
	assert.True(t, a.Enabled)
	assert.Equal(t, []int{90, 30, 7}, a.Windows)
}

func TestNewExpiryAlert_RejectsUnknownWindow(t *testing.T) {
	_, err := settings.NewExpiryAlert(true, []int{90, 60})
	assert.ErrorIs(t, err, settings.ErrInvalidExpiryAlertWindow)
}

func TestNewExpiryAlert_EnabledNeedsAWindow(t *testing.T) {
	_, err := settings.NewExpiryAlert(true, nil)
	assert.ErrorIs(t, err, settings.ErrExpiryAlertWindowsRequired)
}

func TestNewExpiryAlert_DisabledKeepsChosenWindows(t *testing.T) {
	a, err := settings.NewExpiryAlert(false, []int{15})
	require.NoError(t, err)
	assert.False(t, a.Enabled)
	assert.Equal(t, []int{15}, a.Windows)

	a, err = settings.NewExpiryAlert(false, nil)
	require.NoError(t, err)
	assert.Equal(t, []int{}, a.Windows)
}

func TestExpiryAlert_Bands(t *testing.T) {
	a, err := settings.NewExpiryAlert(true, []int{90, 30, 7})
	require.NoError(t, err)

	assert.Equal(t, []settings.ExpiryBand{
		{WindowDays: 90, MinDaysLeft: 31},
		{WindowDays: 30, MinDaysLeft: 8},
		{WindowDays: 7, MinDaysLeft: 0},
	}, a.Bands())
}

func TestExpiryAlert_BandsEmptyWhenDisabled(t *testing.T) {
	a, err := settings.NewExpiryAlert(false, []int{90})
	require.NoError(t, err)
	assert.Empty(t, a.Bands())
}

func TestExpiryBand_Contains(t *testing.T) {
	b := settings.ExpiryBand{WindowDays: 30, MinDaysLeft: 8}
	assert.False(t, b.Contains(7))
	assert.True(t, b.Contains(8))
	assert.True(t, b.Contains(30))
	assert.False(t, b.Contains(31))
}

func TestExpiryAlert_ReviewBandsUseChosenWindowsEvenWhenDisabled(t *testing.T) {
	a, err := settings.NewExpiryAlert(false, []int{90, 30})
	require.NoError(t, err)

	assert.Equal(t, []settings.ExpiryBand{
		{WindowDays: 90, MinDaysLeft: 31},
		{WindowDays: 30, MinDaysLeft: 0},
	}, a.ReviewBands())
}

func TestExpiryAlert_ReviewBandsDefaultToEveryWindow(t *testing.T) {
	assert.Equal(t, []settings.ExpiryBand{
		{WindowDays: 90, MinDaysLeft: 31},
		{WindowDays: 30, MinDaysLeft: 16},
		{WindowDays: 15, MinDaysLeft: 8},
		{WindowDays: 7, MinDaysLeft: 0},
	}, settings.ExpiryAlert{}.ReviewBands())
}

func TestExpiryAlert_ReviewBand(t *testing.T) {
	a, err := settings.NewExpiryAlert(true, []int{90, 15})
	require.NoError(t, err)

	b, ok := a.ReviewBand(15)
	assert.True(t, ok)
	assert.Equal(t, settings.ExpiryBand{WindowDays: 15, MinDaysLeft: 0}, b)

	_, ok = a.ReviewBand(30)
	assert.False(t, ok)
}
