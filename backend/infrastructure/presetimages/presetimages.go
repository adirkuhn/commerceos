// Package presetimages serves the pictures theme presets ship with,
// embedded in the binary (see theme.PresetImages). The Supermarket
// ("fresh") pictures come from the grocery template it is modelled on,
// used with the merchant's permission.
package presetimages

import (
	"embed"
	"fmt"
	"path"

	"github.com/adirkuhn/commerceos/domain/theme"
)

//go:embed fresh
var files embed.FS

// Files reads preset pictures from the embedded files: <preset>/<filename>.
type Files struct{}

func (Files) Read(presetKey string, img theme.PresetImage) ([]byte, error) {
	b, err := files.ReadFile(path.Join(presetKey, path.Base(img.Filename)))
	if err != nil {
		return nil, fmt.Errorf("presetimages: %w", err)
	}
	return b, nil
}
