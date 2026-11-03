package pms_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	pmshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/pms"
)

func init() { gin.SetMode(gin.TestMode) }

// ── Stub use cases ────────────────────────────────────────────────────────────

type stubListCategories struct {
	out []pmsDomain.Category
	err error
}

func (s *stubListCategories) Execute(_ context.Context) ([]pmsDomain.Category, error) {
	return s.out, s.err
}

type stubCreateCategory struct {
	out *pmsDomain.Category
	err error
}

func (s *stubCreateCategory) Execute(_ context.Context, _ apppms.CreateCategoryInput) (*pmsDomain.Category, error) {
	return s.out, s.err
}

type stubUpdateCategory struct {
	out *pmsDomain.Category
	err error
}

func (s *stubUpdateCategory) Execute(_ context.Context, _ apppms.UpdateCategoryInput) (*pmsDomain.Category, error) {
	return s.out, s.err
}

type stubDeleteCategory struct{ err error }

func (s *stubDeleteCategory) Execute(_ context.Context, _ string) error { return s.err }

type stubListProducts struct {
	out      []apppms.POSProductWithSKUs
	total    int64
	err      error
	gotInput apppms.ListProductsInput
}

func (s *stubListProducts) Execute(_ context.Context, in apppms.ListProductsInput) (*apppms.ListPOSProductCatalogResult, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	total := s.total
	if total == 0 {
		total = int64(len(s.out))
	}
	return &apppms.ListPOSProductCatalogResult{Products: s.out, Total: total}, nil
}

type stubCreateProduct struct {
	out *pmsDomain.Product
	err error
}

func (s *stubCreateProduct) Execute(_ context.Context, _ apppms.CreateProductInput) (*pmsDomain.Product, error) {
	return s.out, s.err
}

type stubGetProduct struct {
	out *apppms.ProductDetail
	err error
}

func (s *stubGetProduct) Execute(_ context.Context, _ string) (*apppms.ProductDetail, error) {
	return s.out, s.err
}

type stubUpdateProduct struct {
	out   *pmsDomain.Product
	err   error
	gotIn apppms.UpdateProductInput
}

func (s *stubUpdateProduct) Execute(_ context.Context, in apppms.UpdateProductInput) (*pmsDomain.Product, error) {
	s.gotIn = in
	return s.out, s.err
}

type stubDuplicateProduct struct {
	out          *apppms.ProductDetail
	err          error
	gotProductID string
}

func (s *stubDuplicateProduct) Execute(_ context.Context, sourceProductID string) (*apppms.ProductDetail, error) {
	s.gotProductID = sourceProductID
	return s.out, s.err
}

type stubDeleteProduct struct{ err error }

func (s *stubDeleteProduct) Execute(_ context.Context, _ string) error { return s.err }

type stubCreateVariant struct {
	out *pmsDomain.Variant
	err error
}

func (s *stubCreateVariant) Execute(_ context.Context, _ apppms.CreateVariantInput) (*pmsDomain.Variant, error) {
	return s.out, s.err
}

type stubUpdateVariant struct {
	out *pmsDomain.Variant
	err error
}

func (s *stubUpdateVariant) Execute(_ context.Context, _ apppms.UpdateVariantInput) (*pmsDomain.Variant, error) {
	return s.out, s.err
}

type stubDeleteVariant struct{ err error }

func (s *stubDeleteVariant) Execute(_ context.Context, _ string) error { return s.err }

type stubCreateSKU struct {
	out      *pmsDomain.SKU
	err      error
	gotInput apppms.CreateSKUInput
}

func (s *stubCreateSKU) Execute(_ context.Context, in apppms.CreateSKUInput) (*pmsDomain.SKU, error) {
	s.gotInput = in
	return s.out, s.err
}

type stubUpdateSKU struct {
	out      *pmsDomain.SKU
	err      error
	gotInput apppms.UpdateSKUInput
}

func (s *stubUpdateSKU) Execute(_ context.Context, in apppms.UpdateSKUInput) (*pmsDomain.SKU, error) {
	s.gotInput = in
	return s.out, s.err
}

type stubDeleteSKU struct{ err error }

func (s *stubDeleteSKU) Execute(_ context.Context, _ string) error { return s.err }

type stubSetManualCostPrice struct {
	out                    *pmsDomain.SKU
	err                    error
	gotSKUID               string
	gotCostPriceMinorUnits int64
}

func (s *stubSetManualCostPrice) Execute(_ context.Context, skuID string, costPriceMinorUnits int64) (*pmsDomain.SKU, error) {
	s.gotSKUID = skuID
	s.gotCostPriceMinorUnits = costPriceMinorUnits
	return s.out, s.err
}

type stubCorrectCost struct {
	out   *pmsDomain.SKU
	err   error
	gotIn apppms.CorrectCostInput
}

func (s *stubCorrectCost) Execute(_ context.Context, in apppms.CorrectCostInput) (*pmsDomain.SKU, error) {
	s.gotIn = in
	return s.out, s.err
}

type stubClearCostPriceOverride struct {
	out      *pmsDomain.SKU
	err      error
	gotSKUID string
}

func (s *stubClearCostPriceOverride) Execute(_ context.Context, skuID string) (*pmsDomain.SKU, error) {
	s.gotSKUID = skuID
	return s.out, s.err
}

type stubUploadMedia struct {
	out *pmsDomain.Media
	err error
}

func (s *stubUploadMedia) Execute(_ context.Context, _ apppms.UploadMediaInput) (*pmsDomain.Media, error) {
	return s.out, s.err
}

type stubListProductMedia struct {
	out []pmsDomain.Media
	err error
}

func (s *stubListProductMedia) Execute(_ context.Context, _ string) ([]pmsDomain.Media, error) {
	return s.out, s.err
}

type stubDeleteMedia struct{ err error }

func (s *stubDeleteMedia) Execute(_ context.Context, _ string) error { return s.err }

type stubUpdateMedia struct {
	out *pmsDomain.Media
	err error
}

func (s *stubUpdateMedia) Execute(_ context.Context, _ apppms.UpdateMediaInput) (*pmsDomain.Media, error) {
	return s.out, s.err
}

type stubResolveBarcode struct {
	out      *apppms.ResolveBarcodeResult
	err      error
	gotInput string
}

func (s *stubResolveBarcode) Execute(_ context.Context, code string) (*apppms.ResolveBarcodeResult, error) {
	s.gotInput = code
	return s.out, s.err
}

type stubGetSKUMarginInsight struct {
	out *apppms.GetSKUMarginInsightResult
	err error
}

func (s *stubGetSKUMarginInsight) Execute(_ context.Context, _ string) (*apppms.GetSKUMarginInsightResult, error) {
	return s.out, s.err
}

type stubListCostPriceHistory struct {
	out []pmsDomain.SKUCostPriceHistory
	err error
}

func (s *stubListCostPriceHistory) Execute(_ context.Context, _ string, _ int) ([]pmsDomain.SKUCostPriceHistory, error) {
	return s.out, s.err
}

type stubListMarginReport struct {
	out *apppms.ListMarginReportResult
	err error
}

func (s *stubListMarginReport) Execute(_ context.Context, _ apppms.ListMarginReportInput) (*apppms.ListMarginReportResult, error) {
	return s.out, s.err
}

type stubListSKUPriceHistory struct {
	out []pmsDomain.SKUPriceHistory
	err error
}

func (s *stubListSKUPriceHistory) Execute(_ context.Context, _ string, _ int) ([]pmsDomain.SKUPriceHistory, error) {
	return s.out, s.err
}

type stubListProductHistory struct {
	out []pmsDomain.ProductHistory
	err error
}

func (s *stubListProductHistory) Execute(_ context.Context, _ pmsDomain.ProductHistoryEntityType, _ string) ([]pmsDomain.ProductHistory, error) {
	return s.out, s.err
}

type stubListSKUCurrencyPrices struct {
	out []pmsDomain.SKUCurrencyPrice
	err error
}

func (s *stubListSKUCurrencyPrices) Execute(_ context.Context, _ string) ([]pmsDomain.SKUCurrencyPrice, error) {
	return s.out, s.err
}

type stubUpsertSKUCurrencyPrice struct {
	out      *pmsDomain.SKUCurrencyPrice
	err      error
	gotInput apppms.UpsertSKUCurrencyPriceInput
}

func (s *stubUpsertSKUCurrencyPrice) Execute(_ context.Context, in apppms.UpsertSKUCurrencyPriceInput) (*pmsDomain.SKUCurrencyPrice, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &pmsDomain.SKUCurrencyPrice{SKUID: in.SKUID, CurrencyCode: in.CurrencyCode, PriceMinorUnits: in.PriceMinorUnits, SalePriceMinorUnits: in.SalePriceMinorUnits}, nil
}

type stubDeleteSKUCurrencyPrice struct {
	err         error
	gotSKUID    string
	gotCurrency string
}

func (s *stubDeleteSKUCurrencyPrice) Execute(_ context.Context, skuID, currencyCode string) error {
	s.gotSKUID, s.gotCurrency = skuID, currencyCode
	return s.err
}

type stubListBarcodeAliases struct {
	out      []pmsDomain.BarcodeAlias
	err      error
	gotSKUID string
}

func (s *stubListBarcodeAliases) Execute(_ context.Context, skuID string) ([]pmsDomain.BarcodeAlias, error) {
	s.gotSKUID = skuID
	return s.out, s.err
}

type stubCreateBarcodeAlias struct {
	out      *pmsDomain.BarcodeAlias
	err      error
	gotInput apppms.CreateBarcodeAliasInput
}

func (s *stubCreateBarcodeAlias) Execute(_ context.Context, in apppms.CreateBarcodeAliasInput) (*pmsDomain.BarcodeAlias, error) {
	s.gotInput = in
	return s.out, s.err
}

type stubDeleteBarcodeAlias struct {
	err   error
	gotID string
}

func (s *stubDeleteBarcodeAlias) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

type stubListScaleCodeAliases struct {
	out      []pmsDomain.ScaleCodeAlias
	err      error
	gotSKUID string
}

func (s *stubListScaleCodeAliases) Execute(_ context.Context, skuID string) ([]pmsDomain.ScaleCodeAlias, error) {
	s.gotSKUID = skuID
	return s.out, s.err
}

type stubCreateScaleCodeAlias struct {
	out      *pmsDomain.ScaleCodeAlias
	err      error
	gotInput apppms.CreateScaleCodeAliasInput
}

func (s *stubCreateScaleCodeAlias) Execute(_ context.Context, in apppms.CreateScaleCodeAliasInput) (*pmsDomain.ScaleCodeAlias, error) {
	s.gotInput = in
	return s.out, s.err
}

type stubDeleteScaleCodeAlias struct {
	err   error
	gotID string
}

func (s *stubDeleteScaleCodeAlias) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

// ── Helpers ───────────────────────────────────────────────────────────────────

type stubs struct {
	listCategories         stubListCategories
	createCategory         stubCreateCategory
	updateCategory         stubUpdateCategory
	deleteCategory         stubDeleteCategory
	listProducts           stubListProducts
	createProduct          stubCreateProduct
	getProduct             stubGetProduct
	updateProduct          stubUpdateProduct
	duplicateProduct       stubDuplicateProduct
	deleteProduct          stubDeleteProduct
	createVariant          stubCreateVariant
	updateVariant          stubUpdateVariant
	deleteVariant          stubDeleteVariant
	createSKU              stubCreateSKU
	updateSKU              stubUpdateSKU
	deleteSKU              stubDeleteSKU
	setManualCostPrice     stubSetManualCostPrice
	clearCostPriceOverride stubClearCostPriceOverride
	uploadMedia            stubUploadMedia
	listProductMedia       stubListProductMedia
	deleteMedia            stubDeleteMedia
	updateMedia            stubUpdateMedia
	resolveBarcode         stubResolveBarcode
	getSKUMarginInsight    stubGetSKUMarginInsight
	listCostPriceHistory   stubListCostPriceHistory
	listMarginReport       stubListMarginReport
	listSKUPriceHistory    stubListSKUPriceHistory
	listProductHistory     stubListProductHistory
	listSKUCurrencyPrices  stubListSKUCurrencyPrices
	upsertSKUCurrencyPrice stubUpsertSKUCurrencyPrice
	deleteSKUCurrencyPrice stubDeleteSKUCurrencyPrice
	listBarcodeAliases     stubListBarcodeAliases
	createBarcodeAlias     stubCreateBarcodeAlias
	deleteBarcodeAlias     stubDeleteBarcodeAlias
	listScaleCodeAliases   stubListScaleCodeAliases
	createScaleCodeAlias   stubCreateScaleCodeAlias
	deleteScaleCodeAlias   stubDeleteScaleCodeAlias
	correctCost            stubCorrectCost
	previewProductMerge    stubPreviewProductMerge
	mergeProducts          stubMergeProducts
	setTranslations        stubSetProductTranslations
	setLowStockThreshold   stubSetProductLowStockThreshold
	listPriceList          stubListPriceList
	setAutomaticCostPrice  stubSetManualCostPrice
}

func newTestRouter(s *stubs) *gin.Engine {
	h := pmshandler.NewHandler(
		&s.listCategories, &s.createCategory, &s.updateCategory, &s.deleteCategory,
		&s.listProducts, &s.createProduct, &s.getProduct, &s.updateProduct, &s.duplicateProduct, &s.deleteProduct,
		&s.createVariant, &s.updateVariant, &s.deleteVariant,
		&s.createSKU, &s.updateSKU, &s.deleteSKU,
		&s.setManualCostPrice, &s.clearCostPriceOverride,
		&s.uploadMedia, &s.listProductMedia, &s.deleteMedia, &s.updateMedia,
		&s.resolveBarcode,
		&s.getSKUMarginInsight, &s.listCostPriceHistory, &s.listMarginReport,
		&s.listSKUPriceHistory,
		&s.listProductHistory,
		&s.listSKUCurrencyPrices, &s.upsertSKUCurrencyPrice, &s.deleteSKUCurrencyPrice,
		&s.listBarcodeAliases, &s.createBarcodeAlias, &s.deleteBarcodeAlias,
		&s.listScaleCodeAliases, &s.createScaleCodeAlias, &s.deleteScaleCodeAlias,
		&s.correctCost,
		pmshandler.ProductMergeUseCases{Preview: &s.previewProductMerge, Merge: &s.mergeProducts},
		&s.setTranslations,
		&s.setLowStockThreshold,
		&s.listPriceList,
		&s.setAutomaticCostPrice,
	)
	r := gin.New()
	genpms.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ── Category tests ───────────────────────────────────────────────────────────

func TestListCategories_success(t *testing.T) {
	s := &stubs{listCategories: stubListCategories{out: []pmsDomain.Category{{ID: "c1", Name: "Electronics"}}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/categories", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Electronics")
}

func TestCreateCategory_success(t *testing.T) {
	s := &stubs{createCategory: stubCreateCategory{out: &pmsDomain.Category{ID: "c1", Name: "Electronics"}}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/categories", map[string]string{"name": "Electronics"})
	require.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Electronics")
}

func TestUpdateCategory_notFound(t *testing.T) {
	s := &stubs{updateCategory: stubUpdateCategory{err: pmsDomain.ErrCategoryNotFound}}
	w := doRequest(newTestRouter(s), http.MethodPatch, "/api/v1/pms/categories/missing", map[string]string{"name": "X"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteCategory_success(t *testing.T) {
	s := &stubs{}
	w := doRequest(newTestRouter(s), http.MethodDelete, "/api/v1/pms/categories/c1", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── Product tests ─────────────────────────────────────────────────────────────

func TestListProducts_returnsTotalAlongsideData(t *testing.T) {
	s := &stubs{listProducts: stubListProducts{
		out:   []apppms.POSProductWithSKUs{{Product: pmsDomain.Product{ID: "p1", Name: "Widget"}}},
		total: 806,
	}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/products", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"total":806`)
}

// TestListProducts_includesImageAndPriceFromTheFirstVariantsSKU proves
// the list response carries a thumbnail image and price/sale price --
// only the list endpoint does this (see toGenProductListItem).
func TestListProducts_includesImageAndPriceFromTheFirstVariantsSKU(t *testing.T) {
	imageURL := "https://example.com/widget.jpg"
	salePrice := int64(450)
	s := &stubs{listProducts: stubListProducts{out: []apppms.POSProductWithSKUs{{
		Product:  pmsDomain.Product{ID: "p1", Name: "Widget"},
		ImageURL: &imageURL,
		SKUs: []apppms.POSProductSKU{
			{SKU: pmsDomain.SKU{ID: "sku-1", PriceMinorUnits: 600, SalePriceMinorUnits: &salePrice, CurrencyCode: "EUR"}},
		},
	}}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/products", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"image_url":"https://example.com/widget.jpg"`)
	assert.Contains(t, w.Body.String(), `"price_minor_units":600`)
	assert.Contains(t, w.Body.String(), `"sale_price_minor_units":450`)
	assert.Contains(t, w.Body.String(), `"currency_code":"EUR"`)
}

// TestListProducts_omitsImageAndPriceWhenProductHasNoSKUYet proves a
// product with no variant/SKU at all yet (just created) doesn't crash the
// mapping -- image_url/price fields are simply omitted.
func TestListProducts_omitsImageAndPriceWhenProductHasNoSKUYet(t *testing.T) {
	s := &stubs{listProducts: stubListProducts{out: []apppms.POSProductWithSKUs{{
		Product: pmsDomain.Product{ID: "p1", Name: "Widget"},
	}}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/products", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), `"image_url"`)
	assert.NotContains(t, w.Body.String(), `"price_minor_units"`)
}

func TestListProducts_passesPagePerPageAndSearchQueryParams(t *testing.T) {
	s := &stubs{listProducts: stubListProducts{}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/products?page=3&per_page=50&search=shirt", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 3, s.listProducts.gotInput.Page)
	require.Equal(t, 50, s.listProducts.gotInput.PerPage)
	require.Equal(t, "shirt", s.listProducts.gotInput.Search)
}

func TestCreateProduct_duplicateSlugReturns409(t *testing.T) {
	s := &stubs{createProduct: stubCreateProduct{err: pmsDomain.ErrDuplicateSlug}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/products", map[string]any{
		"name": "Widget", "status": "draft",
	})
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestGetProduct_returnsNestedVariantsAndSKUs(t *testing.T) {
	mediaID := "media-1"
	detail := &apppms.ProductDetail{
		Product: pmsDomain.Product{ID: "p1", Name: "Widget", Status: pmsDomain.ProductStatusActive},
		Variants: []apppms.VariantDetail{
			{
				Variant: pmsDomain.Variant{ID: "v1", Name: "Large", FeaturedMediaID: &mediaID},
				SKUs:    []pmsDomain.SKU{{ID: "s1", SKUCode: "WID-L", PriceMinorUnits: 1999, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive}},
			},
		},
	}
	s := &stubs{getProduct: stubGetProduct{out: detail}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/products/p1", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "WID-L")
	assert.Contains(t, w.Body.String(), "Large")
	// Regression: toGenProductDetail built VariantWithSKUs by hand and
	// dropped FeaturedMediaId, so a variant's selected image always
	// reverted to "none" on reload after being set.
	assert.Contains(t, w.Body.String(), "media-1")
}

func TestGetProduct_notFound(t *testing.T) {
	s := &stubs{getProduct: stubGetProduct{err: pmsDomain.ErrProductNotFound}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/products/missing", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDuplicateProduct_success(t *testing.T) {
	detail := &apppms.ProductDetail{
		Product: pmsDomain.Product{ID: "p2", Name: "Widget (Copy)", Status: pmsDomain.ProductStatusDraft},
		Variants: []apppms.VariantDetail{
			{
				Variant: pmsDomain.Variant{ID: "v2", Name: "Large"},
				SKUs:    []pmsDomain.SKU{{ID: "s2", SKUCode: "WID-L-COPY", PriceMinorUnits: 1999, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive}},
			},
		},
	}
	s := &stubs{duplicateProduct: stubDuplicateProduct{out: detail}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/products/p1/duplicate", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "p1", s.duplicateProduct.gotProductID)
	assert.Contains(t, w.Body.String(), "Widget (Copy)")
	assert.Contains(t, w.Body.String(), "\"status\":\"draft\"")
	assert.Contains(t, w.Body.String(), "WID-L-COPY")
}

func TestDuplicateProduct_notFound(t *testing.T) {
	s := &stubs{duplicateProduct: stubDuplicateProduct{err: pmsDomain.ErrProductNotFound}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/products/missing/duplicate", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateProduct_success(t *testing.T) {
	s := &stubs{updateProduct: stubUpdateProduct{out: &pmsDomain.Product{ID: "p1", Name: "Widget v2", Status: pmsDomain.ProductStatusActive}}}
	w := doRequest(newTestRouter(s), http.MethodPatch, "/api/v1/pms/products/p1", map[string]any{
		"name": "Widget v2", "status": "active",
	})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Widget v2")
}

// TestUpdateProduct_passesSlugThrough proves a slug in the PATCH body
// reaches UpdateProductInput.Slug (non-nil) -- the API-layer half of the
// fix for products (esp. duplicates) being permanently stuck with their
// slug.
func TestUpdateProduct_passesSlugThrough(t *testing.T) {
	s := &stubs{updateProduct: stubUpdateProduct{out: &pmsDomain.Product{ID: "p1", Name: "Widget", Slug: "widget-v2", Status: pmsDomain.ProductStatusActive}}}
	w := doRequest(newTestRouter(s), http.MethodPatch, "/api/v1/pms/products/p1", map[string]any{
		"name": "Widget", "status": "active", "slug": "widget-v2",
	})
	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, s.updateProduct.gotIn.Slug)
	assert.Equal(t, "widget-v2", *s.updateProduct.gotIn.Slug)
}

// TestUpdateProduct_omittedSlugStaysNil proves omitting slug from the PATCH
// body leaves UpdateProductInput.Slug nil -- the use case's "leave slug
// unchanged" branch only fires when nil, not on an empty string.
func TestUpdateProduct_omittedSlugStaysNil(t *testing.T) {
	s := &stubs{updateProduct: stubUpdateProduct{out: &pmsDomain.Product{ID: "p1", Name: "Widget", Status: pmsDomain.ProductStatusActive}}}
	w := doRequest(newTestRouter(s), http.MethodPatch, "/api/v1/pms/products/p1", map[string]any{
		"name": "Widget", "status": "active",
	})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, s.updateProduct.gotIn.Slug)
}

func TestCreateProduct_returnsTags(t *testing.T) {
	s := &stubs{createProduct: stubCreateProduct{out: &pmsDomain.Product{
		ID: "p1", Name: "Widget", Status: pmsDomain.ProductStatusDraft, Tags: []string{"summer", "sale"},
	}}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/products", map[string]any{
		"name": "Widget", "status": "draft", "tags": []string{"summer", "sale"},
	})
	require.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "summer")
	assert.Contains(t, w.Body.String(), "sale")
}

func TestDeleteProduct_success(t *testing.T) {
	s := &stubs{}
	w := doRequest(newTestRouter(s), http.MethodDelete, "/api/v1/pms/products/p1", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── Variant/SKU tests ─────────────────────────────────────────────────────────

func TestCreateVariant_success(t *testing.T) {
	s := &stubs{createVariant: stubCreateVariant{out: &pmsDomain.Variant{ID: "v1", ProductID: "p1", Name: "Large"}}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/products/p1/variants", map[string]string{"name": "Large"})
	require.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Large")
}

func TestCreateVariant_productNotFound(t *testing.T) {
	s := &stubs{createVariant: stubCreateVariant{err: pmsDomain.ErrProductNotFound}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/products/missing/variants", map[string]string{"name": "Large"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateVariant_success(t *testing.T) {
	mediaID := "m1"
	s := &stubs{updateVariant: stubUpdateVariant{out: &pmsDomain.Variant{ID: "v1", ProductID: "p1", Name: "Large", FeaturedMediaID: &mediaID}}}
	w := doRequest(newTestRouter(s), http.MethodPatch, "/api/v1/pms/variants/v1", map[string]any{
		"name": "Large", "featured_media_id": "m1",
	})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "m1")
}

func TestUpdateVariant_featuredMediaProductMismatchReturns422(t *testing.T) {
	s := &stubs{updateVariant: stubUpdateVariant{err: pmsDomain.ErrFeaturedMediaProductMismatch}}
	w := doRequest(newTestRouter(s), http.MethodPatch, "/api/v1/pms/variants/v1", map[string]any{
		"name": "Large", "featured_media_id": "m1",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateMedia_success(t *testing.T) {
	s := &stubs{updateMedia: stubUpdateMedia{out: &pmsDomain.Media{ID: "m1", AltText: "Front view"}}}
	w := doRequest(newTestRouter(s), http.MethodPatch, "/api/v1/pms/media/m1", map[string]any{
		"alt_text": "Front view",
	})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Front view")
}

func TestUpdateMedia_notFound(t *testing.T) {
	s := &stubs{updateMedia: stubUpdateMedia{err: pmsDomain.ErrMediaNotFound}}
	w := doRequest(newTestRouter(s), http.MethodPatch, "/api/v1/pms/media/missing", map[string]any{
		"alt_text": "Front view",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateSKU_duplicateCodeReturns409(t *testing.T) {
	s := &stubs{createSKU: stubCreateSKU{err: pmsDomain.ErrDuplicateSKUCode}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/variants/v1/skus", map[string]any{
		"sku_code": "WID-L", "price_minor_units": 1999, "currency_code": "USD", "status": "active",
	})
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestGetProduct_returnsSKUWooTaxProvenance(t *testing.T) {
	wooSlug, wooStatus := "reduced-rate", "taxable"
	sku := pmsDomain.SKU{
		ID: "s1", SKUCode: "WID-L", PriceMinorUnits: 2499, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
		WooTaxClassSlug: &wooSlug, WooTaxStatus: &wooStatus,
	}
	s := &stubs{getProduct: stubGetProduct{out: &apppms.ProductDetail{
		Product:  pmsDomain.Product{ID: "p1", Name: "Widget", Status: pmsDomain.ProductStatusActive},
		Variants: []apppms.VariantDetail{{Variant: pmsDomain.Variant{ID: "v1", ProductID: "p1"}, SKUs: []pmsDomain.SKU{sku}}},
	}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/products/p1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "reduced-rate")
	assert.Contains(t, w.Body.String(), "taxable")
}

func TestUpdateSKU_success(t *testing.T) {
	s := &stubs{updateSKU: stubUpdateSKU{out: &pmsDomain.SKU{ID: "s1", SKUCode: "WID-L", PriceMinorUnits: 2499, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive}}}
	w := doRequest(newTestRouter(s), http.MethodPatch, "/api/v1/pms/skus/s1", map[string]any{
		"price_minor_units": 2499, "currency_code": "USD", "status": "active",
	})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "2499")
}

func TestCreateSKU_passesTaxClassIdThrough(t *testing.T) {
	taxClassID := "class-1"
	s := &stubs{createSKU: stubCreateSKU{out: &pmsDomain.SKU{ID: "s1", SKUCode: "WID-L", PriceMinorUnits: 1999, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, TaxClassID: &taxClassID}}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/variants/v1/skus", map[string]any{
		"sku_code": "WID-L", "price_minor_units": 1999, "currency_code": "USD", "status": "active", "tax_class_id": "class-1",
	})
	require.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, s.createSKU.gotInput.TaxClassID)
	assert.Equal(t, "class-1", *s.createSKU.gotInput.TaxClassID)
	assert.Contains(t, w.Body.String(), "class-1")
}

func TestUpdateSKU_passesTaxClassIdThrough(t *testing.T) {
	taxClassID := "class-2"
	s := &stubs{updateSKU: stubUpdateSKU{out: &pmsDomain.SKU{ID: "s1", SKUCode: "WID-L", PriceMinorUnits: 2499, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive, TaxClassID: &taxClassID}}}
	w := doRequest(newTestRouter(s), http.MethodPatch, "/api/v1/pms/skus/s1", map[string]any{
		"price_minor_units": 2499, "currency_code": "USD", "status": "active", "tax_class_id": "class-2",
	})
	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, s.updateSKU.gotInput.TaxClassID)
	assert.Equal(t, "class-2", *s.updateSKU.gotInput.TaxClassID)
	assert.Contains(t, w.Body.String(), "class-2")
}

func TestUpdateSKU_passesSKUCodeThrough(t *testing.T) {
	s := &stubs{updateSKU: stubUpdateSKU{out: &pmsDomain.SKU{ID: "s1", SKUCode: "WID-L-NEW", PriceMinorUnits: 2499, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive}}}
	w := doRequest(newTestRouter(s), http.MethodPatch, "/api/v1/pms/skus/s1", map[string]any{
		"sku_code": "WID-L-NEW", "price_minor_units": 2499, "currency_code": "USD", "status": "active",
	})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "WID-L-NEW", s.updateSKU.gotInput.SKUCode)
	assert.Contains(t, w.Body.String(), "WID-L-NEW")
}

func TestCreateSKU_passesPricingModeAndScaleCodeThrough(t *testing.T) {
	pricePerKilo := int64(4999)
	scaleCode := "94721"
	s := &stubs{createSKU: stubCreateSKU{out: &pmsDomain.SKU{
		ID: "s1", SKUCode: "DRYMEAT-1", PriceMinorUnits: 0, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
		PricingMode: pmsDomain.SKUPricingModePerWeight, PricePerKiloMinorUnits: &pricePerKilo, ScaleCode: &scaleCode,
	}}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/variants/v1/skus", map[string]any{
		"sku_code": "DRYMEAT-1", "price_minor_units": 0, "currency_code": "USD", "status": "active",
		"pricing_mode": "per_weight", "price_per_kilo_minor_units": 4999, "scale_code": "94721",
	})
	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, pmsDomain.SKUPricingModePerWeight, s.createSKU.gotInput.PricingMode)
	require.NotNil(t, s.createSKU.gotInput.PricePerKiloMinorUnits)
	assert.Equal(t, int64(4999), *s.createSKU.gotInput.PricePerKiloMinorUnits)
	require.NotNil(t, s.createSKU.gotInput.ScaleCode)
	assert.Equal(t, "94721", *s.createSKU.gotInput.ScaleCode)
	assert.Contains(t, w.Body.String(), `"pricing_mode":"per_weight"`)
	assert.Contains(t, w.Body.String(), `"scale_code":"94721"`)
}

func TestCreateSKU_rejectsPerWeightWithoutPricePerKilo(t *testing.T) {
	s := &stubs{createSKU: stubCreateSKU{err: pmsDomain.ErrPricePerKiloRequiredForPerWeight}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/variants/v1/skus", map[string]any{
		"sku_code": "DRYMEAT-1", "price_minor_units": 0, "currency_code": "USD", "status": "active",
		"pricing_mode": "per_weight",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateSKU_duplicateSKUCodeReturns409(t *testing.T) {
	s := &stubs{updateSKU: stubUpdateSKU{err: pmsDomain.ErrDuplicateSKUCode}}
	w := doRequest(newTestRouter(s), http.MethodPatch, "/api/v1/pms/skus/s1", map[string]any{
		"sku_code": "TAKEN", "price_minor_units": 2499, "currency_code": "USD", "status": "active",
	})
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestDeleteSKU_notFound(t *testing.T) {
	s := &stubs{deleteSKU: stubDeleteSKU{err: pmsDomain.ErrSKUNotFound}}
	w := doRequest(newTestRouter(s), http.MethodDelete, "/api/v1/pms/skus/missing", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetManualCostPrice_success(t *testing.T) {
	costPrice := int64(1250)
	s := &stubs{setManualCostPrice: stubSetManualCostPrice{out: &pmsDomain.SKU{
		ID: "s1", SKUCode: "WID-L", PriceMinorUnits: 2499, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
		CostPriceMinorUnits: &costPrice, CostPriceSource: pmsDomain.SKUCostPriceSourceManual,
	}}}
	w := doRequest(newTestRouter(s), http.MethodPut, "/api/v1/pms/skus/s1/cost-price", map[string]any{
		"cost_price_minor_units": 1250,
	})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "s1", s.setManualCostPrice.gotSKUID)
	assert.Equal(t, int64(1250), s.setManualCostPrice.gotCostPriceMinorUnits)
	assert.Contains(t, w.Body.String(), `"cost_price_source":"manual"`)
}

func TestSetManualCostPrice_notFound(t *testing.T) {
	s := &stubs{setManualCostPrice: stubSetManualCostPrice{err: pmsDomain.ErrSKUNotFound}}
	w := doRequest(newTestRouter(s), http.MethodPut, "/api/v1/pms/skus/missing/cost-price", map[string]any{
		"cost_price_minor_units": 1250,
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestClearCostPriceOverride_success(t *testing.T) {
	costPrice := int64(900)
	s := &stubs{clearCostPriceOverride: stubClearCostPriceOverride{out: &pmsDomain.SKU{
		ID: "s1", SKUCode: "WID-L", PriceMinorUnits: 2499, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
		CostPriceMinorUnits: &costPrice, CostPriceSource: pmsDomain.SKUCostPriceSourceComputed,
	}}}
	w := doRequest(newTestRouter(s), http.MethodDelete, "/api/v1/pms/skus/s1/cost-price", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "s1", s.clearCostPriceOverride.gotSKUID)
	assert.Contains(t, w.Body.String(), `"cost_price_source":"computed"`)
}

func TestClearCostPriceOverride_notFound(t *testing.T) {
	s := &stubs{clearCostPriceOverride: stubClearCostPriceOverride{err: pmsDomain.ErrSKUNotFound}}
	w := doRequest(newTestRouter(s), http.MethodDelete, "/api/v1/pms/skus/missing/cost-price", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCorrectCost_success(t *testing.T) {
	lastReceipt, newAverage := int64(320), int64(350)
	s := &stubs{correctCost: stubCorrectCost{out: &pmsDomain.SKU{
		ID: "s1", SKUCode: "WID-L", PriceMinorUnits: 2499, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive,
		CostPriceMinorUnits: &lastReceipt, InventoryCostMinorUnits: &newAverage, CostPriceSource: pmsDomain.SKUCostPriceSourceComputed,
	}}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/skus/s1/cost-price/correction", map[string]any{
		"adjustment_value_minor_units": 5000, "reference_type": "purchase_order_line_item", "reference_id": "line-1",
	})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "s1", s.correctCost.gotIn.SKUID)
	assert.Equal(t, int64(5000), s.correctCost.gotIn.AdjustmentValueMinorUnits)
	assert.Equal(t, "purchase_order_line_item", s.correctCost.gotIn.ReferenceType)
	assert.Equal(t, "line-1", s.correctCost.gotIn.ReferenceID)
	assert.Contains(t, w.Body.String(), `"cost_price_minor_units":320`)
	assert.Contains(t, w.Body.String(), `"inventory_cost_minor_units":350`)
}

func TestCorrectCost_unprocessableWhenCostUnknown(t *testing.T) {
	s := &stubs{correctCost: stubCorrectCost{err: pmsDomain.ErrCostPriceUnknownForCorrection}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/skus/s1/cost-price/correction", map[string]any{
		"adjustment_value_minor_units": 5000,
	})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCorrectCost_unprocessableWhenNoStockOnHand(t *testing.T) {
	s := &stubs{correctCost: stubCorrectCost{err: pmsDomain.ErrNoStockOnHandForCorrection}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/skus/s1/cost-price/correction", map[string]any{
		"adjustment_value_minor_units": 5000,
	})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestResolveBarcode_success(t *testing.T) {
	s := &stubs{resolveBarcode: stubResolveBarcode{out: &apppms.ResolveBarcodeResult{
		SKU:          pmsDomain.SKU{ID: "s1", SKUCode: "WID-L", PriceMinorUnits: 2499, CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive},
		UnitsPerScan: 1,
		ProductName:  "Widget",
	}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/skus/barcode-lookup/012345", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "012345", s.resolveBarcode.gotInput)
	assert.Contains(t, w.Body.String(), "WID-L")
	assert.Contains(t, w.Body.String(), `"units_per_scan":1`)
	assert.Contains(t, w.Body.String(), `"product_name":"Widget"`)
}

func TestResolveBarcode_includesWeightAndLineTotalForWeightEmbeddedScan(t *testing.T) {
	weightGrams := int64(462)
	lineTotal := int64(2310)
	s := &stubs{resolveBarcode: stubResolveBarcode{out: &apppms.ResolveBarcodeResult{
		SKU:                 pmsDomain.SKU{ID: "s1", SKUCode: "DRYMEAT-1", CurrencyCode: "USD", Status: pmsDomain.SKUStatusActive},
		UnitsPerScan:        1,
		ProductName:         "Dry Meat",
		WeightGrams:         &weightGrams,
		LineTotalMinorUnits: &lineTotal,
	}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/skus/barcode-lookup/2994721004622", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"weight_grams":462`)
	assert.Contains(t, w.Body.String(), `"line_total_minor_units":2310`)
}

func TestResolveBarcode_corruptedWeightEmbeddedCheckDigitReturns422(t *testing.T) {
	s := &stubs{resolveBarcode: stubResolveBarcode{err: pmsDomain.ErrInvalidCheckDigit}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/skus/barcode-lookup/2900633003339", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreateBarcodeAlias_defaultsUnitsPerScanAndReturns201(t *testing.T) {
	s := &stubs{createBarcodeAlias: stubCreateBarcodeAlias{out: &pmsDomain.BarcodeAlias{ID: "alias-1", SKUID: "sku-1", Code: "0630509892495", UnitsPerScan: 1}}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/skus/sku-1/barcode-aliases", map[string]any{
		"code": "0630509892495",
	})
	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "sku-1", s.createBarcodeAlias.gotInput.SKUID)
	assert.Equal(t, "0630509892495", s.createBarcodeAlias.gotInput.Code)
	assert.Contains(t, w.Body.String(), `"units_per_scan":1`)
}

func TestCreateBarcodeAlias_duplicateReturns409(t *testing.T) {
	s := &stubs{createBarcodeAlias: stubCreateBarcodeAlias{err: pmsDomain.ErrDuplicateBarcodeAliasCode}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/skus/sku-1/barcode-aliases", map[string]any{"code": "TAKEN"})
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestListBarcodeAliases_success(t *testing.T) {
	s := &stubs{listBarcodeAliases: stubListBarcodeAliases{out: []pmsDomain.BarcodeAlias{{ID: "a1", Code: "111"}, {ID: "a2", Code: "222"}}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/skus/sku-1/barcode-aliases", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "sku-1", s.listBarcodeAliases.gotSKUID)
	assert.Contains(t, w.Body.String(), `"111"`)
	assert.Contains(t, w.Body.String(), `"222"`)
}

func TestDeleteBarcodeAlias_success(t *testing.T) {
	s := &stubs{deleteBarcodeAlias: stubDeleteBarcodeAlias{}}
	w := doRequest(newTestRouter(s), http.MethodDelete, "/api/v1/pms/barcode-aliases/alias-1", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "alias-1", s.deleteBarcodeAlias.gotID)
}

func TestCreateScaleCodeAlias_returns201(t *testing.T) {
	s := &stubs{createScaleCodeAlias: stubCreateScaleCodeAlias{out: &pmsDomain.ScaleCodeAlias{ID: "sa-1", SKUID: "sku-1", Code: "94722"}}}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/skus/sku-1/scale-code-aliases", map[string]any{"code": "94722"})
	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "sku-1", s.createScaleCodeAlias.gotInput.SKUID)
	assert.Equal(t, "94722", s.createScaleCodeAlias.gotInput.Code)
}

func TestListScaleCodeAliases_success(t *testing.T) {
	s := &stubs{listScaleCodeAliases: stubListScaleCodeAliases{out: []pmsDomain.ScaleCodeAlias{{ID: "sa-1", Code: "94722"}}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/skus/sku-1/scale-code-aliases", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"94722"`)
}

func TestDeleteScaleCodeAlias_success(t *testing.T) {
	s := &stubs{deleteScaleCodeAlias: stubDeleteScaleCodeAlias{}}
	w := doRequest(newTestRouter(s), http.MethodDelete, "/api/v1/pms/scale-code-aliases/sa-1", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "sa-1", s.deleteScaleCodeAlias.gotID)
}

func TestResolveBarcode_notFound(t *testing.T) {
	s := &stubs{resolveBarcode: stubResolveBarcode{err: pmsDomain.ErrSKUNotFound}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/skus/barcode-lookup/UNKNOWN", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetSKUMarginInsight_success(t *testing.T) {
	s := &stubs{getSKUMarginInsight: stubGetSKUMarginInsight{out: &apppms.GetSKUMarginInsightResult{
		CostPriceAvailable: true, CostPriceMinorUnits: 700, RecommendedPriceMinorUnits: 1000,
		ProfitAtPriceMinorUnits: 300, MarginAtPriceBasisPoints: 3000, TotalOnHandQuantity: 5,
	}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/skus/sku-1/margin-insight", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"cost_price_minor_units":700`)
	assert.Contains(t, w.Body.String(), `"recommended_price_minor_units":1000`)
}

func TestGetSKUMarginInsight_notFound(t *testing.T) {
	s := &stubs{getSKUMarginInsight: stubGetSKUMarginInsight{err: pmsDomain.ErrSKUNotFound}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/skus/missing/margin-insight", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListSKUCostPriceHistory_success(t *testing.T) {
	s := &stubs{listCostPriceHistory: stubListCostPriceHistory{out: []pmsDomain.SKUCostPriceHistory{
		{ID: "h1", SKUID: "sku-1", CostPriceMinorUnits: 700, RecommendedPriceMinorUnits: 1000},
	}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/skus/sku-1/cost-price-history", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"cost_price_minor_units":700`)
}

func TestListSKUPriceHistory_success(t *testing.T) {
	salePrice := int64(799)
	s := &stubs{listSKUPriceHistory: stubListSKUPriceHistory{out: []pmsDomain.SKUPriceHistory{
		{ID: "h1", SKUID: "sku-1", PriceMinorUnits: 999, SalePriceMinorUnits: &salePrice, CurrencyCode: "EUR", Reason: pmsDomain.PriceChangeReasonManualUpdate},
	}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/skus/sku-1/price-history", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"price_minor_units":999`)
	assert.Contains(t, w.Body.String(), `"reason":"manual_update"`)
}

func TestListSKUCurrencyPrices_success(t *testing.T) {
	s := &stubs{listSKUCurrencyPrices: stubListSKUCurrencyPrices{out: []pmsDomain.SKUCurrencyPrice{
		{SKUID: "sku-1", CurrencyCode: "USD", PriceMinorUnits: 1200},
	}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/skus/sku-1/currency-prices", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"currency_code":"USD"`)
	assert.Contains(t, w.Body.String(), `"price_minor_units":1200`)
}

func TestUpsertSKUCurrencyPrice_success(t *testing.T) {
	s := &stubs{}
	w := doRequest(newTestRouter(s), http.MethodPut, "/api/v1/pms/skus/sku-1/currency-prices", map[string]any{
		"currency_code": "USD", "price_minor_units": 1200,
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "sku-1", s.upsertSKUCurrencyPrice.gotInput.SKUID)
	assert.Equal(t, "USD", s.upsertSKUCurrencyPrice.gotInput.CurrencyCode)
	assert.Equal(t, int64(1200), s.upsertSKUCurrencyPrice.gotInput.PriceMinorUnits)
}

func TestUpsertSKUCurrencyPrice_validationError(t *testing.T) {
	s := &stubs{upsertSKUCurrencyPrice: stubUpsertSKUCurrencyPrice{err: fxDomain.ErrInvalidCurrencyCode}}
	w := doRequest(newTestRouter(s), http.MethodPut, "/api/v1/pms/skus/sku-1/currency-prices", map[string]any{
		"currency_code": "us", "price_minor_units": 1200,
	})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDeleteSKUCurrencyPrice_success(t *testing.T) {
	s := &stubs{}
	w := doRequest(newTestRouter(s), http.MethodDelete, "/api/v1/pms/skus/sku-1/currency-prices/USD", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "sku-1", s.deleteSKUCurrencyPrice.gotSKUID)
	assert.Equal(t, "USD", s.deleteSKUCurrencyPrice.gotCurrency)
}

func TestListProductHistory_success(t *testing.T) {
	s := &stubs{listProductHistory: stubListProductHistory{out: []pmsDomain.ProductHistory{
		{ID: "h1", EntityType: pmsDomain.ProductHistoryEntityProduct, EntityID: "p1", Field: pmsDomain.ProductHistoryFieldName, OldValue: "Old", NewValue: "New"},
	}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/products/p1/history", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"old_value":"Old"`)
	assert.Contains(t, w.Body.String(), `"field":"name"`)
}

func TestListVariantHistory_success(t *testing.T) {
	s := &stubs{listProductHistory: stubListProductHistory{out: []pmsDomain.ProductHistory{
		{ID: "h1", EntityType: pmsDomain.ProductHistoryEntityVariant, EntityID: "v1", Field: pmsDomain.ProductHistoryFieldName, OldValue: "Old", NewValue: "New"},
	}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/variants/v1/history", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"entity_type":"variant"`)
}

func TestListMarginReport_success(t *testing.T) {
	s := &stubs{listMarginReport: stubListMarginReport{out: &apppms.ListMarginReportResult{
		Rows:  []apppms.MarginReportRow{{SKUID: "sku-1", SKUCode: "WID-L", CostPriceMinorUnits: 700, PriceMinorUnits: 1000, ProfitMinorUnits: 300, MarginBasisPoints: 3000, BelowTarget: false}},
		Total: 1,
	}}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/margin-report", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"sku_code":"WID-L"`)
	assert.Contains(t, w.Body.String(), `"total":1`)
}

func TestDeleteProduct_blockedByHistoryIsConflictWithCode(t *testing.T) {
	s := &stubs{deleteProduct: stubDeleteProduct{err: pmsDomain.ErrProductHasReceivingHistory}}
	w := doRequest(newTestRouter(s), http.MethodDelete, "/api/v1/pms/products/p1", nil)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), `"code":"PRODUCT_HAS_RECEIVING_HISTORY"`)
	assert.Contains(t, w.Body.String(), "receiving history")
}

func TestGetProduct_namesTheLinkedCaseAndPortionSKUs(t *testing.T) {
	unit := "s-unit"
	detail := &apppms.ProductDetail{
		Product: pmsDomain.Product{ID: "p1", Name: "Caixa Refresco", Status: pmsDomain.ProductStatusActive},
		Variants: []apppms.VariantDetail{{
			Variant: pmsDomain.Variant{ID: "v1"},
			SKUs: []pmsDomain.SKU{
				{ID: "s-case", SKUCode: "CX-30", CaseOfSKUID: &unit, CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive},
				{ID: "s-portion", SKUCode: "P-1", PortionedFromSKUID: &unit, CurrencyCode: "EUR", Status: pmsDomain.SKUStatusActive},
			},
		}},
		LinkedSKUs: map[string]apppms.LinkedSKU{
			"s-unit": {ID: "s-unit", ProductName: "Refresco em pó Uva", SKUCode: "REF-UVA", Barcode: "7891132082346"},
		},
	}
	s := &stubs{getProduct: stubGetProduct{out: detail}}

	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/products/p1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	summary := `{"barcode":"7891132082346","id":"s-unit","product_name":"Refresco em pó Uva","sku_code":"REF-UVA"}`
	assert.Contains(t, w.Body.String(), `"case_of_sku":`+summary)
	assert.Contains(t, w.Body.String(), `"portioned_from_sku":`+summary)
}
