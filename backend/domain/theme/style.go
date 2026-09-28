package theme

import (
	"fmt"
	"regexp"
)

// NodeStyle is a component's look, per breakpoint. Base applies
// everywhere; Tablet and Mobile override it at narrower widths. Values
// come from a fixed vocabulary (scale keys, colour tokens or hex, enums)
// -- never raw CSS -- so merchants can't break the layout or inject CSS,
// and a theme-wide change (e.g. the "lg" spacing step) restyles every
// component consistently.
type NodeStyle struct {
	Base   map[string]string
	Tablet map[string]string
	Mobile map[string]string
}

type styleRule func(string) bool

func enum(values ...string) styleRule {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	return func(s string) bool { return set[s] }
}

var (
	hexColor     = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
	assetRef     = regexp.MustCompile(`^asset:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	colorTokens  = enum("primary", "secondary", "background", "surface", "text", "muted", "accent", "transparent")
	spacingScale = enum("none", "xs", "sm", "md", "lg", "xl", "2xl", "3xl")
)

func color(s string) bool { return colorTokens(s) || hexColor.MatchString(s) }

// StyleVocabulary is the complete set of style properties a node may set.
var styleVocabulary = map[string]styleRule{
	"padding_x":           spacingScale,
	"padding_y":           spacingScale,
	"margin_top":          spacingScale,
	"margin_bottom":       spacingScale,
	"gap":                 spacingScale,
	"text_align":          enum("left", "center", "right"),
	"align_items":         enum("start", "center", "end", "stretch"),
	"justify_content":     enum("start", "center", "end", "between"),
	"font_family":         enum("heading", "body"),
	"font_size":           enum("xs", "sm", "base", "lg", "xl", "2xl", "3xl", "4xl", "5xl"),
	"font_weight":         enum("regular", "medium", "semibold", "bold"),
	"line_height":         enum("tight", "normal", "relaxed"),
	"letter_spacing":      enum("tight", "normal", "wide"),
	"text_transform":      enum("none", "uppercase"),
	"color":               color,
	"background_color":    color,
	"border_color":        color,
	"border_width":        enum("none", "thin", "medium", "thick"),
	"border_radius":       enum("none", "sm", "md", "lg", "xl", "full"),
	"shadow":              enum("none", "sm", "md", "lg"),
	"width":               enum("auto", "full", "content", "narrow"),
	"min_height":          enum("none", "sm", "md", "lg", "screen"),
	"aspect_ratio":        enum("auto", "1:1", "4:3", "3:2", "16:9", "3:4", "2:3"),
	"background_image":    assetRef.MatchString,
	"background_size":     enum("cover", "contain"),
	"background_position": enum("center", "top", "bottom"),
	"columns":             enum("1", "2", "3", "4", "5", "6"),
	"display":             enum("show", "hide"),
}

// StyleProperties lists the vocabulary's property names.
func StyleProperties() []string {
	out := make([]string, 0, len(styleVocabulary))
	for k := range styleVocabulary {
		out = append(out, k)
	}
	return out
}

func ValidateStyle(s NodeStyle) error {
	for bp, props := range map[string]map[string]string{"base": s.Base, "tablet": s.Tablet, "mobile": s.Mobile} {
		for prop, value := range props {
			rule, ok := styleVocabulary[prop]
			if !ok {
				return fmt.Errorf("%w: unknown property %q (%s)", ErrInvalidStyle, prop, bp)
			}
			if !rule(value) {
				return fmt.Errorf("%w: %q is not a valid %s (%s)", ErrInvalidStyle, value, prop, bp)
			}
		}
	}
	return nil
}
