package pms_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestPackagePriceMinorUnits_roundsDownToAPriceEndingIn9(t *testing.T) {
	cases := []struct {
		name          string
		perKilo, gram int64
		want          int64
	}{
		{"29.99/kg x 380g = 11.40 -> 11.39", 2999, 380, 1139},
		{"32.99/kg x 380g = 12.54 -> 12.49", 3299, 380, 1249},
		{"already ends in 9 stays", 1000, 1249, 1249}, // 12.49
		{"x8 drops to the x9 below", 1000, 12480, 12479}, // 124.80 -> 124.79
		{"29.99/kg x 528g = 15.83 -> 15.79", 2999, 528, 1579},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pmsDomain.PackagePriceMinorUnits(tc.perKilo, tc.gram)
			assert.Equal(t, tc.want, got)
			exact := (tc.perKilo*tc.gram + 500) / 1000
			assert.LessOrEqual(t, got, exact, "never more than the scale's own label price")
			assert.Equal(t, int64(9), got%10)
		})
	}
}

func TestPackagePriceMinorUnits_keepsATinyPriceExact(t *testing.T) {
	// 0.05 can't round down to an x9 ending -- keep it rather than go to 0.
	assert.Equal(t, int64(5), pmsDomain.PackagePriceMinorUnits(100, 50))
}
