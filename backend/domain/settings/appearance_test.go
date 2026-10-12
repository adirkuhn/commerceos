package settings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/settings"
)

func validAppearanceInput() settings.AppearanceInput {
	return settings.AppearanceInput{
		Font: "inter", AccentColor: "#2f33b8", SidebarColor: "#14163f",
		Corners: "round", TextSize: "large", Density: "compact",
	}
}

func TestNewAppearance_ValidInputIsNormalised(t *testing.T) {
	a, err := settings.NewAppearance("tenant-1", validAppearanceInput(), now)

	require.NoError(t, err)
	assert.Equal(t, settings.Appearance{
		TenantID: "tenant-1", Font: settings.FontInter, AccentColor: "#2F33B8", SidebarColor: "#14163F",
		Corners: settings.CornersRound, TextSize: settings.TextSizeLarge, Density: settings.DensityCompact,
		UpdatedAt: now,
	}, *a)
}

func TestNewAppearance_AcceptsEveryOption(t *testing.T) {
	for _, font := range []string{"atkinson", "inter", "ibm_plex_sans", "source_sans", "barlow"} {
		in := validAppearanceInput()
		in.Font = font
		_, err := settings.NewAppearance("t", in, now)
		assert.NoError(t, err, font)
	}
	for _, c := range []string{"sharp", "normal", "round"} {
		in := validAppearanceInput()
		in.Corners = c
		_, err := settings.NewAppearance("t", in, now)
		assert.NoError(t, err, c)
	}
	for _, s := range []string{"normal", "large", "larger"} {
		in := validAppearanceInput()
		in.TextSize = s
		_, err := settings.NewAppearance("t", in, now)
		assert.NoError(t, err, s)
	}
	for _, d := range []string{"compact", "comfortable"} {
		in := validAppearanceInput()
		in.Density = d
		_, err := settings.NewAppearance("t", in, now)
		assert.NoError(t, err, d)
	}
}

func TestNewAppearance_RejectsInvalidInput(t *testing.T) {
	cases := map[string]struct {
		mutate func(*settings.AppearanceInput)
		want   error
	}{
		"unknown font":            {func(in *settings.AppearanceInput) { in.Font = "comic_sans" }, settings.ErrInvalidAppearanceFont},
		"empty font":              {func(in *settings.AppearanceInput) { in.Font = "" }, settings.ErrInvalidAppearanceFont},
		"accent without hash":     {func(in *settings.AppearanceInput) { in.AccentColor = "2F33B8" }, settings.ErrInvalidAppearanceColor},
		"accent short form":       {func(in *settings.AppearanceInput) { in.AccentColor = "#23B" }, settings.ErrInvalidAppearanceColor},
		"accent not hex":          {func(in *settings.AppearanceInput) { in.AccentColor = "#GGGGGG" }, settings.ErrInvalidAppearanceColor},
		"sidebar named colour":    {func(in *settings.AppearanceInput) { in.SidebarColor = "navy" }, settings.ErrInvalidAppearanceColor},
		"sidebar with alpha":      {func(in *settings.AppearanceInput) { in.SidebarColor = "#14163F80" }, settings.ErrInvalidAppearanceColor},
		"accent too light":        {func(in *settings.AppearanceInput) { in.AccentColor = "#FFFF00" }, settings.ErrAppearanceAccentLowContrast},
		"accent just under 4.5:1": {func(in *settings.AppearanceInput) { in.AccentColor = "#777777" }, settings.ErrAppearanceAccentLowContrast},
		"unknown corners":         {func(in *settings.AppearanceInput) { in.Corners = "pill" }, settings.ErrInvalidAppearanceCorners},
		"unknown text size":       {func(in *settings.AppearanceInput) { in.TextSize = "huge" }, settings.ErrInvalidAppearanceTextSize},
		"unknown density":         {func(in *settings.AppearanceInput) { in.Density = "cosy" }, settings.ErrInvalidAppearanceDensity},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			in := validAppearanceInput()
			tc.mutate(&in)

			a, err := settings.NewAppearance("tenant-1", in, now)

			assert.Nil(t, a)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestNewAppearance_LightSidebarIsAllowed(t *testing.T) {
	in := validAppearanceInput()
	in.SidebarColor = "#FFFFFF" // its text colour is derived, no contrast rule

	_, err := settings.NewAppearance("t", in, now)

	assert.NoError(t, err)
}

func TestDefaultAppearance(t *testing.T) {
	a := settings.DefaultAppearance("tenant-1")

	assert.Equal(t, settings.Appearance{
		TenantID: "tenant-1", Font: settings.FontAtkinson, AccentColor: "#2F33B8", SidebarColor: "#14163F",
		Corners: settings.CornersNormal, TextSize: settings.TextSizeNormal, Density: settings.DensityComfortable,
	}, a)
	assert.True(t, a.UpdatedAt.IsZero(), "defaults were never saved")
	assert.GreaterOrEqual(t, settings.ContrastRatio(a.AccentColor, "#FFFFFF"), settings.MinAccentContrast,
		"the default accent must pass its own rule")
}

func TestContrastRatio_InvalidColourHasNoContrast(t *testing.T) {
	assert.Equal(t, 1.0, settings.ContrastRatio("", "#FFFFFF"))
	assert.Equal(t, 1.0, settings.ContrastRatio("#000000", "#fff"))
}

func TestContrastRatio(t *testing.T) {
	cases := []struct {
		a, b settings.HexColor
		want float64
	}{
		{"#000000", "#FFFFFF", 21},
		{"#FFFFFF", "#000000", 21}, // order does not matter
		{"#FFFFFF", "#FFFFFF", 1},
		{"#2F33B8", "#FFFFFF", 9.18},
		{"#767676", "#FFFFFF", 4.54},
		{"#777777", "#FFFFFF", 4.48},
		{"#FFFF00", "#FFFFFF", 1.07},
		{"#ff0000", "#FFFFFF", 4.00}, // either case,
	}
	for _, tc := range cases {
		t.Run(string(tc.a)+"/"+string(tc.b), func(t *testing.T) {
			assert.InDelta(t, tc.want, settings.ContrastRatio(tc.a, tc.b), 0.01)
		})
	}
}
