package reports

import "sort"

// frequentItemset is one FP-Growth-mined itemset (any size >= 1) with its
// support count -- the number of transactions containing every item in
// Items.
type frequentItemset struct {
	Items   []string
	Support int
}

// fpGrowth mines every frequent itemset (support >= minSupport, size <=
// maxItemsetSize) from transactions using the FP-Growth algorithm: compute
// global item frequencies, drop infrequent items, sort each transaction's
// remaining items by descending global frequency, build a prefix-sharing
// FP-tree from the sorted transactions, then recursively mine each item's
// conditional pattern base. Item order within an itemset is not
// meaningful; duplicate items within one transaction count once.
func fpGrowth(transactions [][]string, minSupport, maxItemsetSize int) []frequentItemset {
	if minSupport < 1 {
		minSupport = 1
	}
	if maxItemsetSize < 1 {
		return nil
	}

	itemCounts := countItems(transactions)
	frequent := make(map[string]int, len(itemCounts))
	for item, count := range itemCounts {
		if count >= minSupport {
			frequent[item] = count
		}
	}
	if len(frequent) == 0 {
		return nil
	}

	order := itemsSortedByFrequencyDesc(frequent)
	rank := make(map[string]int, len(order))
	for i, item := range order {
		rank[item] = i
	}

	tree := newFPTree()
	for _, txn := range transactions {
		filtered := filterAndSortByRank(txn, frequent, rank)
		if len(filtered) > 0 {
			tree.insert(filtered, 1)
		}
	}

	var results []frequentItemset
	mineFPTree(tree, nil, minSupport, maxItemsetSize, &results)
	return results
}

// countItems counts, per item, how many transactions contain it at least
// once (duplicates within one transaction count once).
func countItems(transactions [][]string) map[string]int {
	counts := map[string]int{}
	for _, txn := range transactions {
		for _, item := range dedupe(txn) {
			counts[item]++
		}
	}
	return counts
}

func dedupe(items []string) []string {
	seen := make(map[string]bool, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

// itemsSortedByFrequencyDesc orders frequent items by descending global
// count, tie-broken by item value for a deterministic, reproducible tree
// shape (and therefore deterministic output order) across runs.
func itemsSortedByFrequencyDesc(frequent map[string]int) []string {
	items := make([]string, 0, len(frequent))
	for item := range frequent {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if frequent[items[i]] != frequent[items[j]] {
			return frequent[items[i]] > frequent[items[j]]
		}
		return items[i] < items[j]
	})
	return items
}

// filterAndSortByRank drops items not in frequent and sorts the rest by
// their precomputed global-frequency rank -- the ordering FP-tree
// insertion depends on for prefix sharing to work.
func filterAndSortByRank(txn []string, frequent map[string]int, rank map[string]int) []string {
	filtered := make([]string, 0, len(txn))
	for _, item := range dedupe(txn) {
		if _, ok := frequent[item]; ok {
			filtered = append(filtered, item)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return rank[filtered[i]] < rank[filtered[j]] })
	return filtered
}

// ── FP-tree ──────────────────────────────────────────────────────────────

type fpNode struct {
	item     string
	count    int
	parent   *fpNode
	children map[string]*fpNode
}

// fpTree's headerTable maps each item to every node representing it in
// the tree (its "node-links" in the standard FP-Growth description),
// letting mineFPTree find every occurrence of an item without walking the
// whole tree.
type fpTree struct {
	root        *fpNode
	headerTable map[string][]*fpNode
}

func newFPTree() *fpTree {
	return &fpTree{root: &fpNode{children: map[string]*fpNode{}}, headerTable: map[string][]*fpNode{}}
}

// insert adds one transaction's items (already filtered+sorted by rank) to
// the tree, incrementing count on any shared prefix rather than creating
// new nodes -- the compression that makes an FP-tree usually much smaller
// than the raw transaction list.
func (t *fpTree) insert(items []string, count int) {
	node := t.root
	for _, item := range items {
		child, ok := node.children[item]
		if !ok {
			child = &fpNode{item: item, children: map[string]*fpNode{}, parent: node}
			node.children[item] = child
			t.headerTable[item] = append(t.headerTable[item], child)
		}
		child.count += count
		node = child
	}
}

func totalSupport(nodes []*fpNode) int {
	total := 0
	for _, n := range nodes {
		total += n.count
	}
	return total
}

// conditionalPatternBase collects, for every node representing item in the
// tree, the path of items from the tree's root down to (but excluding)
// that node, weighted by the node's own count -- the standard FP-Growth
// "conditional pattern base" for item.
func conditionalPatternBase(nodes []*fpNode) (paths [][]string, counts []int) {
	for _, node := range nodes {
		var path []string
		for p := node.parent; p != nil && p.item != ""; p = p.parent {
			path = append([]string{p.item}, path...)
		}
		if len(path) > 0 {
			paths = append(paths, path)
			counts = append(counts, node.count)
		}
	}
	return paths, counts
}

// mineFPTree recursively extracts every frequent itemset from tree,
// prefixed by suffix (the items already fixed by the caller's recursion).
// Each header-table item present in tree is itself a frequent itemset
// (suffix + item) as long as its support in this tree meets minSupport;
// growing beyond maxItemsetSize is simply not attempted, bounding
// worst-case output size on dense datasets.
func mineFPTree(tree *fpTree, suffix []string, minSupport, maxItemsetSize int, results *[]frequentItemset) {
	items := make([]string, 0, len(tree.headerTable))
	for item := range tree.headerTable {
		items = append(items, item)
	}
	sort.Strings(items) // deterministic iteration order -> deterministic results order

	for _, item := range items {
		nodes := tree.headerTable[item]
		support := totalSupport(nodes)
		if support < minSupport {
			continue
		}
		itemset := make([]string, 0, len(suffix)+1)
		itemset = append(itemset, suffix...)
		itemset = append(itemset, item)
		*results = append(*results, frequentItemset{Items: itemset, Support: support})

		if len(itemset) >= maxItemsetSize {
			continue
		}
		paths, counts := conditionalPatternBase(nodes)
		if len(paths) == 0 {
			continue
		}
		condTree := newFPTree()
		for i, path := range paths {
			condTree.insert(path, counts[i])
		}
		if len(condTree.headerTable) > 0 {
			mineFPTree(condTree, itemset, minSupport, maxItemsetSize, results)
		}
	}
}
