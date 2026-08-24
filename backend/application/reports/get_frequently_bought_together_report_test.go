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

type stubBasketLister struct {
	out []omsDomain.OrderSKUs
	err error
}

func (s *stubBasketLister) ListOrderSKUsBetween(_ context.Context, _, _ time.Time, _ string) ([]omsDomain.OrderSKUs, error) {
	return s.out, s.err
}

func TestGetFrequentlyBoughtTogetherReport_findsPairsAboveMinSupport(t *testing.T) {
	basket := &stubBasketLister{out: []omsDomain.OrderSKUs{
		{OrderID: "o1", SKUIDs: []string{"sku-a", "sku-b"}},
		{OrderID: "o2", SKUIDs: []string{"sku-a", "sku-b"}},
		{OrderID: "o3", SKUIDs: []string{"sku-a"}},
	}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{
		"sku-a": {ID: "sku-a", SKUCode: "SKU-A", VariantID: "variant-1"},
		"sku-b": {ID: "sku-b", SKUCode: "SKU-B", VariantID: "variant-1"},
	}}
	variantLookup := &stubVariantLookupForSlowMoving{byID: map[string]*pmsDomain.Variant{
		"variant-1": {ID: "variant-1", ProductID: "product-1"},
	}}
	productLookup := &stubProductLookupForSlowMoving{byID: map[string]*pmsDomain.Product{
		"product-1": {ID: "product-1", Name: "Widget"},
	}}
	uc := appreports.NewGetFrequentlyBoughtTogetherReportUseCase(newMockGormDB(t), basket, &stubSettingsRepo{}, skuLookup, variantLookup, productLookup)

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), 2)

	require.NoError(t, err)
	require.Equal(t, int64(3), result.TotalOrders)
	require.Len(t, result.Rows, 1)
	row := result.Rows[0]
	require.Equal(t, int64(2), row.Support)
	require.InDelta(t, 66.67, row.SupportPercent, 0.1) // 2/3 orders
	require.Len(t, row.Items, 2)
	codes := []string{row.Items[0].SKUCode, row.Items[1].SKUCode}
	require.ElementsMatch(t, []string{"SKU-A", "SKU-B"}, codes)
	require.Equal(t, "Widget", row.Items[0].ProductName)
}

func TestGetFrequentlyBoughtTogetherReport_excludesSingletonItemsets(t *testing.T) {
	basket := &stubBasketLister{out: []omsDomain.OrderSKUs{
		{OrderID: "o1", SKUIDs: []string{"sku-a"}},
		{OrderID: "o2", SKUIDs: []string{"sku-a"}},
	}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{"sku-a": {ID: "sku-a", SKUCode: "SKU-A"}}}
	uc := appreports.NewGetFrequentlyBoughtTogetherReportUseCase(newMockGormDB(t), basket, &stubSettingsRepo{}, skuLookup, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), 1)

	require.NoError(t, err)
	require.Empty(t, result.Rows, "a SKU frequently bought alone is not \"frequently bought together\"")
}

func TestGetFrequentlyBoughtTogetherReport_defaultsMinSupportWhenNotPositive(t *testing.T) {
	basket := &stubBasketLister{out: []omsDomain.OrderSKUs{
		{OrderID: "o1", SKUIDs: []string{"sku-a", "sku-b"}},
	}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{
		"sku-a": {ID: "sku-a", SKUCode: "SKU-A"}, "sku-b": {ID: "sku-b", SKUCode: "SKU-B"},
	}}
	uc := appreports.NewGetFrequentlyBoughtTogetherReportUseCase(newMockGormDB(t), basket, &stubSettingsRepo{}, skuLookup, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	// A single order's pair has support 1 -- below the default min support
	// (2), so it must not appear even though minSupport=0 was passed
	// (0/negative means "use the default", not "no threshold").
	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), 0)

	require.NoError(t, err)
	require.Empty(t, result.Rows)
}

func TestGetFrequentlyBoughtTogetherReport_toleratesADeletedSKU(t *testing.T) {
	basket := &stubBasketLister{out: []omsDomain.OrderSKUs{
		{OrderID: "o1", SKUIDs: []string{"sku-a", "sku-deleted"}},
		{OrderID: "o2", SKUIDs: []string{"sku-a", "sku-deleted"}},
	}}
	skuLookup := &stubSKUCostPriceLookup{byID: map[string]*pmsDomain.SKU{"sku-a": {ID: "sku-a", SKUCode: "SKU-A"}}}
	uc := appreports.NewGetFrequentlyBoughtTogetherReportUseCase(newMockGormDB(t), basket, &stubSettingsRepo{}, skuLookup, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	result, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), 2)

	require.NoError(t, err)
	require.Len(t, result.Rows, 1)
	var deletedItem *appreports.FrequentlyBoughtTogetherItem
	for i, item := range result.Rows[0].Items {
		if item.SKUID == "sku-deleted" {
			deletedItem = &result.Rows[0].Items[i]
		}
	}
	require.NotNil(t, deletedItem, "a since-deleted SKU must still appear in the itemset")
	require.Equal(t, "", deletedItem.SKUCode)
}

func TestGetFrequentlyBoughtTogetherReport_rejectsToBeforeFrom(t *testing.T) {
	uc := appreports.NewGetFrequentlyBoughtTogetherReportUseCase(newMockGormDB(t), &stubBasketLister{}, &stubSettingsRepo{}, &stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	_, err := uc.Execute(context.Background(), time.Now(), time.Now().Add(-time.Hour), 2)

	require.Error(t, err)
}

func TestGetFrequentlyBoughtTogetherReport_propagatesBasketRepoError(t *testing.T) {
	uc := appreports.NewGetFrequentlyBoughtTogetherReportUseCase(newMockGormDB(t), &stubBasketLister{err: context.DeadlineExceeded}, &stubSettingsRepo{}, &stubSKUCostPriceLookup{}, &stubVariantLookupForSlowMoving{}, &stubProductLookupForSlowMoving{})

	_, err := uc.Execute(context.Background(), time.Now().Add(-time.Hour), time.Now(), 2)

	require.Error(t, err)
}
