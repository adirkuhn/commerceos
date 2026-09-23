package settings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/settings"
)

func TestParsePickingMode(t *testing.T) {
	mode, err := settings.ParsePickingMode("list")
	require.NoError(t, err)
	assert.Equal(t, settings.PickingModeList, mode)

	mode, err = settings.ParsePickingMode("scan")
	require.NoError(t, err)
	assert.Equal(t, settings.PickingModeScan, mode)

	mode, err = settings.ParsePickingMode("")
	require.NoError(t, err)
	assert.Equal(t, settings.PickingModeScan, mode, "unset means the default, scan")

	_, err = settings.ParsePickingMode("barcode")
	assert.ErrorIs(t, err, settings.ErrInvalidPickingMode)
}
