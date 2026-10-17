package channels

import (
	"fmt"
	"math"
)

// Bol pricing: unless the merchant types a price, a Bol offer is priced so
// that each unit leaves exactly what the same unit leaves when sold in the
// webshop at its regular price -- after product VAT, Bol's fee (plus the
// VAT Bol charges on it) and the shipping paid per parcel. All money is in
// minor units (cents), prices include product VAT.
const (
	// BolFeeVATBasisPoints is the VAT Bol adds to its own fee (21%).
	BolFeeVATBasisPoints = 2100
	// BolBundleMinStepBasisPoints: each bundle tier's unit price is at
	// least 5% below the tier before it.
	BolBundleMinStepBasisPoints = 500
	// BolMinUnitPriceMinorUnits / BolMaxUnitPriceMinorUnits are the unit
	// price limits of Bol's Offers API (unitPrice minimum 1, maximum 9999).
	BolMinUnitPriceMinorUnits = 100
	BolMaxUnitPriceMinorUnits = 999900
)

// defaultBolBundleQuantities are the tiers a product starts with when bulk
// prices are switched on.
var defaultBolBundleQuantities = []int{2, 4, 6, 8}

// BolBundleTier is one bulk price level of a product's Bol offer: from
// Quantity units on, each unit costs UnitPriceMinorUnits. A nil price is
// automatic (see ResolveBolPrices). Quantity 1 is never a tier -- it is the
// offer's own price.
type BolBundleTier struct {
	Quantity            int
	UnitPriceMinorUnits *int64
}

// DefaultBolBundleTiers returns the 2/4/6/8 automatic tiers.
func DefaultBolBundleTiers() []BolBundleTier {
	tiers := make([]BolBundleTier, len(defaultBolBundleQuantities))
	for i, q := range defaultBolBundleQuantities {
		tiers[i] = BolBundleTier{Quantity: q}
	}
	return tiers
}

// ValidateBolBundleTiers checks what can be checked before any price is
// resolved: quantities from 2 up, strictly increasing; typed prices within
// Bol's range and each at least 5% below the previous typed price.
func ValidateBolBundleTiers(tiers []BolBundleTier) error {
	prevQty := 1
	var prevPrice *int64
	for _, t := range tiers {
		if t.Quantity <= prevQty {
			return fmt.Errorf("%w: quantity %d must be above %d", ErrBolInvalidBundleTiers, t.Quantity, prevQty)
		}
		prevQty = t.Quantity
		if t.UnitPriceMinorUnits == nil {
			continue
		}
		p := *t.UnitPriceMinorUnits
		if p < BolMinUnitPriceMinorUnits || p > BolMaxUnitPriceMinorUnits {
			return fmt.Errorf("%w: quantity %d: price must be between 1.00 and 9999.00", ErrBolInvalidBundleTiers, t.Quantity)
		}
		if prevPrice != nil && !isBundleStep(*prevPrice, p) {
			return fmt.Errorf("%w: quantity %d: price must be at least 5%% below the tier before", ErrBolInvalidBundleTiers, t.Quantity)
		}
		prevPrice = &p
	}
	return nil
}

// BolFee is Bol's commission for one unit: a fixed amount plus a
// percentage of the unit price, both excluding the VAT Bol adds.
type BolFee struct {
	FixedMinorUnits    int64
	PercentBasisPoints int64
}

type BolPriceInputs struct {
	// RegularPriceMinorUnits is the webshop price, including product VAT.
	RegularPriceMinorUnits int64
	ProductVATBasisPoints  int64
	// ShippingMinorUnits is what one parcel costs the merchant.
	ShippingMinorUnits int64
}

// BolBreakEvenPrice is the lowest unit price (rounded up to the cent) at
// which each of qty units sent in one parcel leaves what the webshop
// leaves at the regular price:
//
//	P/(1+v) − (F + p·P)·1.21 − S/qty = R/(1+v)
func BolBreakEvenPrice(in BolPriceInputs, fee BolFee, qty int) (int64, error) {
	v := 1 + bp(in.ProductVATBasisPoints)
	feeVAT := 1 + bp(BolFeeVATBasisPoints)
	denominator := 1 - feeVAT*bp(fee.PercentBasisPoints)*v
	if denominator <= 0 {
		return 0, ErrBolFeeTooHigh
	}
	shippingShare := float64(in.ShippingMinorUnits) / float64(qty)
	price := (float64(in.RegularPriceMinorUnits) + v*(feeVAT*float64(fee.FixedMinorUnits)+shippingShare)) / denominator
	return int64(math.Ceil(price - 1e-6)), nil
}

// BolUnitEconomics is what one unit sold at UnitPriceMinorUnits leaves,
// when Quantity units go in one parcel.
type BolUnitEconomics struct {
	Quantity                int
	UnitPriceMinorUnits     int64
	ProductVATMinorUnits    int64
	BolFeeInclVATMinorUnits int64
	ShippingShareMinorUnits int64
	// NetMinorUnits is what is left after VAT, Bol's fee and shipping.
	NetMinorUnits int64
	// RegularNetMinorUnits is what the webshop leaves (regular price minus
	// its VAT) -- the target.
	RegularNetMinorUnits int64
}

func BolUnitEconomicsAt(in BolPriceInputs, fee BolFee, qty int, price int64) BolUnitEconomics {
	v := 1 + bp(in.ProductVATBasisPoints)
	p := float64(price)
	vat := p - p/v
	feeInclVAT := (float64(fee.FixedMinorUnits) + p*bp(fee.PercentBasisPoints)) * (1 + bp(BolFeeVATBasisPoints))
	shipping := float64(in.ShippingMinorUnits) / float64(qty)
	return BolUnitEconomics{
		Quantity:                qty,
		UnitPriceMinorUnits:     price,
		ProductVATMinorUnits:    round(vat),
		BolFeeInclVATMinorUnits: round(feeInclVAT),
		ShippingShareMinorUnits: round(shipping),
		NetMinorUnits:           round(p - vat - feeInclVAT - shipping),
		RegularNetMinorUnits:    round(float64(in.RegularPriceMinorUnits) / v),
	}
}

// BolResolvedPrice is one price level as it goes to Bol.
type BolResolvedPrice struct {
	Quantity            int
	UnitPriceMinorUnits int64
	// Auto: computed, not typed by the merchant.
	Auto      bool
	Fee       BolFee
	Economics BolUnitEconomics
}

// BelowRegular: this level leaves less than the webshop does.
func (p BolResolvedPrice) BelowRegular() bool {
	return p.Economics.NetMinorUnits < p.Economics.RegularNetMinorUnits
}

// maxFeeRounds bounds the bracket search in autoPrice: a new price can land
// in a fee bracket that moves the price again.
const maxFeeRounds = 5

// ResolveBolPrices prices quantity 1 and each tier. customPrice (quantity
// 1) and typed tier prices are used as they are. An automatic level is the
// break-even price for its quantity -- tiers share one parcel's shipping --
// rounded up to a .49/.99 ending; an automatic tier is also capped at 5%
// below the tier before it (rounded down to an ending), which can leave it
// below break-even (BelowRegular). feeAt returns Bol's fee at a unit
// price -- it depends on the price bracket.
func ResolveBolPrices(in BolPriceInputs, customPrice *int64, tiers []BolBundleTier, feeAt func(price int64) (BolFee, error)) ([]BolResolvedPrice, error) {
	levels := append([]BolBundleTier{{Quantity: 1, UnitPriceMinorUnits: customPrice}}, tiers...)
	out := make([]BolResolvedPrice, 0, len(levels))
	for i, level := range levels {
		var price int64
		auto := level.UnitPriceMinorUnits == nil
		if auto {
			var err error
			if price, err = autoPrice(in, level.Quantity, feeAt); err != nil {
				return nil, err
			}
			if i > 0 {
				price = min(price, maxNextBundlePrice(out[i-1].UnitPriceMinorUnits))
			}
		} else {
			price = *level.UnitPriceMinorUnits
		}
		if price < BolMinUnitPriceMinorUnits || price > BolMaxUnitPriceMinorUnits {
			return nil, fmt.Errorf("%w: quantity %d: %s", ErrBolPriceOutOfRange, level.Quantity, formatMinor(price))
		}
		if i > 0 && !isBundleStep(out[i-1].UnitPriceMinorUnits, price) {
			return nil, fmt.Errorf("%w: quantity %d: %s is not 5%% below %s", ErrBolBundlePriceStep,
				level.Quantity, formatMinor(price), formatMinor(out[i-1].UnitPriceMinorUnits))
		}
		fee, err := feeAt(price)
		if err != nil {
			return nil, err
		}
		out = append(out, BolResolvedPrice{
			Quantity: level.Quantity, UnitPriceMinorUnits: price, Auto: auto, Fee: fee,
			Economics: BolUnitEconomicsAt(in, fee, level.Quantity, price),
		})
	}
	return out, nil
}

func autoPrice(in BolPriceInputs, qty int, feeAt func(int64) (BolFee, error)) (int64, error) {
	fee, err := feeAt(in.RegularPriceMinorUnits)
	if err != nil {
		return 0, err
	}
	var price int64
	for range maxFeeRounds {
		breakEven, err := BolBreakEvenPrice(in, fee, qty)
		if err != nil {
			return 0, err
		}
		price = RoundUpToPriceEnding(breakEven)
		next, err := feeAt(price)
		if err != nil {
			return 0, err
		}
		if next == fee {
			break
		}
		fee = next
	}
	return price, nil
}

// RoundUpToPriceEnding is the nearest price ending in .49 or .99 at or
// above minor.
func RoundUpToPriceEnding(minor int64) int64 {
	euros, cents := minor/100, minor%100
	switch {
	case cents <= 49:
		return euros*100 + 49
	default:
		return euros*100 + 99
	}
}

// RoundDownToPriceEnding is the nearest price ending in .49 or .99 at or
// below minor.
func RoundDownToPriceEnding(minor int64) int64 {
	euros, cents := minor/100, minor%100
	switch {
	case cents >= 99:
		return euros*100 + 99
	case cents >= 49:
		return euros*100 + 49
	default:
		return euros*100 - 1
	}
}

func maxNextBundlePrice(prev int64) int64 {
	return RoundDownToPriceEnding(prev * (10000 - BolBundleMinStepBasisPoints) / 10000)
}

func isBundleStep(prev, next int64) bool {
	return next*10000 <= prev*(10000-BolBundleMinStepBasisPoints)
}

func bp(basisPoints int64) float64 { return float64(basisPoints) / 10000 }

func round(minor float64) int64 { return int64(math.Round(minor)) }

func formatMinor(minor int64) string { return fmt.Sprintf("%d.%02d", minor/100, minor%100) }
