package pms_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func skus(ids ...string) []pmsDomain.SKU {
	out := make([]pmsDomain.SKU, len(ids))
	for i, id := range ids {
		out[i] = pmsDomain.SKU{ID: id}
	}
	return out
}

func TestNewProductMerge_validMapping(t *testing.T) {
	m, err := pmsDomain.NewProductMerge("dup", "orig", skus("d1", "d2"), skus("o1", "o2"), []pmsDomain.SKUMergePair{
		{SourceSKUID: "d1", TargetSKUID: "o1"}, {SourceSKUID: "d2", TargetSKUID: "o1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "dup", m.SourceProductID)
	assert.Equal(t, "orig", m.TargetProductID)
	assert.Len(t, m.Pairs, 2, "two duplicate SKUs may fold into the same existing SKU")
}

func TestNewProductMerge_rejects(t *testing.T) {
	cases := map[string]struct {
		source, target string
		pairs          []pmsDomain.SKUMergePair
		want           error
	}{
		"same product":        {"p", "p", []pmsDomain.SKUMergePair{{SourceSKUID: "d1", TargetSKUID: "o1"}}, pmsDomain.ErrMergeIntoSelf},
		"unmapped source sku": {"dup", "orig", []pmsDomain.SKUMergePair{{SourceSKUID: "d1", TargetSKUID: "o1"}}, pmsDomain.ErrMergeMappingIncomplete},
		"sku mapped twice": {"dup", "orig", []pmsDomain.SKUMergePair{
			{SourceSKUID: "d1", TargetSKUID: "o1"}, {SourceSKUID: "d1", TargetSKUID: "o2"}, {SourceSKUID: "d2", TargetSKUID: "o1"},
		}, pmsDomain.ErrMergeSKUMappedTwice},
		"source sku of another product": {"dup", "orig", []pmsDomain.SKUMergePair{
			{SourceSKUID: "d1", TargetSKUID: "o1"}, {SourceSKUID: "d2", TargetSKUID: "o1"}, {SourceSKUID: "x9", TargetSKUID: "o1"},
		}, pmsDomain.ErrMergeSKUNotInProduct},
		"target sku of another product": {"dup", "orig", []pmsDomain.SKUMergePair{
			{SourceSKUID: "d1", TargetSKUID: "o1"}, {SourceSKUID: "d2", TargetSKUID: "x9"},
		}, pmsDomain.ErrMergeSKUNotInProduct},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := pmsDomain.NewProductMerge(c.source, c.target, skus("d1", "d2"), skus("o1", "o2"), c.pairs)
			require.ErrorIs(t, err, c.want)
			require.ErrorIs(t, err, pmsDomain.ErrInvalidProductMerge)
		})
	}
}

func TestDefaultSKUMergePairs(t *testing.T) {
	pairs, ok := pmsDomain.DefaultSKUMergePairs(skus("d1"), skus("o1"))
	require.True(t, ok)
	assert.Equal(t, []pmsDomain.SKUMergePair{{SourceSKUID: "d1", TargetSKUID: "o1"}}, pairs)

	_, ok = pmsDomain.DefaultSKUMergePairs(skus("d1", "d2"), skus("o1"))
	assert.False(t, ok, "more than one SKU on either side needs an explicit mapping")
}

func TestBlendMovingAverageCost(t *testing.T) {
	c := func(v int64) *int64 { return &v }
	assert.Equal(t, c(150), pmsDomain.BlendMovingAverageCost(10, c(100), 10, c(200)))
	assert.Equal(t, c(133), pmsDomain.BlendMovingAverageCost(20, c(100), 10, c(200)), "rounded half-up: 133.33")
	assert.Equal(t, c(200), pmsDomain.BlendMovingAverageCost(10, nil, 10, c(200)), "an unknown side takes the known cost")
	assert.Equal(t, c(100), pmsDomain.BlendMovingAverageCost(10, c(100), 5, nil))
	assert.Nil(t, pmsDomain.BlendMovingAverageCost(10, nil, 5, nil))
	assert.Equal(t, c(100), pmsDomain.BlendMovingAverageCost(0, c(100), 0, c(200)), "no stock on either side keeps the target's cost")
	assert.Equal(t, c(200), pmsDomain.BlendMovingAverageCost(0, c(100), 5, c(200)), "only the duplicate has stock")
	assert.Equal(t, c(100), pmsDomain.BlendMovingAverageCost(-3, c(100), 0, c(200)), "negative on-hand counts as none")
}
