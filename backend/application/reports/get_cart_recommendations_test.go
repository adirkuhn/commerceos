package reports_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubSKUCodeLookup struct {
	byCode map[string]*pmsDomain.SKU
}

func (s *stubSKUCodeLookup) FindByCode(_ context.Context, code string) (*pmsDomain.SKU, error) {
	sku, ok := s.byCode[code]
	if !ok {
		return nil, pmsDomain.ErrSKUNotFound
	}
	return sku, nil
}

func TestGetCartRecommendations_ranksByQuantityWeightedSupport(t *testing.T) {
	basket := &stubBasketLister{out: []omsDomain.OrderSKUs{
		{OrderID: "o1", SKUIDs: []string{"sku-a", "sku-x"}},
		{OrderID: "o2", SKUIDs: []string{"sku-a", "sku-x"}},
		{OrderID: "o3", SKUIDs: []string{"sku-b", "sku-y"}},
		{OrderID: "o4", SKUIDs: []string{"sku-b", "sku-y"}},
		{OrderID: "o5", SKUIDs: []string{"sku-b", "sku-y"}},
		// Unrelated orders -- inflate TotalOrders without touching either
		// candidate's raw support, so SKU-X's confidence (2/7) lands below
		// the bundle threshold while SKU-Y's (3/7) stays above it.
		{OrderID: "o6", SKUIDs: []string{"sku-p", "sku-q"}},
		{OrderID: "o7", SKUIDs: []string{"sku-p", "sku-q"}},
	}}
	skuCodeLookup := &stubSKUCodeLookup{byCode: map[string]*pmsDomain.SKU{
		"SKU-A": {ID: "sku-a"},
		"SKU-B": {ID: "sku-b"},
	}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{
		"sku-x": {ID: "sku-x", SKUCode: "SKU-X", VariantID: "variant-x"},
		"sku-y": {ID: "sku-y", SKUCode: "SKU-Y", VariantID: "variant-y"},
	}}
	variantLookup := &stubVariantLookupForSlowMoving{byID: map[string]*pmsDomain.Variant{
		"variant-x": {ID: "variant-x", ProductID: "product-x"},
		"variant-y": {ID: "variant-y", ProductID: "product-y"},
	}}
	productLookup := &stubProductLookupForSlowMoving{byID: map[string]*pmsDomain.Product{
		"product-x": {ID: "product-x", Name: "Widget X"},
		"product-y": {ID: "product-y", Name: "Widget Y"},
	}}
	uc := appreports.NewGetCartRecommendationsUseCase(newMockGormDB(t), basket, &stubSettingsRepo{}, skuLookup, skuCodeLookup, variantLookup, productLookup)

	// Cart has 1x SKU-A (support 2) and 5x SKU-B (support 3) -- SKU-B's
	// weaker raw support still wins the ranking because of the quantity
	// weighting (5*3=15 vs 1*2=2), while each candidate's reported
	// Confidence stays its own honest raw-support percentage.
	report, err := uc.Execute(context.Background(), []appreports.CartLine{
		{SKUCode: "SKU-A", Quantity: 1},
		{SKUCode: "SKU-B", Quantity: 5},
	})

	require.NoError(t, err)
	require.Equal(t, int64(7), report.TotalOrders)
	require.Len(t, report.Recommendations, 2)
	require.Equal(t, "SKU-Y", report.Recommendations[0].SKUCode)
	require.InDelta(t, 3.0/7.0, report.Recommendations[0].Confidence, 0.0001)
	require.True(t, report.Recommendations[0].IsBundleCandidate)
	require.Equal(t, "SKU-X", report.Recommendations[1].SKUCode)
	require.InDelta(t, 2.0/7.0, report.Recommendations[1].Confidence, 0.0001)
	require.False(t, report.Recommendations[1].IsBundleCandidate)
}

func TestGetCartRecommendations_excludesSKUsAlreadyInCart(t *testing.T) {
	basket := &stubBasketLister{out: []omsDomain.OrderSKUs{
		{OrderID: "o1", SKUIDs: []string{"sku-a", "sku-b"}},
		{OrderID: "o2", SKUIDs: []string{"sku-a", "sku-b"}},
	}}
	skuCodeLookup := &stubSKUCodeLookup{byCode: map[string]*pmsDomain.SKU{
		"SKU-A": {ID: "sku-a"},
		"SKU-B": {ID: "sku-b"},
	}}
	uc := appreports.NewGetCartRecommendationsUseCase(newMockGormDB(t), basket, &stubSettingsRepo{}, &stubSKUCostPriceLookup{}, skuCodeLookup, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	report, err := uc.Execute(context.Background(), []appreports.CartLine{
		{SKUCode: "SKU-A", Quantity: 1},
		{SKUCode: "SKU-B", Quantity: 1},
	})

	require.NoError(t, err)
	require.Empty(t, report.Recommendations)
}

func TestGetCartRecommendations_dropsCandidateWithNoLiveSKU(t *testing.T) {
	basket := &stubBasketLister{out: []omsDomain.OrderSKUs{
		{OrderID: "o1", SKUIDs: []string{"sku-a", "sku-deleted"}},
		{OrderID: "o2", SKUIDs: []string{"sku-a", "sku-deleted"}},
	}}
	skuCodeLookup := &stubSKUCodeLookup{byCode: map[string]*pmsDomain.SKU{"SKU-A": {ID: "sku-a"}}}
	uc := appreports.NewGetCartRecommendationsUseCase(newMockGormDB(t), basket, &stubSettingsRepo{}, &stubSKUCostPriceLookup{}, skuCodeLookup, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	report, err := uc.Execute(context.Background(), []appreports.CartLine{{SKUCode: "SKU-A", Quantity: 1}})

	require.NoError(t, err)
	require.Empty(t, report.Recommendations)
}

func TestGetCartRecommendations_skipsUnknownCartSKUCode(t *testing.T) {
	basket := &stubBasketLister{out: []omsDomain.OrderSKUs{
		{OrderID: "o1", SKUIDs: []string{"sku-a", "sku-b"}},
		{OrderID: "o2", SKUIDs: []string{"sku-a", "sku-b"}},
	}}
	skuCodeLookup := &stubSKUCodeLookup{byCode: map[string]*pmsDomain.SKU{"SKU-A": {ID: "sku-a"}}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{"sku-b": {ID: "sku-b", SKUCode: "SKU-B"}}}
	uc := appreports.NewGetCartRecommendationsUseCase(newMockGormDB(t), basket, &stubSettingsRepo{}, skuLookup, skuCodeLookup, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	// SKU-UNKNOWN isn't in skuCodeLookup -- must be silently skipped, not
	// fail the whole request; SKU-A alone still yields SKU-B.
	report, err := uc.Execute(context.Background(), []appreports.CartLine{
		{SKUCode: "SKU-A", Quantity: 1},
		{SKUCode: "SKU-UNKNOWN", Quantity: 1},
	})

	require.NoError(t, err)
	require.Len(t, report.Recommendations, 1)
	require.Equal(t, "SKU-B", report.Recommendations[0].SKUCode)
}

func TestGetCartRecommendations_rejectsEmptyCart(t *testing.T) {
	uc := appreports.NewGetCartRecommendationsUseCase(newMockGormDB(t), &stubBasketLister{}, &stubSettingsRepo{}, &stubSKUCostPriceLookup{}, &stubSKUCodeLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	_, err := uc.Execute(context.Background(), nil)

	require.Error(t, err)
}

func TestGetCartRecommendations_propagatesBasketRepoError(t *testing.T) {
	uc := appreports.NewGetCartRecommendationsUseCase(newMockGormDB(t), &stubBasketLister{err: context.DeadlineExceeded}, &stubSettingsRepo{}, &stubSKUCostPriceLookup{}, &stubSKUCodeLookup{byCode: map[string]*pmsDomain.SKU{"SKU-A": {ID: "sku-a"}}}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	_, err := uc.Execute(context.Background(), []appreports.CartLine{{SKUCode: "SKU-A", Quantity: 1}})

	require.Error(t, err)
}
