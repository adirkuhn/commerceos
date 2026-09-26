package pms

import (
	"errors"
	"fmt"
	"time"
)

// A product merge folds a duplicate product (typically booked in as "new"
// when it already existed) into the existing one: each duplicate SKU is
// matched to an existing SKU, every reference to it (stock, batches, cost,
// orders, receiving, aliases, ...) moves to that SKU, and the duplicate is
// deleted. The existing product's own data -- name, description, category,
// prices -- is never changed.

// SKUMergePair matches one duplicate SKU to the existing SKU it folds into.
// Several duplicate SKUs may fold into the same existing SKU.
type SKUMergePair struct {
	SourceSKUID string
	TargetSKUID string
}

// ProductMerge is a validated merge plan.
type ProductMerge struct {
	SourceProductID string
	TargetProductID string
	Pairs           []SKUMergePair
}

var (
	// ErrInvalidProductMerge is what every merge-plan validation error wraps.
	ErrInvalidProductMerge    = errors.New("invalid product merge")
	ErrMergeIntoSelf          = fmt.Errorf("%w: a product cannot be merged into itself", ErrInvalidProductMerge)
	ErrMergeMappingIncomplete = fmt.Errorf("%w: every SKU of the duplicate must be matched to an existing SKU", ErrInvalidProductMerge)
	ErrMergeSKUMappedTwice    = fmt.Errorf("%w: a duplicate SKU is matched more than once", ErrInvalidProductMerge)
	ErrMergeSKUNotInProduct   = fmt.Errorf("%w: a matched SKU does not belong to its product", ErrInvalidProductMerge)
	// ErrProductMergeForbidden: merging is irreversible -- tenant owners
	// and admins only.
	ErrProductMergeForbidden = errors.New("only an owner or admin can merge products")
)

// NewProductMerge validates a merge plan against both products' SKUs: the
// products differ, every duplicate SKU is matched exactly once, and every
// pair's SKUs belong to their respective product.
func NewProductMerge(sourceProductID, targetProductID string, sourceSKUs, targetSKUs []SKU, pairs []SKUMergePair) (*ProductMerge, error) {
	if sourceProductID == targetProductID {
		return nil, ErrMergeIntoSelf
	}
	inSource := idSet(sourceSKUs)
	inTarget := idSet(targetSKUs)
	matched := make(map[string]bool, len(pairs))
	for _, p := range pairs {
		if !inSource[p.SourceSKUID] || !inTarget[p.TargetSKUID] {
			return nil, ErrMergeSKUNotInProduct
		}
		if matched[p.SourceSKUID] {
			return nil, ErrMergeSKUMappedTwice
		}
		matched[p.SourceSKUID] = true
	}
	if len(matched) != len(inSource) {
		return nil, ErrMergeMappingIncomplete
	}
	return &ProductMerge{SourceProductID: sourceProductID, TargetProductID: targetProductID, Pairs: pairs}, nil
}

// DefaultSKUMergePairs is the obvious mapping when both products have
// exactly one SKU; any other shape needs the user to match SKUs.
func DefaultSKUMergePairs(sourceSKUs, targetSKUs []SKU) ([]SKUMergePair, bool) {
	if len(sourceSKUs) != 1 || len(targetSKUs) != 1 {
		return nil, false
	}
	return []SKUMergePair{{SourceSKUID: sourceSKUs[0].ID, TargetSKUID: targetSKUs[0].ID}}, true
}

// BlendMovingAverageCost is the existing SKU's moving-average cost after a
// merge folds the duplicate's on-hand stock into it:
//
//	(target_qty*target_cost + source_qty*source_cost) / (target_qty + source_qty)
//
// A nil cost is unknown (SKUCostPriceSourceUnknown): that side takes the
// other side's cost rather than counting as 0, same policy as a receipt
// into an unknown-cost SKU. Negative on-hand counts as none. With no stock
// on either side the target's cost stands (or the source's if the target
// has none). Rounded half-up, no floats.
func BlendMovingAverageCost(targetQty int64, targetCost *int64, sourceQty int64, sourceCost *int64) *int64 {
	targetQty, sourceQty = max(targetQty, 0), max(sourceQty, 0)
	switch {
	case targetCost == nil && sourceCost == nil:
		return nil
	case targetCost == nil:
		return sourceCost
	case sourceCost == nil:
		return targetCost
	}
	total := targetQty + sourceQty
	if total == 0 {
		return targetCost
	}
	blended := (targetQty*(*targetCost) + sourceQty*(*sourceCost) + total/2) / total
	return &blended
}

// MergedSKU records one duplicate SKU for the merge audit trail -- the
// duplicate row is deleted, so its identifying codes are kept here.
type MergedSKU struct {
	SourceSKUID   string
	SourceSKUCode string
	SourceBarcode string
	TargetSKUID   string
}

// ProductMergeRecord is the audit row of a completed merge.
type ProductMergeRecord struct {
	ID                string
	TenantID          string
	SourceProductID   string
	SourceProductName string
	TargetProductID   string
	SKUs              []MergedSKU
	MergedBy          string
	MergedAt          time.Time
}

func idSet(skus []SKU) map[string]bool {
	set := make(map[string]bool, len(skus))
	for _, s := range skus {
		set[s.ID] = true
	}
	return set
}
