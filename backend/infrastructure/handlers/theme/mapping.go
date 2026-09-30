package theme

import (
	"encoding/json"

	"github.com/google/uuid"

	gentheme "github.com/adirkuhn/commerceos/api/theme"
	apptheme "github.com/adirkuhn/commerceos/application/theme"
	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/theme"
)

func toUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s) // ids come from the database, always UUIDs
	return id
}

func toUUIDPtr(s *string) *uuid.UUID {
	if s == nil {
		return nil
	}
	id := toUUID(*s)
	return &id
}

func fromUUIDPtr(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

func toGenTheme(t theme.Theme) gentheme.Theme {
	return gentheme.Theme{
		Id: toUUID(t.ID), SalesChannelId: toUUID(t.SalesChannelID), Name: t.Name, PresetKey: t.PresetKey,
		PublishedVersionId: toUUIDPtr(t.PublishedVersionID), IsLive: t.IsLive, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func toGenSummary(v theme.Version) gentheme.VersionSummary {
	return gentheme.VersionSummary{
		Id: toUUID(v.ID), Number: v.Number, State: gentheme.VersionState(v.State), Revision: v.Revision,
		Note: v.Note, CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt, SealedAt: v.SealedAt,
	}
}

func toGenSummaryPtr(v *theme.Version) *gentheme.VersionSummary {
	if v == nil {
		return nil
	}
	s := toGenSummary(*v)
	return &s
}

func toGenVersion(v theme.Version) gentheme.Version {
	return gentheme.Version{
		Id: toUUID(v.ID), Number: v.Number, State: gentheme.VersionState(v.State), Revision: v.Revision,
		Note: v.Note, CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt, SealedAt: v.SealedAt,
		Settings: toGenSettings(v.Settings),
	}
}

func toGenSummaries(docs []theme.Document) []gentheme.DocumentSummary {
	out := make([]gentheme.DocumentSummary, len(docs))
	for i, d := range docs {
		out[i] = gentheme.DocumentSummary{Kind: gentheme.DocumentKind(d.Kind), Key: d.Key, Title: d.Title}
	}
	return out
}

func toGenDraft(v apptheme.VersionView) gentheme.Draft {
	return gentheme.Draft{
		Theme: toGenTheme(v.Theme), Version: toGenVersion(v.Version), Documents: toGenSummaries(v.Documents),
		SalesChannelId: toUUID(v.Theme.SalesChannelID), Assets: toGenAssetRefs(v.Assets),
	}
}

func toGenAssetRefs(assets map[string]theme.Asset) *gentheme.AssetRefs {
	refs := gentheme.AssetRefs{}
	for id, a := range assets {
		refs[id] = gentheme.AssetRef{Url: a.URL, Width: a.Width, Height: a.Height, Alt: a.Alt}
	}
	return &refs
}

func toGenAsset(a theme.Asset) gentheme.Asset {
	return gentheme.Asset{
		Id: toUUID(a.ID), Url: a.URL, Filename: a.Filename, ContentType: a.ContentType, Bytes: a.Bytes,
		Width: a.Width, Height: a.Height, Alt: a.Alt, CreatedAt: a.CreatedAt,
	}
}

func toGenDocumentView(v apptheme.DocumentView) gentheme.Document {
	d := toGenDocument(v.Document)
	d.Assets = toGenAssetRefs(v.Assets)
	return d
}

func toGenSEO(s theme.SEO) gentheme.SEO {
	title, desc := s.Title, s.Description
	seo := gentheme.SEO{}
	if title != "" {
		seo.Title = &title
	}
	if desc != "" {
		seo.Description = &desc
	}
	return seo
}

func toGenDocument(d theme.Document) gentheme.Document {
	out := gentheme.Document{
		Kind: gentheme.DocumentKind(d.Kind), Key: d.Key, Title: d.Title, Seo: toGenSEO(d.SEO), Root: toGenNode(d.Root),
	}
	if len(d.Translations) > 0 {
		tr := make(gentheme.DocumentTranslations, len(d.Translations))
		for l, t := range d.Translations {
			texts := gentheme.DocumentTexts{}
			if t.Title != "" {
				title := t.Title
				texts.Title = &title
			}
			if t.SEO != (theme.SEO{}) {
				seo := toGenSEO(t.SEO)
				texts.Seo = &seo
			}
			tr[string(l)] = texts
		}
		out.Translations = &tr
	}
	return out
}

func fromGenDocumentTranslations(tr *gentheme.DocumentTranslations) map[locale.Locale]theme.DocumentTexts {
	if tr == nil {
		return nil
	}
	out := make(map[locale.Locale]theme.DocumentTexts, len(*tr))
	for l, t := range *tr {
		texts := theme.DocumentTexts{SEO: fromGenSEO(t.Seo)}
		if t.Title != nil {
			texts.Title = *t.Title
		}
		out[locale.Locale(l)] = texts
	}
	return out
}

func fromGenSEO(s *gentheme.SEO) theme.SEO {
	var out theme.SEO
	if s == nil {
		return out
	}
	if s.Title != nil {
		out.Title = *s.Title
	}
	if s.Description != nil {
		out.Description = *s.Description
	}
	return out
}

// nodeJSON is the wire shape of a node; converting through it avoids
// mapping every one of the generated StyleValues pointer fields by hand.
type nodeJSON struct {
	ID           string                           `json:"id"`
	Type         string                           `json:"type"`
	Props        map[string]any                   `json:"props,omitempty"`
	Translations map[locale.Locale]map[string]any `json:"translations,omitempty"`
	Style        *styleJSON                       `json:"style,omitempty"`
	Children     []nodeJSON                       `json:"children,omitempty"`
}

type styleJSON struct {
	Base   map[string]string `json:"base,omitempty"`
	Tablet map[string]string `json:"tablet,omitempty"`
	Mobile map[string]string `json:"mobile,omitempty"`
}

func toNodeJSON(n theme.Node) nodeJSON {
	out := nodeJSON{ID: n.ID, Type: n.Type, Props: n.Props, Translations: n.Translations}
	if len(n.Style.Base)+len(n.Style.Tablet)+len(n.Style.Mobile) > 0 {
		out.Style = &styleJSON{Base: n.Style.Base, Tablet: n.Style.Tablet, Mobile: n.Style.Mobile}
	}
	for _, c := range n.Children {
		out.Children = append(out.Children, toNodeJSON(c))
	}
	return out
}

func (n nodeJSON) toDomain() theme.Node {
	out := theme.Node{ID: n.ID, Type: n.Type, Props: n.Props, Translations: n.Translations}
	if n.Style != nil {
		out.Style = theme.NodeStyle{Base: n.Style.Base, Tablet: n.Style.Tablet, Mobile: n.Style.Mobile}
	}
	for _, c := range n.Children {
		out.Children = append(out.Children, c.toDomain())
	}
	return out
}

func toGenNode(n theme.Node) gentheme.ThemeNode {
	var out gentheme.ThemeNode
	b, _ := json.Marshal(toNodeJSON(n)) // plain maps/strings: cannot fail
	_ = json.Unmarshal(b, &out)
	return out
}

func fromGenNode(n gentheme.ThemeNode) (theme.Node, error) {
	b, err := json.Marshal(n)
	if err != nil {
		return theme.Node{}, err
	}
	var j nodeJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return theme.Node{}, err
	}
	return j.toDomain(), nil
}

func toGenSettings(s theme.Settings) gentheme.ThemeSettings {
	var g gentheme.ThemeSettings
	g.Brand.LogoAssetId, g.Brand.FaviconAssetId = toUUIDPtr(s.Brand.LogoAssetID), toUUIDPtr(s.Brand.FaviconAssetID)
	c := s.Colors
	g.Colors.Primary, g.Colors.Secondary, g.Colors.Background = c.Primary, c.Secondary, c.Background
	g.Colors.Surface, g.Colors.Text, g.Colors.Muted, g.Colors.Accent = c.Surface, c.Text, c.Muted, c.Accent
	g.Typography.Source = gentheme.ThemeSettingsTypographySource(s.Typography.Source)
	g.Typography.HeadingFont, g.Typography.BodyFont = s.Typography.HeadingFont, s.Typography.BodyFont
	g.Typography.Scale = gentheme.ThemeSettingsTypographyScale(s.Typography.Scale)
	g.Layout.MaxWidth, g.Layout.Gutter = s.Layout.MaxWidth, gentheme.SpacingScale(s.Layout.Gutter)
	g.Buttons.Radius = gentheme.ThemeSettingsButtonsRadius(s.Buttons.Radius)
	g.Buttons.Size = gentheme.ThemeSettingsButtonsSize(s.Buttons.Size)
	g.Buttons.Style = gentheme.ThemeSettingsButtonsStyle(s.Buttons.Style)
	g.Product.GridColumns = s.Product.GridColumns
	g.Product.ImageRatio = gentheme.ThemeSettingsProductImageRatio(s.Product.ImageRatio)
	g.Header.Layout, g.Header.Sticky = gentheme.ThemeSettingsHeaderLayout(s.Header.Layout), s.Header.Sticky
	g.Footer.Layout = gentheme.ThemeSettingsFooterLayout(s.Footer.Layout)
	style := gentheme.ThemeSettingsStyle(theme.StyleDefault)
	if s.Style != "" {
		style = gentheme.ThemeSettingsStyle(s.Style)
	}
	g.Style = &style
	return g
}

func fromGenSettings(g gentheme.ThemeSettings) theme.Settings {
	return theme.Settings{
		Brand: theme.BrandSettings{LogoAssetID: fromUUIDPtr(g.Brand.LogoAssetId), FaviconAssetID: fromUUIDPtr(g.Brand.FaviconAssetId)},
		Colors: theme.ColorSettings{
			Primary: g.Colors.Primary, Secondary: g.Colors.Secondary, Background: g.Colors.Background,
			Surface: g.Colors.Surface, Text: g.Colors.Text, Muted: g.Colors.Muted, Accent: g.Colors.Accent,
		},
		Typography: theme.TypographySettings{
			Source: theme.FontSource(g.Typography.Source), HeadingFont: g.Typography.HeadingFont,
			BodyFont: g.Typography.BodyFont, Scale: string(g.Typography.Scale),
		},
		Layout:  theme.LayoutSettings{MaxWidth: g.Layout.MaxWidth, Gutter: string(g.Layout.Gutter)},
		Buttons: theme.ButtonSettings{Radius: string(g.Buttons.Radius), Size: string(g.Buttons.Size), Style: string(g.Buttons.Style)},
		Product: theme.ProductSettings{GridColumns: g.Product.GridColumns, ImageRatio: string(g.Product.ImageRatio)},
		Header:  theme.HeaderSettings{Layout: string(g.Header.Layout), Sticky: g.Header.Sticky},
		Footer:  theme.FooterSettings{Layout: string(g.Footer.Layout)},
		Style:   styleOr(g.Style),
	}
}

func styleOr(s *gentheme.ThemeSettingsStyle) string {
	if s == nil {
		return theme.StyleDefault
	}
	return string(*s)
}
