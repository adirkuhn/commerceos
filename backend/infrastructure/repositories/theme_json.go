package repositories

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/theme"
)

// jsonb stores any JSON-serialisable value in a JSONB column.
type jsonb[T any] struct{ V T }

func (j jsonb[T]) Value() (driver.Value, error) {
	b, err := json.Marshal(j.V)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (j *jsonb[T]) Scan(value any) error {
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	case nil:
		return nil
	default:
		return fmt.Errorf("jsonb.Scan: unsupported type %T", value)
	}
	return json.Unmarshal(raw, &j.V)
}

// The structs below are the persisted JSON shapes. They're kept apart from
// the domain types so the domain carries no storage tags, and so the stored
// format only changes on purpose.

type themeNodeJSON struct {
	ID           string                           `json:"id"`
	Type         string                           `json:"type"`
	Props        map[string]any                   `json:"props,omitempty"`
	Translations map[locale.Locale]map[string]any `json:"translations,omitempty"`
	Style        themeStyleJSON                   `json:"style,omitempty"`
	Children     []themeNodeJSON                  `json:"children,omitempty"`
}

type themeStyleJSON struct {
	Base   map[string]string `json:"base,omitempty"`
	Tablet map[string]string `json:"tablet,omitempty"`
	Mobile map[string]string `json:"mobile,omitempty"`
}

func toNodeJSON(n theme.Node) themeNodeJSON {
	out := themeNodeJSON{
		ID: n.ID, Type: n.Type, Props: n.Props, Translations: n.Translations,
		Style: themeStyleJSON{Base: n.Style.Base, Tablet: n.Style.Tablet, Mobile: n.Style.Mobile},
	}
	for _, c := range n.Children {
		out.Children = append(out.Children, toNodeJSON(c))
	}
	return out
}

func (n themeNodeJSON) toDomain() theme.Node {
	out := theme.Node{
		ID: n.ID, Type: n.Type, Props: n.Props, Translations: n.Translations,
		Style: theme.NodeStyle{Base: n.Style.Base, Tablet: n.Style.Tablet, Mobile: n.Style.Mobile},
	}
	for _, c := range n.Children {
		out.Children = append(out.Children, c.toDomain())
	}
	return out
}

type themeSEOJSON struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

type themeSettingsJSON struct {
	Brand struct {
		LogoAssetID    *string `json:"logo_asset_id,omitempty"`
		FaviconAssetID *string `json:"favicon_asset_id,omitempty"`
	} `json:"brand"`
	Colors struct {
		Primary    string `json:"primary"`
		Secondary  string `json:"secondary"`
		Background string `json:"background"`
		Surface    string `json:"surface"`
		Text       string `json:"text"`
		Muted      string `json:"muted"`
		Accent     string `json:"accent"`
	} `json:"colors"`
	Typography struct {
		Source      string `json:"source"`
		HeadingFont string `json:"heading_font"`
		BodyFont    string `json:"body_font"`
		Scale       string `json:"scale"`
	} `json:"typography"`
	Layout struct {
		MaxWidth int    `json:"max_width"`
		Gutter   string `json:"gutter"`
	} `json:"layout"`
	Buttons struct {
		Radius string `json:"radius"`
		Size   string `json:"size"`
		Style  string `json:"style"`
	} `json:"buttons"`
	Product struct {
		GridColumns int    `json:"grid_columns"`
		ImageRatio  string `json:"image_ratio"`
	} `json:"product"`
	Header struct {
		Layout string `json:"layout"`
		Sticky bool   `json:"sticky"`
	} `json:"header"`
	Footer struct {
		Layout string `json:"layout"`
	} `json:"footer"`
}

func toSettingsJSON(s theme.Settings) themeSettingsJSON {
	var j themeSettingsJSON
	j.Brand.LogoAssetID, j.Brand.FaviconAssetID = s.Brand.LogoAssetID, s.Brand.FaviconAssetID
	c := s.Colors
	j.Colors.Primary, j.Colors.Secondary, j.Colors.Background, j.Colors.Surface = c.Primary, c.Secondary, c.Background, c.Surface
	j.Colors.Text, j.Colors.Muted, j.Colors.Accent = c.Text, c.Muted, c.Accent
	ty := s.Typography
	j.Typography.Source, j.Typography.HeadingFont, j.Typography.BodyFont, j.Typography.Scale = string(ty.Source), ty.HeadingFont, ty.BodyFont, ty.Scale
	j.Layout.MaxWidth, j.Layout.Gutter = s.Layout.MaxWidth, s.Layout.Gutter
	j.Buttons.Radius, j.Buttons.Size, j.Buttons.Style = s.Buttons.Radius, s.Buttons.Size, s.Buttons.Style
	j.Product.GridColumns, j.Product.ImageRatio = s.Product.GridColumns, s.Product.ImageRatio
	j.Header.Layout, j.Header.Sticky = s.Header.Layout, s.Header.Sticky
	j.Footer.Layout = s.Footer.Layout
	return j
}

func (j themeSettingsJSON) toDomain() theme.Settings {
	return theme.Settings{
		Brand: theme.BrandSettings{LogoAssetID: j.Brand.LogoAssetID, FaviconAssetID: j.Brand.FaviconAssetID},
		Colors: theme.ColorSettings{
			Primary: j.Colors.Primary, Secondary: j.Colors.Secondary, Background: j.Colors.Background,
			Surface: j.Colors.Surface, Text: j.Colors.Text, Muted: j.Colors.Muted, Accent: j.Colors.Accent,
		},
		Typography: theme.TypographySettings{
			Source: theme.FontSource(j.Typography.Source), HeadingFont: j.Typography.HeadingFont,
			BodyFont: j.Typography.BodyFont, Scale: j.Typography.Scale,
		},
		Layout:  theme.LayoutSettings{MaxWidth: j.Layout.MaxWidth, Gutter: j.Layout.Gutter},
		Buttons: theme.ButtonSettings{Radius: j.Buttons.Radius, Size: j.Buttons.Size, Style: j.Buttons.Style},
		Product: theme.ProductSettings{GridColumns: j.Product.GridColumns, ImageRatio: j.Product.ImageRatio},
		Header:  theme.HeaderSettings{Layout: j.Header.Layout, Sticky: j.Header.Sticky},
		Footer:  theme.FooterSettings{Layout: j.Footer.Layout},
	}
}
