package planning

import "time"

// PlanningSKU is a catalog SKU as the planner sees it.
type PlanningSKU struct {
	SKUID       string
	SKUCode     string
	ProductName string
	VariantName string
	// IsWeightPool: stock is grams, bought per kg (pms.SKUUnitTypeKilo).
	IsWeightPool bool
	// PortionedFromSKUID/WeightGrams are set on a weight pool's package
	// SKU (e.g. "~380g") -- its sales and stock are the pool's grams.
	PortionedFromSKUID *string
	WeightGrams        *int64
	// Active: the SKU and its product are still sold.
	Active bool
	// OnHand is stock across every warehouse, in stock units.
	OnHand int64
}

// Rollup folds weight-pool packages into their pool: a package is never
// bought, its pool is, so package sales/stock count as the pool's grams.
type Rollup struct {
	skus []PlanningSKU
	// packages maps a package SKU to its pool and grams per package.
	packages map[string]packageOf
	// unresolved are packages whose weight or pool is unknown.
	unresolved map[string]bool
}

type packageOf struct {
	poolID string
	grams  int64
}

func NewRollup(skus []PlanningSKU) *Rollup {
	r := &Rollup{skus: skus, packages: map[string]packageOf{}, unresolved: map[string]bool{}}
	pools := map[string]bool{}
	for _, s := range skus {
		if s.IsWeightPool {
			pools[s.SKUID] = true
		}
	}
	for _, s := range skus {
		if s.PortionedFromSKUID == nil {
			continue
		}
		if s.WeightGrams != nil && pools[*s.PortionedFromSKUID] {
			r.packages[s.SKUID] = packageOf{poolID: *s.PortionedFromSKUID, grams: *s.WeightGrams}
		} else {
			r.unresolved[s.SKUID] = true
		}
	}
	return r
}

// Plannable is every SKU worth a recommendation: active, and not a package.
func (r *Rollup) Plannable() []PlanningSKU {
	var out []PlanningSKU
	for _, s := range r.skus {
		if s.Active && s.PortionedFromSKUID == nil {
			out = append(out, s)
		}
	}
	return out
}

// Units rolls per-SKU quantities up: a package's units become units x
// grams on its pool. A package without a weight or a known pool can't be
// converted and is dropped.
func (r *Rollup) Units(perSKU map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(perSKU))
	for id, qty := range perSKU {
		if key, factor, ok := r.target(id); ok {
			out[key] += qty * factor
		}
	}
	return out
}

// Earliest rolls per-SKU dates up, keeping the earliest per target.
func (r *Rollup) Earliest(perSKU map[string]time.Time) map[string]time.Time {
	out := make(map[string]time.Time, len(perSKU))
	for id, at := range perSKU {
		key, _, ok := r.target(id)
		if !ok {
			continue
		}
		if prev, seen := out[key]; !seen || at.Before(prev) {
			out[key] = at
		}
	}
	return out
}

func (r *Rollup) target(skuID string) (string, int64, bool) {
	if p, ok := r.packages[skuID]; ok {
		return p.poolID, p.grams, true
	}
	if r.unresolved[skuID] {
		return "", 0, false
	}
	return skuID, 1, true
}
