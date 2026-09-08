package theme_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/theme"
)

func TestBlockType_IsFunctional(t *testing.T) {
	assert.True(t, theme.BlockTypeCheckoutForm.IsFunctional())
	assert.True(t, theme.BlockTypeCartSummary.IsFunctional())
	assert.True(t, theme.BlockTypeProductDetails.IsFunctional())
	assert.True(t, theme.BlockTypeOrderStatus.IsFunctional())
	assert.False(t, theme.BlockTypeHero.IsFunctional())
	assert.False(t, theme.BlockTypeRichText.IsFunctional())
	assert.False(t, theme.BlockTypeProductGrid.IsFunctional())
	assert.False(t, theme.BlockTypeCustomHTML.IsFunctional())
}

func TestPageSlug_RequiredBlockType(t *testing.T) {
	tests := []struct {
		slug     theme.PageSlug
		wantType theme.BlockType
		wantOK   bool
	}{
		{theme.PageSlugHome, "", false},
		{theme.PageSlugProducts, "", false},
		{theme.PageSlugProduct, theme.BlockTypeProductDetails, true},
		{theme.PageSlugCart, theme.BlockTypeCartSummary, true},
		{theme.PageSlugCheckout, theme.BlockTypeCheckoutForm, true},
		{theme.PageSlugOrderConfirmed, theme.BlockTypeOrderStatus, true},
		{theme.PageSlugOrderStatusLookup, theme.BlockTypeOrderStatus, true},
	}
	for _, tc := range tests {
		gotType, gotOK := tc.slug.RequiredBlockType()
		assert.Equal(t, tc.wantOK, gotOK, "slug %q", tc.slug)
		assert.Equal(t, tc.wantType, gotType, "slug %q", tc.slug)
	}
}

func TestThemePage_HasRequiredBlock_trueForPagesWithNoRequirement(t *testing.T) {
	page := theme.ThemePage{Slug: theme.PageSlugHome, Blocks: nil}
	assert.True(t, page.HasRequiredBlock())
}

func TestThemePage_HasRequiredBlock_falseWhenMissing(t *testing.T) {
	page := theme.ThemePage{Slug: theme.PageSlugCheckout, Blocks: []theme.Block{
		{ID: "b1", Type: theme.BlockTypeHero},
	}}
	assert.False(t, page.HasRequiredBlock())
}

func TestThemePage_HasRequiredBlock_trueWhenPresent(t *testing.T) {
	page := theme.ThemePage{Slug: theme.PageSlugCheckout, Blocks: []theme.Block{
		{ID: "b1", Type: theme.BlockTypeHero},
		{ID: "b2", Type: theme.BlockTypeCheckoutForm},
	}}
	assert.True(t, page.HasRequiredBlock())
}
