package reports

import (
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// itemsetKey builds a stable, order-independent key for comparing itemsets
// in tests without caring what order fpGrowth happened to emit their items
// in.
func itemsetKey(items []string) string {
	sorted := append([]string{}, items...)
	sort.Strings(sorted)
	key := ""
	for i, s := range sorted {
		if i > 0 {
			key += ","
		}
		key += s
	}
	return key
}

func supportByItemset(results []frequentItemset) map[string]int {
	out := make(map[string]int, len(results))
	for _, r := range results {
		out[itemsetKey(r.Items)] = r.Support
	}
	return out
}

// A well-known FP-Growth worked example shape (5 transactions over items
// a-o), minSupport 3. Expected supports below are hand-verified against
// the transaction list itself (not just asserted blind):
//
//	a: T1,T2,T4,T5=4  c: T1,T2,T4,T5=4  e: T1,T3,T4,T5=4  g: T2,T4,T5=3
//	a,c: T1,T2,T4,T5=4  a,e: T1,T4,T5=3  a,g: T2,T4,T5=3  c,e: T1,T4,T5=3
//	c,g: T2,T4,T5=3  a,c,e: T1,T4,T5=3  a,c,g: T2,T4,T5=3
//
// (e,g and anything else only co-occurs in T4+T5 = support 2, below
// minSupport, and must NOT appear in the results.)
func TestFPGrowth_worksThroughAMultiTransactionExample(t *testing.T) {
	transactions := [][]string{
		{"a", "b", "c", "e", "f", "o"},
		{"a", "c", "g"},
		{"e", "i"},
		{"a", "c", "d", "e", "g"},
		{"a", "c", "e", "g", "l"},
	}

	results := fpGrowth(transactions, 3, 5)
	support := supportByItemset(results)

	require.Equal(t, 4, support["a"])
	require.Equal(t, 4, support["c"])
	require.Equal(t, 4, support["e"])
	require.Equal(t, 3, support["g"])
	require.Equal(t, 4, support["a,c"])
	require.Equal(t, 3, support["a,e"])
	require.Equal(t, 3, support["a,g"])
	require.Equal(t, 3, support["c,e"])
	require.Equal(t, 3, support["c,g"])
	require.Equal(t, 3, support["a,c,e"])
	require.Equal(t, 3, support["a,c,g"])

	_, hasEG := support["e,g"]
	require.False(t, hasEG, "e,g only co-occurs in 2 transactions, below minSupport")
	for key, s := range support {
		require.GreaterOrEqualf(t, s, 3, "itemset %s has support below minSupport", key)
	}
}

func TestFPGrowth_findsPairsAndTriplesAboveMinSupport(t *testing.T) {
	transactions := [][]string{
		{"milk", "bread", "butter"},
		{"milk", "bread"},
		{"milk", "bread", "butter"},
		{"bread", "butter"},
		{"milk", "butter"},
	}

	results := fpGrowth(transactions, 2, 5)
	support := supportByItemset(results)

	require.Equal(t, 4, support["milk"])
	require.Equal(t, 4, support["bread"])
	require.Equal(t, 4, support["butter"])
	require.Equal(t, 3, support["bread,milk"])
	require.Equal(t, 3, support["butter,milk"])
	require.Equal(t, 3, support["bread,butter"])
	require.Equal(t, 2, support["bread,butter,milk"])

	// Nothing below minSupport should appear at all.
	for key, s := range support {
		require.GreaterOrEqualf(t, s, 2, "itemset %s has support below minSupport", key)
	}
}

func TestFPGrowth_respectsMinSupportThreshold(t *testing.T) {
	transactions := [][]string{
		{"a", "b"},
		{"a", "c"},
		{"a"},
	}

	// b and c each appear in only 1 transaction -- with minSupport 2,
	// only "a" (support 3) should survive.
	results := fpGrowth(transactions, 2, 5)

	require.Len(t, results, 1)
	require.Equal(t, []string{"a"}, results[0].Items)
	require.Equal(t, 3, results[0].Support)
}

func TestFPGrowth_emptyTransactionsGivesNoResults(t *testing.T) {
	results := fpGrowth(nil, 1, 5)
	require.Empty(t, results)
}

func TestFPGrowth_respectsMaxItemsetSize(t *testing.T) {
	transactions := [][]string{
		{"a", "b", "c", "d"},
		{"a", "b", "c", "d"},
		{"a", "b", "c", "d"},
	}

	results := fpGrowth(transactions, 2, 2)

	for _, r := range results {
		require.LessOrEqualf(t, len(r.Items), 2, "itemset %v exceeds maxItemsetSize", r.Items)
	}
	// With 4 fully-co-occurring items capped at size 2, every pair (6 of
	// them) plus every singleton (4) should be present, nothing larger.
	require.Len(t, results, 4+6)
}

func TestFPGrowth_duplicateItemsWithinATransactionCountOnce(t *testing.T) {
	transactions := [][]string{
		{"a", "a", "b"},
		{"a", "b"},
	}

	results := fpGrowth(transactions, 1, 5)
	support := supportByItemset(results)

	require.Equal(t, 2, support["a"], "duplicate 'a' within one transaction must not inflate its support")
	require.Equal(t, 2, support["a,b"])
}

func TestFPGrowth_deterministicOutputOrderAcrossRuns(t *testing.T) {
	transactions := [][]string{
		{"x", "y", "z"},
		{"x", "y"},
		{"y", "z"},
		{"x", "z"},
	}

	first := fpGrowth(transactions, 2, 5)
	second := fpGrowth(transactions, 2, 5)

	require.True(t, reflect.DeepEqual(first, second), "fpGrowth must be deterministic given the same input")
}
