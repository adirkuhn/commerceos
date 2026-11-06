package channels

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func oneSKU(product, variant, code string) relinkSKU {
	return relinkSKU{ProductID: product, ProductName: product, VariantID: variant, SKUCode: code, OnlySellableSKU: true}
}

func oneOfSeveral(product, variant, code string) relinkSKU {
	return relinkSKU{ProductID: product, ProductName: product, VariantID: variant, SKUCode: code}
}

func wooSimple(id, sku string) relinkWooItem {
	return relinkWooItem{ID: id, Name: "woo " + id, SKU: sku, Type: "simple"}
}

func wooParent(id string) relinkWooItem {
	return relinkWooItem{ID: id, Name: "woo " + id, Type: wooTypeVariable}
}

func wooVariation(id, parent, sku string) relinkWooItem {
	return relinkWooItem{ID: id, ParentID: parent, SKU: sku}
}

func TestPlanWooRelink_SimpleProductGetsAProductLevelRow(t *testing.T) {
	plan := planWooRelink([]relinkSKU{oneSKU("p1", "v1", "A")}, []relinkWooItem{wooSimple("10", "A")}, nil)

	require.Len(t, plan.ToLink, 1)
	assert.Equal(t, RelinkLink{ProductID: "p1", ProductName: "p1", ExternalID: "10", SKUCode: "A"}, plan.ToLink[0])
	assert.Empty(t, plan.Conflicts)
	assert.Empty(t, plan.MissingInWoo)
	assert.Empty(t, plan.MissingInCommerceOS)
}

func TestPlanWooRelink_SeparateVariantsEachGetTheirOwnRow(t *testing.T) {
	plan := planWooRelink(
		[]relinkSKU{oneOfSeveral("p1", "v1", "A"), oneOfSeveral("p1", "v2", "B")},
		[]relinkWooItem{wooSimple("10", "A"), wooSimple("11", "B")}, nil)

	v1, v2 := "v1", "v2"
	assert.Equal(t, []RelinkLink{
		{ProductID: "p1", ProductName: "p1", VariantID: &v1, ExternalID: "10", SKUCode: "A"},
		{ProductID: "p1", ProductName: "p1", VariantID: &v2, ExternalID: "11", SKUCode: "B"},
	}, plan.ToLink)
}

func TestPlanWooRelink_VariableProductGetsParentAndVariationRows(t *testing.T) {
	plan := planWooRelink(
		[]relinkSKU{oneOfSeveral("p1", "v1", "A"), oneOfSeveral("p1", "v2", "B")},
		[]relinkWooItem{wooParent("20"), wooVariation("21", "20", "A"), wooVariation("22", "20", "B")}, nil)

	v1, v2 := "v1", "v2"
	assert.Equal(t, []RelinkLink{
		{ProductID: "p1", ProductName: "p1", ExternalID: "20"},
		{ProductID: "p1", ProductName: "p1", VariantID: &v1, ExternalID: "21", SKUCode: "A", IsVariation: true},
		{ProductID: "p1", ProductName: "p1", VariantID: &v2, ExternalID: "22", SKUCode: "B", IsVariation: true},
	}, plan.ToLink)
	assert.Empty(t, plan.MissingInCommerceOS, "a parent with no SKU of its own is not an unmatched product")
}

func TestPlanWooRelink_ComparesSKUsIgnoringSurroundingSpaces(t *testing.T) {
	plan := planWooRelink([]relinkSKU{oneSKU("p1", "v1", "A")}, []relinkWooItem{wooSimple("10", " A ")}, nil)

	require.Len(t, plan.ToLink, 1)
}

func TestPlanWooRelink_ListsWhatMatchesNothing(t *testing.T) {
	plan := planWooRelink(
		[]relinkSKU{oneSKU("p1", "v1", "A")},
		[]relinkWooItem{wooSimple("10", "Z"), wooSimple("11", ""), wooParent("20"), wooVariation("21", "20", "Y")}, nil)

	assert.Empty(t, plan.ToLink)
	assert.Equal(t, []RelinkItem{{ID: "p1", Name: "p1", SKUCode: "A"}}, plan.MissingInWoo)
	assert.Equal(t, []RelinkItem{
		{ID: "10", Name: "woo 10", SKUCode: "Z"},
		{ID: "11", Name: "woo 11"},
		{ID: "21", Name: "woo 20", SKUCode: "Y"},
	}, plan.MissingInCommerceOS)
}

func TestPlanWooRelink_SkipsWhatIsAlreadyLinked(t *testing.T) {
	existing := []channelsDomain.ProductChannelListing{{ProductID: "p1", ExternalID: "10"}}

	plan := planWooRelink([]relinkSKU{oneSKU("p1", "v1", "A")}, []relinkWooItem{wooSimple("10", "A")}, existing)

	assert.Empty(t, plan.ToLink)
	assert.Empty(t, plan.Conflicts)
	assert.Equal(t, 1, plan.AlreadyLinked)
}

func TestPlanWooRelink_AddsOnlyTheMissingRowsOfAVariableProduct(t *testing.T) {
	v1, v2 := "v1", "v2"
	existing := []channelsDomain.ProductChannelListing{
		{ProductID: "p1", ExternalID: "20"},
		{ProductID: "p1", VariantID: &v1, ExternalID: "21", IsVariation: true},
	}

	plan := planWooRelink(
		[]relinkSKU{oneOfSeveral("p1", "v1", "A"), oneOfSeveral("p1", "v2", "B")},
		[]relinkWooItem{wooParent("20"), wooVariation("21", "20", "A"), wooVariation("22", "20", "B")}, existing)

	assert.Equal(t, []RelinkLink{{ProductID: "p1", ProductName: "p1", VariantID: &v2, ExternalID: "22", SKUCode: "B", IsVariation: true}}, plan.ToLink)
	assert.Equal(t, 2, plan.AlreadyLinked)
}

func TestPlanWooRelink_MissingInWooLeavesOutLinkedSKUs(t *testing.T) {
	existing := []channelsDomain.ProductChannelListing{{ProductID: "p1", ExternalID: "10"}}

	plan := planWooRelink([]relinkSKU{oneSKU("p1", "v1", "A")}, nil, existing)

	assert.Empty(t, plan.MissingInWoo)
}

func TestPlanWooRelink_ConflictsAreReportedNotGuessed(t *testing.T) {
	cases := map[string]struct {
		skus     []relinkSKU
		woo      []relinkWooItem
		existing []channelsDomain.ProductChannelListing
		reason   string
	}{
		"same SKU twice in Woo": {
			skus:   []relinkSKU{oneSKU("p1", "v1", "A")},
			woo:    []relinkWooItem{wooSimple("10", "A"), wooSimple("11", "A")},
			reason: RelinkConflictDuplicateWooSKU,
		},
		"same SKU on several CommerceOS variants": {
			skus:   []relinkSKU{oneSKU("p1", "v1", "A"), oneSKU("p2", "v2", "A")},
			woo:    []relinkWooItem{wooSimple("10", "A")},
			reason: RelinkConflictDuplicateSKU,
		},
		"Woo id linked to another product": {
			skus:     []relinkSKU{oneSKU("p1", "v1", "A")},
			woo:      []relinkWooItem{wooSimple("10", "A")},
			existing: []channelsDomain.ProductChannelListing{{ProductID: "p9", ExternalID: "10"}},
			reason:   RelinkConflictExternalIDTaken,
		},
		"product already linked to another Woo id": {
			skus:     []relinkSKU{oneSKU("p1", "v1", "A")},
			woo:      []relinkWooItem{wooSimple("10", "A")},
			existing: []channelsDomain.ProductChannelListing{{ProductID: "p1", ExternalID: "99"}},
			reason:   RelinkConflictProductLinkedElsewhere,
		},
		"variations of one parent belong to different products": {
			skus:   []relinkSKU{oneSKU("p1", "v1", "A"), oneSKU("p2", "v2", "B")},
			woo:    []relinkWooItem{wooParent("20"), wooVariation("21", "20", "A"), wooVariation("22", "20", "B")},
			reason: RelinkConflictParentSpansProducts,
		},
		"product matches two Woo shapes": {
			skus:   []relinkSKU{oneOfSeveral("p1", "v1", "A"), oneOfSeveral("p1", "v2", "B")},
			woo:    []relinkWooItem{wooSimple("10", "A"), wooParent("20"), wooVariation("21", "20", "B")},
			reason: RelinkConflictProductSpansShapes,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			plan := planWooRelink(tc.skus, tc.woo, tc.existing)

			assert.Empty(t, plan.ToLink)
			require.NotEmpty(t, plan.Conflicts)
			assert.Equal(t, tc.reason, plan.Conflicts[0].Reason)
		})
	}
}

func TestPlanWooRelink_ConflictOnAParentDropsItsVariationsToo(t *testing.T) {
	existing := []channelsDomain.ProductChannelListing{{ProductID: "p9", ExternalID: "20"}}

	plan := planWooRelink(
		[]relinkSKU{oneOfSeveral("p1", "v1", "A")},
		[]relinkWooItem{wooParent("20"), wooVariation("21", "20", "A")}, existing)

	assert.Empty(t, plan.ToLink)
	require.Len(t, plan.Conflicts, 1)
	assert.Equal(t, RelinkConflictExternalIDTaken, plan.Conflicts[0].Reason)
}

func TestPlanWooRelink_AConflictedSKUIsNotAlsoMissing(t *testing.T) {
	plan := planWooRelink([]relinkSKU{oneSKU("p1", "v1", "A")}, []relinkWooItem{wooSimple("10", "A"), wooSimple("11", "A")}, nil)

	assert.Empty(t, plan.MissingInWoo)
	assert.Empty(t, plan.MissingInCommerceOS)
}
