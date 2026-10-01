package theme

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// presetImageReader serves the bytes of a picture a preset ships with.
type presetImageReader interface {
	Read(presetKey string, img theme.PresetImage) ([]byte, error)
}

// maxLibraryScan bounds the look-up for pictures already in the library.
const maxLibraryScan = 1000

// addPresetImages makes sure every picture of the preset is in the
// tenant's media library -- reusing one added for an earlier theme (same
// filename) -- and returns name -> asset id. Files are saved outside the
// theme's transaction, like any upload.
func (d Deps) addPresetImages(ctx context.Context, presetKey, userID string) (map[string]string, error) {
	images := theme.PresetImages(presetKey)
	if len(images) == 0 || d.PresetImages == nil {
		return nil, nil
	}
	existing := map[string]string{}
	if err := tenant.WithTenant(ctx, d.DB, func(ctx context.Context) error {
		assets, _, err := d.Assets.ListActive(ctx, maxLibraryScan, 0)
		for _, a := range assets {
			existing[a.Filename] = a.ID
		}
		return err
	}); err != nil {
		return nil, err
	}
	ids := make(map[string]string, len(images))
	for _, img := range images {
		if id, ok := existing[img.Filename]; ok {
			ids[img.Name] = id
			continue
		}
		data, err := d.PresetImages.Read(presetKey, img)
		if err != nil {
			return nil, fmt.Errorf("preset image %s: %w", img.Name, err)
		}
		asset, err := d.storeAsset(ctx, img.Filename, data, img.Alt, userID)
		if err != nil {
			return nil, fmt.Errorf("preset image %s: %w", img.Name, err)
		}
		ids[img.Name] = asset.ID
	}
	return ids, nil
}
