package channels

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundUpToPriceEnding(t *testing.T) {
	cases := map[int64]int64{3520: 3549, 3549: 3549, 3550: 3599, 3559: 3599, 3599: 3599, 3600: 3649, 1: 49}
	for in, want := range cases {
		assert.Equal(t, want, RoundUpToPriceEnding(in), "in=%d", in)
	}
}

func TestRoundDownToPriceEnding(t *testing.T) {
	cases := map[int64]int64{3540: 3499, 3549: 3549, 3598: 3549, 3599: 3599, 3700: 3699}
	for in, want := range cases {
		assert.Equal(t, want, RoundDownToPriceEnding(in), "in=%d", in)
	}
}

// R €25.00 at 9% VAT, shipping €4.95, Bol fee €0.30 + 15% (+21% VAT on the
// fee): the price where one unit in one parcel leaves what the webshop
// leaves at €25.00.
func TestBolBreakEvenPrice(t *testing.T) {
	in := BolPriceInputs{RegularPriceMinorUnits: 2500, ProductVATBasisPoints: 900, ShippingMinorUnits: 495}
	fee := BolFee{FixedMinorUnits: 30, PercentBasisPoints: 1500}

	p1, err := BolBreakEvenPrice(in, fee, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(3839), p1)

	p2, err := BolBreakEvenPrice(in, fee, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(3503), p2, "two units share one parcel")

	e := BolUnitEconomicsAt(in, fee, 1, p1)
	assert.GreaterOrEqual(t, e.NetMinorUnits, e.RegularNetMinorUnits)
	e = BolUnitEconomicsAt(in, fee, 1, p1-1)
	assert.Less(t, e.NetMinorUnits, e.RegularNetMinorUnits, "one cent less no longer breaks even")
}

func TestBolBreakEvenPrice_noFeesNoShippingIsTheRegularPrice(t *testing.T) {
	p, err := BolBreakEvenPrice(BolPriceInputs{RegularPriceMinorUnits: 1999, ProductVATBasisPoints: 2100}, BolFee{}, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1999), p)
}

func TestBolBreakEvenPrice_feeTooHighCannotBreakEven(t *testing.T) {
	_, err := BolBreakEvenPrice(BolPriceInputs{RegularPriceMinorUnits: 1000}, BolFee{PercentBasisPoints: 9000}, 1)
	assert.ErrorIs(t, err, ErrBolFeeTooHigh)
}

func TestBolUnitEconomicsAt(t *testing.T) {
	in := BolPriceInputs{RegularPriceMinorUnits: 2500, ProductVATBasisPoints: 900, ShippingMinorUnits: 495}
	e := BolUnitEconomicsAt(in, BolFee{FixedMinorUnits: 30, PercentBasisPoints: 1500}, 1, 3849)

	assert.Equal(t, int64(3849), e.UnitPriceMinorUnits)
	assert.Equal(t, int64(318), e.ProductVATMinorUnits)   // 38.49 × 9/109
	assert.Equal(t, int64(735), e.BolFeeInclVATMinorUnits) // (0.30 + 5.77) × 1.21
	assert.Equal(t, int64(495), e.ShippingShareMinorUnits)
	assert.Equal(t, int64(2301), e.NetMinorUnits)
	assert.Equal(t, int64(2294), e.RegularNetMinorUnits) // 25.00 × 100/109
	assert.Equal(t, int64(3849), e.TotalPriceMinorUnits)
	assert.Equal(t, int64(2301), e.TotalNetMinorUnits)
}

// A bulk level's totals are for its minimum quantity in one parcel.
func TestBolUnitEconomicsAt_totalsForTheLevelsQuantity(t *testing.T) {
	in := BolPriceInputs{RegularPriceMinorUnits: 2500, ProductVATBasisPoints: 900, ShippingMinorUnits: 495}
	e := BolUnitEconomicsAt(in, BolFee{FixedMinorUnits: 30, PercentBasisPoints: 1500}, 6, 3299)

	assert.Equal(t, int64(2309), e.NetMinorUnits)
	assert.Equal(t, int64(19794), e.TotalPriceMinorUnits, "6 × 32.99")
	assert.Equal(t, int64(13854), e.TotalNetMinorUnits, "6 × 23.0904, rounded once")
}

func constFee(f BolFee) func(int64) (BolFee, error) {
	return func(int64) (BolFee, error) { return f, nil }
}

func TestResolveBolPrices_autoTiersBreakEvenWithSharedShipping(t *testing.T) {
	in := BolPriceInputs{RegularPriceMinorUnits: 2500, ProductVATBasisPoints: 900, ShippingMinorUnits: 495}
	tiers := []BolBundleTier{{Quantity: 2}, {Quantity: 4}, {Quantity: 6}}

	prices, err := ResolveBolPrices(in, nil, tiers, constFee(BolFee{FixedMinorUnits: 30, PercentBasisPoints: 1500}))

	require.NoError(t, err)
	require.Len(t, prices, 4)
	assert.Equal(t, []int{1, 2, 4, 6}, []int{prices[0].Quantity, prices[1].Quantity, prices[2].Quantity, prices[3].Quantity})
	assert.Equal(t, int64(3849), prices[0].UnitPriceMinorUnits, "break-even 38.39, rounded up to .49/.99")
	assert.Equal(t, int64(3549), prices[1].UnitPriceMinorUnits, "shared parcel: break-even 35.03")
	assert.Equal(t, int64(3349), prices[2].UnitPriceMinorUnits)
	assert.Equal(t, int64(3299), prices[3].UnitPriceMinorUnits, "break-even 32.78")
	for _, p := range prices {
		assert.True(t, p.Auto)
		assert.False(t, p.BelowRegular())
	}
}

// Without shipping to share, every quantity breaks even at the same price;
// Bol needs each level lower than the one before, so an automatic level
// drops to the next lower .49/.99 ending.
func TestResolveBolPrices_autoTierIsAlwaysLowerThanTheLevelBefore(t *testing.T) {
	prices, err := ResolveBolPrices(BolPriceInputs{RegularPriceMinorUnits: 2500}, nil, []BolBundleTier{{Quantity: 2}, {Quantity: 3}}, constFee(BolFee{}))

	require.NoError(t, err)
	assert.Equal(t, []int64{2549, 2499, 2449}, []int64{prices[0].UnitPriceMinorUnits, prices[1].UnitPriceMinorUnits, prices[2].UnitPriceMinorUnits})
	assert.False(t, prices[0].BelowRegular())
	assert.True(t, prices[2].BelowRegular(), "staying lower than the level before costs margin here")
}

func TestResolveBolPrices_manualPricesWin(t *testing.T) {
	in := BolPriceInputs{RegularPriceMinorUnits: 2500}
	custom := int64(3000)
	manual := int64(2500)
	prices, err := ResolveBolPrices(in, &custom, []BolBundleTier{{Quantity: 2, UnitPriceMinorUnits: &manual}}, constFee(BolFee{}))

	require.NoError(t, err)
	assert.Equal(t, int64(3000), prices[0].UnitPriceMinorUnits)
	assert.False(t, prices[0].Auto)
	assert.Equal(t, int64(2500), prices[1].UnitPriceMinorUnits)
	assert.False(t, prices[1].Auto)
}

func TestResolveBolPrices_manualTierNotLowerThanTheLevelBeforeFails(t *testing.T) {
	custom := int64(3000)
	manual := int64(3000)
	_, err := ResolveBolPrices(BolPriceInputs{RegularPriceMinorUnits: 2500}, &custom, []BolBundleTier{{Quantity: 2, UnitPriceMinorUnits: &manual}}, constFee(BolFee{}))

	assert.ErrorIs(t, err, ErrBolBundlePriceStep)
}

func TestResolveBolPrices_outOfBolPriceRangeFails(t *testing.T) {
	custom := int64(50)
	_, err := ResolveBolPrices(BolPriceInputs{RegularPriceMinorUnits: 2500}, &custom, nil, constFee(BolFee{}))
	assert.ErrorIs(t, err, ErrBolPriceOutOfRange)
}

// Bol's fixed fee depends on the price bracket: the price is recalculated
// with the fee of the bracket it lands in.
func TestResolveBolPrices_refetchesTheFeeOfTheNewPriceBracket(t *testing.T) {
	in := BolPriceInputs{RegularPriceMinorUnits: 1900}
	feeAt := func(price int64) (BolFee, error) {
		if price < 2000 {
			return BolFee{FixedMinorUnits: 20, PercentBasisPoints: 1000}, nil
		}
		return BolFee{FixedMinorUnits: 100, PercentBasisPoints: 1000}, nil
	}

	prices, err := ResolveBolPrices(in, nil, nil, feeAt)

	require.NoError(t, err)
	assert.Equal(t, BolFee{FixedMinorUnits: 100, PercentBasisPoints: 1000}, prices[0].Fee)
	assert.False(t, prices[0].BelowRegular(), "priced with the fee that actually applies")
}

func TestResolveBolPrices_feeLookupErrorIsReturned(t *testing.T) {
	boom := errors.New("bol down")
	_, err := ResolveBolPrices(BolPriceInputs{RegularPriceMinorUnits: 1900}, nil, nil, func(int64) (BolFee, error) { return BolFee{}, boom })
	assert.ErrorIs(t, err, boom)
}

func TestValidateBolBundleTiers(t *testing.T) {
	p := func(v int64) *int64 { return &v }
	assert.NoError(t, ValidateBolBundleTiers(DefaultBolBundleTiers()))
	assert.NoError(t, ValidateBolBundleTiers([]BolBundleTier{{Quantity: 2, UnitPriceMinorUnits: p(2500)}, {Quantity: 3}}))
	assert.ErrorIs(t, ValidateBolBundleTiers([]BolBundleTier{{Quantity: 1}}), ErrBolInvalidBundleTiers, "quantity 1 is the regular price")
	assert.ErrorIs(t, ValidateBolBundleTiers([]BolBundleTier{{Quantity: 4}, {Quantity: 2}}), ErrBolInvalidBundleTiers, "increasing quantities")
	assert.ErrorIs(t, ValidateBolBundleTiers([]BolBundleTier{{Quantity: 2}, {Quantity: 2}}), ErrBolInvalidBundleTiers)
	assert.ErrorIs(t, ValidateBolBundleTiers([]BolBundleTier{{Quantity: 2, UnitPriceMinorUnits: p(99)}}), ErrBolInvalidBundleTiers, "Bol's €1 minimum")
	assert.NoError(t, ValidateBolBundleTiers([]BolBundleTier{{Quantity: 2, UnitPriceMinorUnits: p(2500)}, {Quantity: 3, UnitPriceMinorUnits: p(2499)}}), "any lower price will do")
	assert.ErrorIs(t, ValidateBolBundleTiers([]BolBundleTier{{Quantity: 2, UnitPriceMinorUnits: p(2500)}, {Quantity: 3, UnitPriceMinorUnits: p(2500)}}), ErrBolInvalidBundleTiers, "prices must go down")
	assert.ErrorIs(t, ValidateBolBundleTiers([]BolBundleTier{{Quantity: 2}, {Quantity: 3}, {Quantity: 4}, {Quantity: 5}}), ErrBolInvalidBundleTiers, "Bol allows 4 levels including quantity 1")
	assert.ErrorIs(t, ValidateBolBundleTiers([]BolBundleTier{{Quantity: 25}}), ErrBolInvalidBundleTiers, "Bol's maximum quantity is 24")
	assert.NoError(t, ValidateBolBundleTiers([]BolBundleTier{{Quantity: 24}}))
}

func TestDefaultBolBundleTiers(t *testing.T) {
	tiers := DefaultBolBundleTiers()
	require.Len(t, tiers, 3)
	for i, q := range []int{2, 4, 6} {
		assert.Equal(t, q, tiers[i].Quantity)
		assert.Nil(t, tiers[i].UnitPriceMinorUnits, "automatic price")
	}
}
