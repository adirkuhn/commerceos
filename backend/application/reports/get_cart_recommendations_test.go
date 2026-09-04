package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
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
		// SKU-A appears in 7 orders total, only 2 of which also have
		// SKU-X -- confidence(SKU-X | SKU-A) = 2/7 ≈ 0.286, below the
		// default 0.30 bundle threshold.
		{OrderID: "o1", SKUIDs: []string{"sku-a", "sku-x"}},
		{OrderID: "o2", SKUIDs: []string{"sku-a", "sku-x"}},
		{OrderID: "o3", SKUIDs: []string{"sku-a"}},
		{OrderID: "o4", SKUIDs: []string{"sku-a"}},
		{OrderID: "o5", SKUIDs: []string{"sku-a"}},
		{OrderID: "o6", SKUIDs: []string{"sku-a"}},
		{OrderID: "o7", SKUIDs: []string{"sku-a"}},
		// SKU-B appears in 3 orders, all 3 of which also have SKU-Y --
		// confidence(SKU-Y | SKU-B) = 3/3 = 1.0, comfortably above it.
		{OrderID: "o8", SKUIDs: []string{"sku-b", "sku-y"}},
		{OrderID: "o9", SKUIDs: []string{"sku-b", "sku-y"}},
		{OrderID: "o10", SKUIDs: []string{"sku-b", "sku-y"}},
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

	// Cart has 1x SKU-A (raw pair support 2) and 5x SKU-B (raw pair
	// support 3) -- SKU-B's weaker raw support still wins the ranking
	// because of the quantity weighting (5*3=15 vs 1*2=2), while each
	// candidate's reported Confidence stays its own honest
	// support(trigger,candidate)/support(trigger) fraction, independent
	// of that weighting and of the cart's other lines.
	report, err := uc.Execute(context.Background(), []appreports.CartLine{
		{SKUCode: "SKU-A", Quantity: 1},
		{SKUCode: "SKU-B", Quantity: 5},
	})

	require.NoError(t, err)
	require.Equal(t, int64(10), report.TotalOrders)
	require.Len(t, report.Recommendations, 2)
	require.Equal(t, "SKU-Y", report.Recommendations[0].SKUCode)
	require.InDelta(t, 1.0, report.Recommendations[0].Confidence, 0.0001)
	require.True(t, report.Recommendations[0].IsBundleCandidate)
	require.Equal(t, "SKU-X", report.Recommendations[1].SKUCode)
	require.InDelta(t, 2.0/7.0, report.Recommendations[1].Confidence, 0.0001)
	require.False(t, report.Recommendations[1].IsBundleCandidate)
}

func TestGetCartRecommendations_bundleThresholdIsTenantConfigurable(t *testing.T) {
	// Same 2/7 ≈ 0.286 pairing as the ranking test above -- below the
	// package default (0.30) but a tenant can lower their own threshold
	// to flag it as bundle-worthy anyway.
	basket := &stubBasketLister{out: []omsDomain.OrderSKUs{
		{OrderID: "o1", SKUIDs: []string{"sku-a", "sku-x"}},
		{OrderID: "o2", SKUIDs: []string{"sku-a", "sku-x"}},
		{OrderID: "o3", SKUIDs: []string{"sku-a"}},
		{OrderID: "o4", SKUIDs: []string{"sku-a"}},
		{OrderID: "o5", SKUIDs: []string{"sku-a"}},
		{OrderID: "o6", SKUIDs: []string{"sku-a"}},
		{OrderID: "o7", SKUIDs: []string{"sku-a"}},
	}}
	skuCodeLookup := &stubSKUCodeLookup{byCode: map[string]*pmsDomain.SKU{"SKU-A": {ID: "sku-a"}}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{
		"sku-x": {ID: "sku-x", SKUCode: "SKU-X", VariantID: "variant-x"},
	}}
	variantLookup := &stubVariantLookupForSlowMoving{byID: map[string]*pmsDomain.Variant{
		"variant-x": {ID: "variant-x", ProductID: "product-x"},
	}}
	productLookup := &stubProductLookupForSlowMoving{byID: map[string]*pmsDomain.Product{
		"product-x": {ID: "product-x", Name: "Widget X"},
	}}
	settingsRepo := &stubSettingsRepo{settings: &settingsDomain.TenantSettings{
		BundleConfidenceThreshold: 0.2,
		UpdatedAt:                 time.Now(),
	}}
	uc := appreports.NewGetCartRecommendationsUseCase(newMockGormDB(t), basket, settingsRepo, skuLookup, skuCodeLookup, variantLookup, productLookup)

	report, err := uc.Execute(context.Background(), []appreports.CartLine{{SKUCode: "SKU-A", Quantity: 1}})

	require.NoError(t, err)
	require.Len(t, report.Recommendations, 1)
	require.True(t, report.Recommendations[0].IsBundleCandidate)
}

func TestGetCartRecommendations_ignoresExplicitZeroThresholdOnAnUnsavedTenant(t *testing.T) {
	// stubSettingsRepo{} (no settings.UpdatedAt) represents a tenant that
	// has never saved TenantSettings -- BundleConfidenceThreshold's
	// zero-value-means-unset convention means this must fall back to the
	// package default (0.30), not treat 0 as "flag everything".
	basket := &stubBasketLister{out: []omsDomain.OrderSKUs{
		{OrderID: "o1", SKUIDs: []string{"sku-a", "sku-x"}},
		{OrderID: "o2", SKUIDs: []string{"sku-a", "sku-x"}},
		{OrderID: "o3", SKUIDs: []string{"sku-a"}},
		{OrderID: "o4", SKUIDs: []string{"sku-a"}},
		{OrderID: "o5", SKUIDs: []string{"sku-a"}},
		{OrderID: "o6", SKUIDs: []string{"sku-a"}},
		{OrderID: "o7", SKUIDs: []string{"sku-a"}},
	}}
	skuCodeLookup := &stubSKUCodeLookup{byCode: map[string]*pmsDomain.SKU{"SKU-A": {ID: "sku-a"}}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{
		"sku-x": {ID: "sku-x", SKUCode: "SKU-X", VariantID: "variant-x"},
	}}
	variantLookup := &stubVariantLookupForSlowMoving{byID: map[string]*pmsDomain.Variant{
		"variant-x": {ID: "variant-x", ProductID: "product-x"},
	}}
	productLookup := &stubProductLookupForSlowMoving{byID: map[string]*pmsDomain.Product{
		"product-x": {ID: "product-x", Name: "Widget X"},
	}}
	uc := appreports.NewGetCartRecommendationsUseCase(newMockGormDB(t), basket, &stubSettingsRepo{}, skuLookup, skuCodeLookup, variantLookup, productLookup)

	report, err := uc.Execute(context.Background(), []appreports.CartLine{{SKUCode: "SKU-A", Quantity: 1}})

	require.NoError(t, err)
	require.Len(t, report.Recommendations, 1)
	require.False(t, report.Recommendations[0].IsBundleCandidate)
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
