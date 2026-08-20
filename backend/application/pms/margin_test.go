package pms_test

import (
	"testing"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
)

func TestEffectiveTargetMarginBasisPoints_categoryOverrideWins(t *testing.T) {
	category := &pmsDomain.Category{TargetMarginBasisPoints: intPtr(4000)}
	assert.Equal(t, 4000, apppms.EffectiveTargetMarginBasisPoints(category, 3000))
}

func TestEffectiveTargetMarginBasisPoints_fallsBackToTenantDefault(t *testing.T) {
	assert.Equal(t, 3000, apppms.EffectiveTargetMarginBasisPoints(nil, 3000))
	assert.Equal(t, 3000, apppms.EffectiveTargetMarginBasisPoints(&pmsDomain.Category{}, 3000))
}

func TestRecommendedPriceMinorUnits_appliesTargetMargin(t *testing.T) {
	// cost 700, target margin 30% -> price such that (price-700)/price = 0.30 -> price = 1000
	assert.Equal(t, int64(1000), apppms.RecommendedPriceMinorUnits(700, 3000))
}

func TestRecommendedPriceMinorUnits_zeroMarginReturnsCostUnchanged(t *testing.T) {
	assert.Equal(t, int64(700), apppms.RecommendedPriceMinorUnits(700, 0))
}
