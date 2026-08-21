package pms_test

import (
	"testing"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
)

func TestEffectiveTargetMarkupBasisPoints_categoryOverrideWins(t *testing.T) {
	category := &pmsDomain.Category{TargetMarkupBasisPoints: intPtr(4000)}
	assert.Equal(t, 4000, apppms.EffectiveTargetMarkupBasisPoints(category, 3000))
}

func TestEffectiveTargetMarkupBasisPoints_fallsBackToTenantDefault(t *testing.T) {
	assert.Equal(t, 3000, apppms.EffectiveTargetMarkupBasisPoints(nil, 3000))
	assert.Equal(t, 3000, apppms.EffectiveTargetMarkupBasisPoints(&pmsDomain.Category{}, 3000))
}

func TestRecommendedPriceMinorUnits_appliesTargetMarkup(t *testing.T) {
	// cost 700, target markup 30% -> price = cost * 1.30 = 910
	assert.Equal(t, int64(910), apppms.RecommendedPriceMinorUnits(700, 3000))
}

func TestRecommendedPriceMinorUnits_zeroMarkupReturnsCostUnchanged(t *testing.T) {
	assert.Equal(t, int64(700), apppms.RecommendedPriceMinorUnits(700, 0))
}

func TestRecommendedPriceMinorUnits_noUpperBoundUnlikeMargin(t *testing.T) {
	// cost 100, 300% markup -> price = 100 * 4 = 400 -- impossible to express
	// as a margin (300% margin is undefined), trivial as a markup.
	assert.Equal(t, int64(400), apppms.RecommendedPriceMinorUnits(100, 30000))
}
