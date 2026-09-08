package channels_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/channels"
)

func TestEffectivePriceMinorUnits_noAdjustmentReturnsPriceUnchanged(t *testing.T) {
	ch := channels.SalesChannel{}

	assert.Equal(t, int64(1000), ch.EffectivePriceMinorUnits(1000))
}

func TestEffectivePriceMinorUnits_zeroAdjustmentReturnsPriceUnchanged(t *testing.T) {
	zero := 0
	ch := channels.SalesChannel{PriceAdjustmentBasisPoints: &zero}

	assert.Equal(t, int64(1000), ch.EffectivePriceMinorUnits(1000))
}

func TestEffectivePriceMinorUnits_appliesDiscountBasisPoints(t *testing.T) {
	twentyPercent := 2000
	ch := channels.SalesChannel{PriceAdjustmentBasisPoints: &twentyPercent}

	assert.Equal(t, int64(800), ch.EffectivePriceMinorUnits(1000))
}

func TestEffectivePriceMinorUnits_fullDiscountReturnsZero(t *testing.T) {
	hundredPercent := 10000
	ch := channels.SalesChannel{PriceAdjustmentBasisPoints: &hundredPercent}

	assert.Equal(t, int64(0), ch.EffectivePriceMinorUnits(1000))
}

func TestEffectivePriceMinorUnits_roundsDown(t *testing.T) {
	tenPercent := 1000
	ch := channels.SalesChannel{PriceAdjustmentBasisPoints: &tenPercent}

	// 999 * 0.9 = 899.1 -> 899, not 900.
	assert.Equal(t, int64(899), ch.EffectivePriceMinorUnits(999))
}

func TestEffectivePriceMinorUnitsForQuantity_belowEveryBreakUsesFlatDiscount(t *testing.T) {
	tenPercent := 1000
	ch := channels.SalesChannel{
		PriceAdjustmentBasisPoints: &tenPercent,
		QuantityBreaks:             []channels.QuantityBreak{{MinQuantity: 10, DiscountBasisPoints: 3000}},
	}

	assert.Equal(t, int64(900), ch.EffectivePriceMinorUnitsForQuantity(1000, 5))
}

func TestEffectivePriceMinorUnitsForQuantity_meetingABreakUsesItsBiggerDiscount(t *testing.T) {
	tenPercent := 1000
	ch := channels.SalesChannel{
		PriceAdjustmentBasisPoints: &tenPercent,
		QuantityBreaks:             []channels.QuantityBreak{{MinQuantity: 10, DiscountBasisPoints: 3000}},
	}

	assert.Equal(t, int64(700), ch.EffectivePriceMinorUnitsForQuantity(1000, 10))
	assert.Equal(t, int64(700), ch.EffectivePriceMinorUnitsForQuantity(1000, 50)) // above MinQuantity too
}

func TestEffectivePriceMinorUnitsForQuantity_picksTheHighestQualifyingTier(t *testing.T) {
	ch := channels.SalesChannel{
		QuantityBreaks: []channels.QuantityBreak{
			{MinQuantity: 10, DiscountBasisPoints: 1000},
			{MinQuantity: 50, DiscountBasisPoints: 2500},
		},
	}

	assert.Equal(t, int64(900), ch.EffectivePriceMinorUnitsForQuantity(1000, 10))
	assert.Equal(t, int64(750), ch.EffectivePriceMinorUnitsForQuantity(1000, 50))
}

func TestEffectivePriceMinorUnitsForQuantity_neverWorseThanTheFlatDiscount(t *testing.T) {
	twentyPercent := 2000
	ch := channels.SalesChannel{
		PriceAdjustmentBasisPoints: &twentyPercent,
		// A break the flat rate already beats -- should never make the
		// price worse than the flat 20% off.
		QuantityBreaks: []channels.QuantityBreak{{MinQuantity: 5, DiscountBasisPoints: 500}},
	}

	assert.Equal(t, int64(800), ch.EffectivePriceMinorUnitsForQuantity(1000, 5))
}

func TestEffectivePriceMinorUnitsForQuantity_noBreaksOrFlatDiscountReturnsPriceUnchanged(t *testing.T) {
	ch := channels.SalesChannel{}

	assert.Equal(t, int64(1000), ch.EffectivePriceMinorUnitsForQuantity(1000, 100))
}
