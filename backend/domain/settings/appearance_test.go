package settings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/settings"
)

func validAppearanceInput() settings.AppearanceInput {
	return settings.AppearanceInput{
		DefaultLook: "graphite", Corners: "round", TextSize: "large", Density: "compact",
	}
}

func TestNewAppearance_ValidInput(t *testing.T) {
	a, err := settings.NewAppearance("tenant-1", validAppearanceInput(), now)

	require.NoError(t, err)
	assert.Equal(t, settings.Appearance{
		TenantID: "tenant-1", DefaultLook: settings.LookGraphite,
		Corners: settings.CornersRound, TextSize: settings.TextSizeLarge, Density: settings.DensityCompact,
		UpdatedAt: now,
	}, *a)
}

func TestNewAppearance_AcceptsEveryOption(t *testing.T) {
	for _, look := range []string{"wallet", "graphite"} {
		in := validAppearanceInput()
		in.DefaultLook = look
		_, err := settings.NewAppearance("t", in, now)
		assert.NoError(t, err, look)
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
		"unknown look":      {func(in *settings.AppearanceInput) { in.DefaultLook = "classic" }, settings.ErrInvalidAppearanceLook},
		"empty look":        {func(in *settings.AppearanceInput) { in.DefaultLook = "" }, settings.ErrInvalidAppearanceLook},
		"look wrong case":   {func(in *settings.AppearanceInput) { in.DefaultLook = "Wallet" }, settings.ErrInvalidAppearanceLook},
		"unknown corners":   {func(in *settings.AppearanceInput) { in.Corners = "pill" }, settings.ErrInvalidAppearanceCorners},
		"unknown text size": {func(in *settings.AppearanceInput) { in.TextSize = "huge" }, settings.ErrInvalidAppearanceTextSize},
		"unknown density":   {func(in *settings.AppearanceInput) { in.Density = "cosy" }, settings.ErrInvalidAppearanceDensity},
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

func TestDefaultAppearance(t *testing.T) {
	a := settings.DefaultAppearance("tenant-1")

	assert.Equal(t, settings.Appearance{
		TenantID: "tenant-1", DefaultLook: settings.LookWallet,
		Corners: settings.CornersNormal, TextSize: settings.TextSizeNormal, Density: settings.DensityComfortable,
	}, a)
	assert.True(t, a.UpdatedAt.IsZero(), "defaults were never saved")
}
