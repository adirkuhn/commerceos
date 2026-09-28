package theme

import (
	"fmt"
	"regexp"
)

type FontSource string

const (
	// FontSourceSystem uses a curated font stack already on the shopper's
	// device -- no third-party request.
	FontSourceSystem FontSource = "system"
	// FontSourceGoogle loads the named families from Google Fonts (a
	// third-party request from the shopper's browser; merchant opt-in).
	FontSourceGoogle FontSource = "google"
)

var (
	systemFonts      = enum("system-sans", "system-serif", "system-rounded", "system-mono")
	googleFontFamily = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ]{0,63}$`)
	uuidPattern      = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// Settings are a theme version's global design choices; components read
// them (as CSS custom properties) instead of hard-coding colours or fonts.
type Settings struct {
	Brand      BrandSettings
	Colors     ColorSettings
	Typography TypographySettings
	Layout     LayoutSettings
	Buttons    ButtonSettings
	Product    ProductSettings
	Header     HeaderSettings
	Footer     FooterSettings
}

type BrandSettings struct {
	LogoAssetID    *string
	FaviconAssetID *string
}

type ColorSettings struct {
	Primary, Secondary, Background, Surface, Text, Muted, Accent string
}

type TypographySettings struct {
	Source      FontSource
	HeadingFont string
	BodyFont    string
	Scale       string
}

type LayoutSettings struct {
	MaxWidth int
	Gutter   string
}

type ButtonSettings struct {
	Radius, Size, Style string
}

type ProductSettings struct {
	GridColumns int
	ImageRatio  string
}

type HeaderSettings struct {
	Layout string
	Sticky bool
}

type FooterSettings struct {
	Layout string
}

func DefaultSettings() Settings {
	return Settings{
		Colors: ColorSettings{
			Primary: "#1f3fbf", Secondary: "#0f172a", Background: "#ffffff", Surface: "#f6f7f9",
			Text: "#111827", Muted: "#6b7280", Accent: "#e4572e",
		},
		Typography: TypographySettings{Source: FontSourceSystem, HeadingFont: "system-sans", BodyFont: "system-sans", Scale: "md"},
		Layout:     LayoutSettings{MaxWidth: 1280, Gutter: "md"},
		Buttons:    ButtonSettings{Radius: "md", Size: "md", Style: "solid"},
		Product:    ProductSettings{GridColumns: 4, ImageRatio: "1:1"},
		Header:     HeaderSettings{Layout: "logo-left", Sticky: true},
		Footer:     FooterSettings{Layout: "columns"},
	}
}

func (s Settings) Validate() error {
	checks := []struct {
		field string
		ok    bool
	}{
		{"brand.logo_asset_id", optionalUUID(s.Brand.LogoAssetID)},
		{"brand.favicon_asset_id", optionalUUID(s.Brand.FaviconAssetID)},
		{"colors.primary", hexColor.MatchString(s.Colors.Primary)},
		{"colors.secondary", hexColor.MatchString(s.Colors.Secondary)},
		{"colors.background", hexColor.MatchString(s.Colors.Background)},
		{"colors.surface", hexColor.MatchString(s.Colors.Surface)},
		{"colors.text", hexColor.MatchString(s.Colors.Text)},
		{"colors.muted", hexColor.MatchString(s.Colors.Muted)},
		{"colors.accent", hexColor.MatchString(s.Colors.Accent)},
		{"typography.source", s.Typography.Source == FontSourceSystem || s.Typography.Source == FontSourceGoogle},
		{"typography.heading_font", validFont(s.Typography.Source, s.Typography.HeadingFont)},
		{"typography.body_font", validFont(s.Typography.Source, s.Typography.BodyFont)},
		{"typography.scale", enum("sm", "md", "lg")(s.Typography.Scale)},
		{"layout.max_width", s.Layout.MaxWidth >= 960 && s.Layout.MaxWidth <= 1920},
		{"layout.gutter", spacingScale(s.Layout.Gutter)},
		{"buttons.radius", enum("none", "sm", "md", "lg", "full")(s.Buttons.Radius)},
		{"buttons.size", enum("sm", "md", "lg")(s.Buttons.Size)},
		{"buttons.style", enum("solid", "outline")(s.Buttons.Style)},
		{"product.grid_columns", s.Product.GridColumns >= 2 && s.Product.GridColumns <= 6},
		{"product.image_ratio", enum("1:1", "4:3", "3:4", "2:3", "16:9")(s.Product.ImageRatio)},
		{"header.layout", enum("logo-left", "logo-center")(s.Header.Layout)},
		{"footer.layout", enum("columns", "simple")(s.Footer.Layout)},
	}
	for _, c := range checks {
		if !c.ok {
			return fmt.Errorf("%w: %s", ErrInvalidSettings, c.field)
		}
	}
	return nil
}

func validFont(source FontSource, font string) bool {
	if source == FontSourceGoogle {
		return googleFontFamily.MatchString(font) || systemFonts(font)
	}
	return systemFonts(font)
}

func optionalUUID(id *string) bool { return id == nil || uuidPattern.MatchString(*id) }
