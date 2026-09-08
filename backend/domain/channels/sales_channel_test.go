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
