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
	out *pmsDomain.Product
	err error
}

func (s *stubUpdateProduct) Execute(_ context.Context, _ apppms.UpdateProductInput) (*pmsDomain.Product, error) {
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

// ── Helpers ───────────────────────────────────────────────────────────────────

type stubs struct {
	listCategories   stubListCategories
	createCategory   stubCreateCategory
	updateCategory   stubUpdateCategory
	deleteCategory   stubDeleteCategory
	listProducts     stubListProducts
	createProduct    stubCreateProduct
	getProduct       stubGetProduct
	updateProduct    stubUpdateProduct
	deleteProduct    stubDeleteProduct
	createVariant    stubCreateVariant
	updateVariant    stubUpdateVariant
	deleteVariant    stubDeleteVariant
	createSKU        stubCreateSKU
	updateSKU        stubUpdateSKU
	deleteSKU        stubDeleteSKU
	uploadMedia      stubUploadMedia
	listProductMedia stubListProductMedia
	deleteMedia      stubDeleteMedia
	updateMedia      stubUpdateMedia
	resolveBarcode   stubResolveBarcode
}

func newTestRouter(s *stubs) *gin.Engine {
	h := pmshandler.NewHandler(
		&s.listCategories, &s.createCategory, &s.updateCategory, &s.deleteCategory,
		&s.listProducts, &s.createProduct, &s.getProduct, &s.updateProduct, &s.deleteProduct,
		&s.createVariant, &s.updateVariant, &s.deleteVariant,
		&s.createSKU, &s.updateSKU, &s.deleteSKU,
		&s.uploadMedia, &s.listProductMedia, &s.deleteMedia, &s.updateMedia,
		&s.resolveBarcode,
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

func TestUpdateProduct_success(t *testing.T) {
	s := &stubs{updateProduct: stubUpdateProduct{out: &pmsDomain.Product{ID: "p1", Name: "Widget v2", Status: pmsDomain.ProductStatusActive}}}
	w := doRequest(newTestRouter(s), http.MethodPatch, "/api/v1/pms/products/p1", map[string]any{
		"name": "Widget v2", "status": "active",
	})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Widget v2")
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

func TestDeleteSKU_notFound(t *testing.T) {
	s := &stubs{deleteSKU: stubDeleteSKU{err: pmsDomain.ErrSKUNotFound}}
	w := doRequest(newTestRouter(s), http.MethodDelete, "/api/v1/pms/skus/missing", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
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

func TestResolveBarcode_notFound(t *testing.T) {
	s := &stubs{resolveBarcode: stubResolveBarcode{err: pmsDomain.ErrSKUNotFound}}
	w := doRequest(newTestRouter(s), http.MethodGet, "/api/v1/pms/skus/barcode-lookup/UNKNOWN", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
