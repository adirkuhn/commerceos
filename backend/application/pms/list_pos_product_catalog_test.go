package pms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubVariantsByProductIDsRepo struct {
	byProductID map[string][]pmsDomain.Variant
	err         error
	gotIDs      []string
}

func (s *stubVariantsByProductIDsRepo) ListByProductIDs(_ context.Context, productIDs []string) ([]pmsDomain.Variant, error) {
	s.gotIDs = productIDs
	if s.err != nil {
		return nil, s.err
	}
	var out []pmsDomain.Variant
	for _, id := range productIDs {
		out = append(out, s.byProductID[id]...)
	}
	return out, nil
}

type stubSKUsByVariantIDsRepo struct {
	byVariantID map[string][]pmsDomain.SKU
	err         error
	gotIDs      []string
}

func (s *stubSKUsByVariantIDsRepo) ListByVariantIDs(_ context.Context, variantIDs []string) ([]pmsDomain.SKU, error) {
	s.gotIDs = variantIDs
	if s.err != nil {
		return nil, s.err
	}
	var out []pmsDomain.SKU
	for _, id := range variantIDs {
		out = append(out, s.byVariantID[id]...)
	}
	return out, nil
}

type stubMediaByProductIDsRepo struct {
	byProductID map[string][]pmsDomain.Media
	err         error
	gotIDs      []string
}

func (s *stubMediaByProductIDsRepo) ListByProductIDs(_ context.Context, productIDs []string) ([]pmsDomain.Media, error) {
	s.gotIDs = productIDs
	if s.err != nil {
		return nil, s.err
	}
	var out []pmsDomain.Media
	for _, id := range productIDs {
		out = append(out, s.byProductID[id]...)
	}
	return out, nil
}

func TestListPOSProductCatalog_assemblesSKUsPerProductAcrossVariants(t *testing.T) {
	productRepo := &stubProductRepo{
		listResult: []pmsDomain.Product{{ID: "p1", Name: "Widget"}},
		listTotal:  1,
	}
	listProducts := apppms.NewListProductsUseCase(newMockGormDB(t), productRepo)
	variantRepo := &stubVariantsByProductIDsRepo{byProductID: map[string][]pmsDomain.Variant{
		"p1": {{ID: "v1", ProductID: "p1", Name: "Small"}, {ID: "v2", ProductID: "p1", Name: "Large"}},
	}}
	skuRepo := &stubSKUsByVariantIDsRepo{byVariantID: map[string][]pmsDomain.SKU{
		"v1": {{ID: "sku-1", VariantID: "v1", Barcode: "111", SKUCode: "W-S"}},
		"v2": {{ID: "sku-2", VariantID: "v2", Barcode: "222", SKUCode: "W-L"}},
	}}
	uc := apppms.NewListPOSProductCatalogUseCase(newMockGormDBN(t, 3), listProducts, variantRepo, skuRepo, &stubMediaByProductIDsRepo{})

	result, err := uc.Execute(context.Background(), apppms.ListProductsInput{})

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, result.Products, 1)
	require.Len(t, result.Products[0].SKUs, 2)
	require.Equal(t, "sku-1", result.Products[0].SKUs[0].ID)
	require.Equal(t, "Small", result.Products[0].SKUs[0].VariantName)
	require.Equal(t, "sku-2", result.Products[0].SKUs[1].ID)
	require.Equal(t, "Large", result.Products[0].SKUs[1].VariantName)
	require.Equal(t, []string{"p1"}, variantRepo.gotIDs)
	require.ElementsMatch(t, []string{"v1", "v2"}, skuRepo.gotIDs)
}

func TestListPOSProductCatalog_productWithNoSKUsGetsEmptySlice(t *testing.T) {
	productRepo := &stubProductRepo{listResult: []pmsDomain.Product{{ID: "p1", Name: "Widget"}}, listTotal: 1}
	listProducts := apppms.NewListProductsUseCase(newMockGormDB(t), productRepo)
	variantRepo := &stubVariantsByProductIDsRepo{}
	skuRepo := &stubSKUsByVariantIDsRepo{}
	uc := apppms.NewListPOSProductCatalogUseCase(newMockGormDBN(t, 3), listProducts, variantRepo, skuRepo, &stubMediaByProductIDsRepo{})

	result, err := uc.Execute(context.Background(), apppms.ListProductsInput{})

	require.NoError(t, err)
	require.Len(t, result.Products, 1)
	require.Empty(t, result.Products[0].SKUs)
}

func TestListPOSProductCatalog_emptyPageSkipsBulkLookups(t *testing.T) {
	productRepo := &stubProductRepo{listResult: nil, listTotal: 0}
	listProducts := apppms.NewListProductsUseCase(newMockGormDB(t), productRepo)
	variantRepo := &stubVariantsByProductIDsRepo{}
	skuRepo := &stubSKUsByVariantIDsRepo{}
	mediaRepo := &stubMediaByProductIDsRepo{}
	uc := apppms.NewListPOSProductCatalogUseCase(newMockGormDB(t), listProducts, variantRepo, skuRepo, mediaRepo)

	result, err := uc.Execute(context.Background(), apppms.ListProductsInput{})

	require.NoError(t, err)
	require.Empty(t, result.Products)
	require.Equal(t, int64(0), result.Total)
	require.Nil(t, variantRepo.gotIDs, "bulk variant lookup must not run for an empty page")
	require.Nil(t, skuRepo.gotIDs, "bulk sku lookup must not run for an empty page")
	require.Nil(t, mediaRepo.gotIDs, "bulk media lookup must not run for an empty page")
}

func TestListPOSProductCatalog_propagatesListProductsError(t *testing.T) {
	productRepo := &stubProductRepo{listErr: errors.New("db down")}
	listProducts := apppms.NewListProductsUseCase(newMockGormDB(t), productRepo)
	uc := apppms.NewListPOSProductCatalogUseCase(newMockGormDB(t), listProducts, &stubVariantsByProductIDsRepo{}, &stubSKUsByVariantIDsRepo{}, &stubMediaByProductIDsRepo{})

	_, err := uc.Execute(context.Background(), apppms.ListProductsInput{})

	require.Error(t, err)
	require.ErrorContains(t, err, "db down")
}

func TestListPOSProductCatalog_propagatesVariantRepoError(t *testing.T) {
	productRepo := &stubProductRepo{listResult: []pmsDomain.Product{{ID: "p1"}}, listTotal: 1}
	listProducts := apppms.NewListProductsUseCase(newMockGormDB(t), productRepo)
	variantRepo := &stubVariantsByProductIDsRepo{err: errors.New("variant lookup failed")}
	uc := apppms.NewListPOSProductCatalogUseCase(newMockGormDBN(t, 2), listProducts, variantRepo, &stubSKUsByVariantIDsRepo{}, &stubMediaByProductIDsRepo{})

	_, err := uc.Execute(context.Background(), apppms.ListProductsInput{})

	require.Error(t, err)
	require.ErrorContains(t, err, "variant lookup failed")
}

func TestListPOSProductCatalog_propagatesMediaRepoError(t *testing.T) {
	productRepo := &stubProductRepo{listResult: []pmsDomain.Product{{ID: "p1"}}, listTotal: 1}
	listProducts := apppms.NewListProductsUseCase(newMockGormDB(t), productRepo)
	mediaRepo := &stubMediaByProductIDsRepo{err: errors.New("media lookup failed")}
	uc := apppms.NewListPOSProductCatalogUseCase(newMockGormDBN(t, 2), listProducts, &stubVariantsByProductIDsRepo{}, &stubSKUsByVariantIDsRepo{}, mediaRepo)

	_, err := uc.Execute(context.Background(), apppms.ListProductsInput{})

	require.Error(t, err)
	require.ErrorContains(t, err, "media lookup failed")
}

// The whole point of adding this: the POS UI needs one thumbnail per
// product to match/import against, without a second per-product call.
func TestListPOSProductCatalog_setsImageURLToTheLowestPositionMedia(t *testing.T) {
	productRepo := &stubProductRepo{listResult: []pmsDomain.Product{{ID: "p1", Name: "Widget"}}, listTotal: 1}
	listProducts := apppms.NewListProductsUseCase(newMockGormDB(t), productRepo)
	mediaRepo := &stubMediaByProductIDsRepo{byProductID: map[string][]pmsDomain.Media{
		"p1": {
			{ID: "m1", ProductID: strPtr("p1"), URL: "https://example.com/featured.jpg", Position: 0},
			{ID: "m2", ProductID: strPtr("p1"), URL: "https://example.com/second.jpg", Position: 1},
		},
	}}
	uc := apppms.NewListPOSProductCatalogUseCase(newMockGormDBN(t, 3), listProducts, &stubVariantsByProductIDsRepo{}, &stubSKUsByVariantIDsRepo{}, mediaRepo)

	result, err := uc.Execute(context.Background(), apppms.ListProductsInput{})

	require.NoError(t, err)
	require.Len(t, result.Products, 1)
	require.NotNil(t, result.Products[0].ImageURL)
	require.Equal(t, "https://example.com/featured.jpg", *result.Products[0].ImageURL)
	require.Equal(t, []string{"p1"}, mediaRepo.gotIDs)
}

func TestListPOSProductCatalog_imageURLIsNilWhenProductHasNoMedia(t *testing.T) {
	productRepo := &stubProductRepo{listResult: []pmsDomain.Product{{ID: "p1", Name: "Widget"}}, listTotal: 1}
	listProducts := apppms.NewListProductsUseCase(newMockGormDB(t), productRepo)
	uc := apppms.NewListPOSProductCatalogUseCase(newMockGormDBN(t, 3), listProducts, &stubVariantsByProductIDsRepo{}, &stubSKUsByVariantIDsRepo{}, &stubMediaByProductIDsRepo{})

	result, err := uc.Execute(context.Background(), apppms.ListProductsInput{})

	require.NoError(t, err)
	require.Nil(t, result.Products[0].ImageURL)
}
