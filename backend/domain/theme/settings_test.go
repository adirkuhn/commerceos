package theme_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/theme"
)

func TestDefaultSettings_areValid(t *testing.T) {
	assert.NoError(t, theme.DefaultSettings().Validate())
}

func TestSettings_Validate(t *testing.T) {
	mutate := map[string]func(*theme.Settings){
		"bad color":            func(s *theme.Settings) { s.Colors.Primary = "blue" },
		"bad font source":      func(s *theme.Settings) { s.Typography.Source = "adobe" },
		"unknown system font":  func(s *theme.Settings) { s.Typography.HeadingFont = "Comic Sans" },
		"google font injected": func(s *theme.Settings) { s.Typography.Source = "google"; s.Typography.BodyFont = "Inter&family=x" },
		"max width too small":  func(s *theme.Settings) { s.Layout.MaxWidth = 100 },
		"grid columns":         func(s *theme.Settings) { s.Product.GridColumns = 9 },
		"image ratio":          func(s *theme.Settings) { s.Product.ImageRatio = "5:1" },
		"button radius":        func(s *theme.Settings) { s.Buttons.Radius = "huge" },
		"header layout":        func(s *theme.Settings) { s.Header.Layout = "floating" },
		"logo not uuid":        func(s *theme.Settings) { v := "logo.png"; s.Brand.LogoAssetID = &v },
	}
	for name, m := range mutate {
		s := theme.DefaultSettings()
		m(&s)
		assert.ErrorIs(t, s.Validate(), theme.ErrInvalidSettings, name)
	}
}

func TestSettings_googleFontsAcceptFamilyNames(t *testing.T) {
	s := theme.DefaultSettings()
	s.Typography.Source = theme.FontSourceGoogle
	s.Typography.HeadingFont = "Playfair Display"
	s.Typography.BodyFont = "Inter"

	assert.NoError(t, s.Validate())
}
