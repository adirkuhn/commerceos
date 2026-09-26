package pms_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func newPreviewUseCase(t *testing.T, h *mergeHarness) *apppms.PreviewProductMergeUseCase {
	return apppms.NewPreviewProductMergeUseCase(newMockGormDBN(t, 2), h.catalog, h.catalog, h.catalog, h.onHand)
}

func TestPreviewProductMerge_singleSKUsAreMatchedWithProjectedCost(t *testing.T) {
	h := newMergeHarness(t)

	p, err := newPreviewUseCase(t, h).Execute(mergeCtx(), apppms.PreviewProductMergeInput{SourceProductID: "dup", TargetProductID: "orig"})

	require.NoError(t, err)
	assert.Equal(t, "Guarana 350ml (new)", p.Source.Name)
	assert.Equal(t, "Guarana Antarctica 350ml", p.Target.Name)
	require.Len(t, p.Source.SKUs, 1)
	assert.Equal(t, apppms.MergePreviewSKU{ID: "d1", SKUCode: "GUA-NEW", Barcode: "8710001", OnHand: 4, CostPriceMinorUnits: cost(200)}, p.Source.SKUs[0])
	assert.Equal(t, []pmsDomain.SKUMergePair{{SourceSKUID: "d1", TargetSKUID: "o1"}}, p.Pairs)
	require.Len(t, p.Targets, 1)
	assert.Equal(t, apppms.MergePreviewTarget{SKUID: "o1", OnHandAfter: 10, CostBefore: cost(100), CostAfter: cost(140)}, p.Targets[0])
}

func TestPreviewProductMerge_suggestsPairsByMatchingBarcode(t *testing.T) {
	h := newMergeHarness(t)
	h.catalog.skus["vd"] = append(h.catalog.skus["vd"], pmsDomain.SKU{ID: "d2", VariantID: "vd", SKUCode: "X", Barcode: "5550002"})
	h.catalog.skus["vo"] = append(h.catalog.skus["vo"], pmsDomain.SKU{ID: "o2", VariantID: "vo", SKUCode: "Y", Barcode: "5550002"})

	p, err := newPreviewUseCase(t, h).Execute(mergeCtx(), apppms.PreviewProductMergeInput{SourceProductID: "dup", TargetProductID: "orig"})

	require.NoError(t, err)
	assert.Equal(t, []pmsDomain.SKUMergePair{{SourceSKUID: "d2", TargetSKUID: "o2"}}, p.Pairs, "only a sure match is suggested; d1 is left for the user")
	assert.Empty(t, p.Targets, "no projection until every SKU is matched")
}

func TestPreviewProductMerge_projectsAGivenMapping(t *testing.T) {
	h := newMergeHarness(t)
	h.catalog.skus["vd"] = append(h.catalog.skus["vd"], pmsDomain.SKU{ID: "d2", VariantID: "vd", CostPriceMinorUnits: cost(300)})
	h.onHand["d2"] = 10

	p, err := newPreviewUseCase(t, h).Execute(mergeCtx(), apppms.PreviewProductMergeInput{SourceProductID: "dup", TargetProductID: "orig",
		Pairs: []pmsDomain.SKUMergePair{{SourceSKUID: "d1", TargetSKUID: "o1"}, {SourceSKUID: "d2", TargetSKUID: "o1"}}})

	require.NoError(t, err)
	require.Len(t, p.Targets, 1)
	// (6*100 + 4*200) / 10 = 140, then (10*140 + 10*300) / 20 = 220
	assert.Equal(t, apppms.MergePreviewTarget{SKUID: "o1", OnHandAfter: 20, CostBefore: cost(100), CostAfter: cost(220)}, p.Targets[0])
}

func TestPreviewProductMerge_rejectsAnInvalidMapping(t *testing.T) {
	h := newMergeHarness(t)

	_, err := newPreviewUseCase(t, h).Execute(mergeCtx(), apppms.PreviewProductMergeInput{SourceProductID: "dup", TargetProductID: "orig",
		Pairs: []pmsDomain.SKUMergePair{{SourceSKUID: "d1", TargetSKUID: "nope"}}})

	require.ErrorIs(t, err, pmsDomain.ErrMergeSKUNotInProduct)
}
