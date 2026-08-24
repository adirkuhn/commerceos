package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestGetProductRecommendations_ranksCoOccurringSKUsBySupportDesc(t *testing.T) {
	basket := &stubBasketLister{out: []omsDomain.OrderSKUs{
		{OrderID: "o1", SKUIDs: []string{"sku-target", "sku-b"}},
		{OrderID: "o2", SKUIDs: []string{"sku-target", "sku-b"}},
		{OrderID: "o3", SKUIDs: []string{"sku-target", "sku-b"}},
		{OrderID: "o4", SKUIDs: []string{"sku-target", "sku-c"}},
		{OrderID: "o5", SKUIDs: []string{"sku-target", "sku-c"}},
		{OrderID: "o6", SKUIDs: []string{"sku-x", "sku-y"}}, // unrelated, must not appear
	}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{
		"sku-b": {ID: "sku-b", SKUCode: "SKU-B", VariantID: "variant-b"},
		"sku-c": {ID: "sku-c", SKUCode: "SKU-C", VariantID: "variant-c"},
	}}
	variantLookup := &stubVariantLookupForSlowMoving{byID: map[string]*pmsDomain.Variant{
		"variant-b": {ID: "variant-b", ProductID: "product-b"},
		"variant-c": {ID: "variant-c", ProductID: "product-c"},
	}}
	productLookup := &stubProductLookupForSlowMoving{byID: map[string]*pmsDomain.Product{
		"product-b": {ID: "product-b", Name: "Widget B"},
		"product-c": {ID: "product-c", Name: "Widget C"},
	}}
	uc := appreports.NewGetProductRecommendationsUseCase(newMockGormDB(t), basket, &stubSettingsRepo{}, skuLookup, variantLookup, productLookup)

	report, err := uc.Execute(context.Background(), "sku-target", time.Now().Add(-time.Hour), time.Now(), 2, 10)

	require.NoError(t, err)
	require.Equal(t, "sku-target", report.SKUID)
	require.Equal(t, int64(6), report.TotalOrders)
	require.Len(t, report.Rows, 2)
	require.Equal(t, "sku-b", report.Rows[0].SKUID)
	require.Equal(t, "SKU-B", report.Rows[0].SKUCode)
	require.Equal(t, "Widget B", report.Rows[0].ProductName)
	require.Equal(t, int64(3), report.Rows[0].Support)
	require.Equal(t, "sku-c", report.Rows[1].SKUID)
	require.Equal(t, int64(2), report.Rows[1].Support)
}

func TestGetProductRecommendations_excludesUnrelatedSKUs(t *testing.T) {
	basket := &stubBasketLister{out: []omsDomain.OrderSKUs{
		{OrderID: "o1", SKUIDs: []string{"sku-target", "sku-b"}},
		{OrderID: "o2", SKUIDs: []string{"sku-target", "sku-b"}},
		{OrderID: "o3", SKUIDs: []string{"sku-x", "sku-y"}},
		{OrderID: "o4", SKUIDs: []string{"sku-x", "sku-y"}},
	}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{"sku-b": {ID: "sku-b", SKUCode: "SKU-B"}}}
	uc := appreports.NewGetProductRecommendationsUseCase(newMockGormDB(t), basket, &stubSettingsRepo{}, skuLookup, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	report, err := uc.Execute(context.Background(), "sku-target", time.Now().Add(-time.Hour), time.Now(), 2, 10)

	require.NoError(t, err)
	require.Len(t, report.Rows, 1)
	require.Equal(t, "sku-b", report.Rows[0].SKUID)
}

func TestGetProductRecommendations_toleratesADeletedCandidateSKU(t *testing.T) {
	basket := &stubBasketLister{out: []omsDomain.OrderSKUs{
		{OrderID: "o1", SKUIDs: []string{"sku-target", "sku-deleted"}},
		{OrderID: "o2", SKUIDs: []string{"sku-target", "sku-deleted"}},
	}}
	uc := appreports.NewGetProductRecommendationsUseCase(newMockGormDB(t), basket, &stubSettingsRepo{}, &stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	report, err := uc.Execute(context.Background(), "sku-target", time.Now().Add(-time.Hour), time.Now(), 2, 10)

	require.NoError(t, err)
	require.Len(t, report.Rows, 1)
	require.Equal(t, "sku-deleted", report.Rows[0].SKUID)
	require.Equal(t, "", report.Rows[0].SKUCode)
}

func TestGetProductRecommendations_capsToLimit(t *testing.T) {
	basket := &stubBasketLister{out: []omsDomain.OrderSKUs{
		{OrderID: "o1", SKUIDs: []string{"sku-target", "sku-a", "sku-b", "sku-c"}},
		{OrderID: "o2", SKUIDs: []string{"sku-target", "sku-a", "sku-b", "sku-c"}},
	}}
	uc := appreports.NewGetProductRecommendationsUseCase(newMockGormDB(t), basket, &stubSettingsRepo{}, &stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	report, err := uc.Execute(context.Background(), "sku-target", time.Now().Add(-time.Hour), time.Now(), 2, 2)

	require.NoError(t, err)
	require.Len(t, report.Rows, 2)
}

func TestGetProductRecommendations_defaultsMinSupportAndLimitWhenNotPositive(t *testing.T) {
	basket := &stubBasketLister{out: []omsDomain.OrderSKUs{
		{OrderID: "o1", SKUIDs: []string{"sku-target", "sku-b"}},
	}}
	uc := appreports.NewGetProductRecommendationsUseCase(newMockGormDB(t), basket, &stubSettingsRepo{}, &stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	// Support 1 is below the default min support (2) -- must not appear.
	report, err := uc.Execute(context.Background(), "sku-target", time.Now().Add(-time.Hour), time.Now(), 0, 0)

	require.NoError(t, err)
	require.Empty(t, report.Rows)
}

func TestGetProductRecommendations_rejectsEmptySKUID(t *testing.T) {
	uc := appreports.NewGetProductRecommendationsUseCase(newMockGormDB(t), &stubBasketLister{}, &stubSettingsRepo{}, &stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	_, err := uc.Execute(context.Background(), "", time.Now().Add(-time.Hour), time.Now(), 2, 10)

	require.Error(t, err)
}

func TestGetProductRecommendations_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetProductRecommendationsUseCase(newMockGormDB(t), &stubBasketLister{}, &stubSettingsRepo{}, &stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	_, err := uc.Execute(context.Background(), "sku-target", time.Now(), time.Now().Add(-time.Hour), 2, 10)

	require.Error(t, err)
}

func TestGetProductRecommendations_propagatesBasketRepoError(t *testing.T) {
	uc := appreports.NewGetProductRecommendationsUseCase(newMockGormDB(t), &stubBasketLister{err: context.DeadlineExceeded}, &stubSettingsRepo{}, &stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	_, err := uc.Execute(context.Background(), "sku-target", time.Now().Add(-time.Hour), time.Now(), 2, 10)

	require.Error(t, err)
}
