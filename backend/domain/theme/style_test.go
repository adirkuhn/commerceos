package theme_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/theme"
)

func TestValidateStyle_acceptsVocabulary(t *testing.T) {
	s := theme.NodeStyle{
		Base: map[string]string{
			"padding_y": "xl", "text_align": "center", "color": "primary", "background_color": "#fff",
			"border_color": "#1A2b3C", "border_radius": "md", "shadow": "sm", "font_size": "2xl",
			"background_image": "asset:0b0c7d1e-6f7a-4b8c-9d0e-1f2a3b4c5d6e", "aspect_ratio": "16:9",
		},
		Tablet: map[string]string{"padding_y": "lg"},
		Mobile: map[string]string{"display": "hide", "columns": "1"},
	}
	assert.NoError(t, theme.ValidateStyle(s))
}

func TestValidateStyle_rejectsOutsideVocabulary(t *testing.T) {
	for name, s := range map[string]theme.NodeStyle{
		"unknown property":   {Base: map[string]string{"position": "absolute"}},
		"bad scale":          {Base: map[string]string{"padding_y": "13px"}},
		"css injection":      {Base: map[string]string{"color": "red;}body{display:none"}},
		"unknown token":      {Base: map[string]string{"color": "chartreuse"}},
		"bad hex":            {Tablet: map[string]string{"color": "#12345"}},
		"url instead asset":  {Base: map[string]string{"background_image": "url(https://evil)"}},
		"bad asset id":       {Base: map[string]string{"background_image": "asset:../x"}},
		"columns out of set": {Mobile: map[string]string{"columns": "9"}},
	} {
		assert.ErrorIs(t, theme.ValidateStyle(s), theme.ErrInvalidStyle, name)
	}
}
