package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/procurement/planning"
)

func ptr[T any](v T) *T { return &v }

// Carne seca: a weight pool (stock in grams) whose ~380g and ~540g
// packages are their own SKUs. Package sales/stock count toward the pool in
// grams -- the pool is what gets bought, never a package.
func rollupSKUs() []planning.PlanningSKU {
	return []planning.PlanningSKU{
		{SKUID: "pool", IsWeightPool: true, Active: true},
		{SKUID: "p380", PortionedFromSKUID: ptr("pool"), WeightGrams: ptr(int64(380)), Active: true},
		{SKUID: "p540", PortionedFromSKUID: ptr("pool"), WeightGrams: ptr(int64(540)), Active: false},
		{SKUID: "noweight", PortionedFromSKUID: ptr("pool")},
		{SKUID: "orphan", PortionedFromSKUID: ptr("gone"), WeightGrams: ptr(int64(100))},
		{SKUID: "rice", Active: true},
	}
}

func TestRollup_UnitsMovePackagesIntoTheirPoolInGrams(t *testing.T) {
	r := planning.NewRollup(rollupSKUs())

	got := r.Units(map[string]int64{"pool": 1000, "p380": 2, "p540": 1, "noweight": 9, "orphan": 4, "rice": 3})

	assert.Equal(t, map[string]int64{"pool": 1000 + 760 + 540, "rice": 3}, got)
}

func TestRollup_EarliestTakesThePackagesFirstSaleForThePool(t *testing.T) {
	r := planning.NewRollup(rollupSKUs())
	early := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	got := r.Earliest(map[string]time.Time{"pool": late, "p380": early, "rice": late, "orphan": early})

	assert.Equal(t, map[string]time.Time{"pool": early, "rice": late}, got)
}

func TestRollup_PlannableExcludesPackagesAndInactive(t *testing.T) {
	r := planning.NewRollup(rollupSKUs())

	var ids []string
	for _, s := range r.Plannable() {
		ids = append(ids, s.SKUID)
	}
	assert.Equal(t, []string{"pool", "rice"}, ids)
}
