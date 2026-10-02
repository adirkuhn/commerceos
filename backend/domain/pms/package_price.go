package pms

// PackagePriceMinorUnits is the fixed price of a weighed package off a
// weight pool: the pool's price per kilo times the package weight, rounded
// DOWN to the nearest price ending in 9 (11.40 -> 11.39, 12.54 -> 12.49).
// Down, never up: the scale prints its own exact price on the package
// label, and the shop must never charge more than the label says. A price
// too small to round that way (under 0.09) stays exact.
//
// Every package price comes from here -- creating packages, adopting
// hand-made ones, and repricing them when the pool's rate changes -- so
// they can never disagree. Per-kilo scale sales of loose weight are not
// packages and stay exact (see application/pms.lineTotalForScannedWeight).
func PackagePriceMinorUnits(pricePerKiloMinorUnits, weightGrams int64) int64 {
	exact := (pricePerKiloMinorUnits*weightGrams + 500) / 1000
	if exact < 9 {
		return exact
	}
	return exact - (exact+1)%10
}
