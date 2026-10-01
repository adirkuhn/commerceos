package theme

import "strings"

// Preset images: a preset's documents refer to its pictures as
// "preset-image:<name>" (an image prop or a background_image style). When a
// theme is created from the preset, each picture is added to the tenant's
// media library and BindPresetImages points the documents at those assets
// -- from then on they are ordinary images the merchant can replace or
// remove.

const presetImagePrefix = "preset-image:"

// PresetImage is one picture a preset ships with.
type PresetImage struct {
	Name     string
	Filename string
	Alt      string
}

func presetImage(name string) string { return presetImagePrefix + name }

// PresetImages lists the pictures a preset ships with (none for most).
func PresetImages(presetKey string) []PresetImage {
	return append([]PresetImage(nil), presetRegistry[presetKey].images...)
}

// PresetLogo is the name of the preset image used as the brand logo, or "".
func PresetLogo(presetKey string) string { return presetRegistry[presetKey].logo }

// BindPresetImages replaces every "preset-image:<name>" with the library
// asset added for it (ids: name -> asset id) and sets the logo. A picture
// without an asset (it couldn't be added) leaves its block empty.
func BindPresetImages(docs []Document, settings *Settings, logo string, ids map[string]string) {
	ref := func(v string) (string, bool) {
		name, ok := strings.CutPrefix(v, presetImagePrefix)
		if !ok {
			return v, false
		}
		if id := ids[name]; id != "" {
			return "asset:" + id, true
		}
		return "", true
	}
	var bind func(n *Node)
	bind = func(n *Node) {
		for k, v := range n.Props {
			if s, ok := v.(string); ok {
				if r, changed := ref(s); changed {
					n.Props[k] = r
				}
			}
		}
		for _, bp := range []map[string]string{n.Style.Base, n.Style.Tablet, n.Style.Mobile} {
			for k, v := range bp {
				if r, changed := ref(v); changed {
					if r == "" {
						delete(bp, k)
					} else {
						bp[k] = r
					}
				}
			}
		}
		for i := range n.Children {
			bind(&n.Children[i])
		}
	}
	for i := range docs {
		bind(&docs[i].Root)
	}
	if id := ids[logo]; logo != "" && id != "" {
		settings.Brand.LogoAssetID = &id
	}
}
