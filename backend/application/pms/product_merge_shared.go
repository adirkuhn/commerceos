package pms

import (
	"context"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// Shared by MergeProductsUseCase and PreviewProductMergeUseCase, so the
// preview shows exactly what a merge will do.

func listProductSKUs(ctx context.Context, variants mergeVariantLister, skus mergeSKULister, productID string) ([]pmsDomain.SKU, error) {
	vs, err := variants.ListByProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	var out []pmsDomain.SKU
	for _, v := range vs {
		s, err := skus.ListByVariant(ctx, v.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, s...)
	}
	return out, nil
}

func sumOnHand(ctx context.Context, onHand mergeOnHand, skuID string) (int64, error) {
	batches, err := onHand.Execute(ctx, skuID)
	if err != nil {
		return 0, err
	}
	var qty int64
	for _, b := range batches {
		qty += b.Quantity
	}
	return qty, nil
}

// mergeStep is one pair folded into its target: the duplicate's on-hand
// joining, and the target's quantity and cost right after.
type mergeStep struct {
	pair        pmsDomain.SKUMergePair
	sourceQty   int64
	qtyAfter    int64
	costBefore  *int64
	costAfter   *int64
	costChanged bool
}

// foldMergePairs applies pairs in order (several duplicates may fold into
// one target): the duplicate's stock joins at what it actually cost,
// blending the target's inventory cost (moving average, see
// pmsDomain.SKU.ActualCostMinorUnits). Negative on-hand counts as none.
func foldMergePairs(pairs []pmsDomain.SKUMergePair, bySKUID map[string]pmsDomain.SKU, onHand map[string]int64) []mergeStep {
	qty := make(map[string]int64)
	cost := make(map[string]*int64)
	steps := make([]mergeStep, len(pairs))
	for i, p := range pairs {
		target := bySKUID[p.TargetSKUID]
		if _, seen := qty[p.TargetSKUID]; !seen {
			qty[p.TargetSKUID] = max(onHand[p.TargetSKUID], 0)
			cost[p.TargetSKUID] = target.ActualCostMinorUnits()
		}
		sourceQty := max(onHand[p.SourceSKUID], 0)
		before := cost[p.TargetSKUID]
		after := pmsDomain.BlendMovingAverageCost(qty[p.TargetSKUID], before, sourceQty, bySKUID[p.SourceSKUID].ActualCostMinorUnits())
		qty[p.TargetSKUID] += sourceQty
		cost[p.TargetSKUID] = after
		steps[i] = mergeStep{pair: p, sourceQty: sourceQty, qtyAfter: qty[p.TargetSKUID], costBefore: before, costAfter: after, costChanged: after != nil}
	}
	return steps
}
