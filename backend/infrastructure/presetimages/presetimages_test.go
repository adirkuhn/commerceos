package presetimages_test

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/infrastructure/presetimages"
)

func TestEveryPresetPictureIsShipped(t *testing.T) {
	for _, key := range theme.PresetKeys() {
		for _, img := range theme.PresetImages(key) {
			data, err := presetimages.Files{}.Read(key, img)
			require.NoError(t, err, "%s: %s", key, img.Filename)
			_, _, err = image.DecodeConfig(bytes.NewReader(data))
			assert.NoError(t, err, "%s: %s is a real image", key, img.Filename)
			assert.LessOrEqual(t, len(data), theme.MaxAssetBytes)
		}
	}
}
