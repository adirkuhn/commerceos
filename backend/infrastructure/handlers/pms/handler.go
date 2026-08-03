package pms

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ── Use-case interfaces (defined at point of use — ISP) ──────────────────────

type listCategoriesUC interface {
	Execute(ctx context.Context) ([]pmsDomain.Category, error)
}
type createCategoryUC interface {
	Execute(ctx context.Context, in apppms.CreateCategoryInput) (*pmsDomain.Category, error)
}
type updateCategoryUC interface {
	Execute(ctx context.Context, in apppms.UpdateCategoryInput) (*pmsDomain.Category, error)
}
type deleteCategoryUC interface {
	Execute(ctx context.Context, id string) error
}
type listProductsUC interface {
	Execute(ctx context.Context, in apppms.ListProductsInput) (*apppms.ListProductsResult, error)
}
type createProductUC interface {
	Execute(ctx context.Context, in apppms.CreateProductInput) (*pmsDomain.Product, error)
}
type getProductUC interface {
	Execute(ctx context.Context, id string) (*apppms.ProductDetail, error)
}
type updateProductUC interface {
	Execute(ctx context.Context, in apppms.UpdateProductInput) (*pmsDomain.Product, error)
}
type deleteProductUC interface {
	Execute(ctx context.Context, id string) error
}
type createVariantUC interface {
	Execute(ctx context.Context, in apppms.CreateVariantInput) (*pmsDomain.Variant, error)
}
type updateVariantUC interface {
	Execute(ctx context.Context, in apppms.UpdateVariantInput) (*pmsDomain.Variant, error)
}
type deleteVariantUC interface {
	Execute(ctx context.Context, id string) error
}
type createSKUUC interface {
	Execute(ctx context.Context, in apppms.CreateSKUInput) (*pmsDomain.SKU, error)
}
type updateSKUUC interface {
	Execute(ctx context.Context, in apppms.UpdateSKUInput) (*pmsDomain.SKU, error)
}
type deleteSKUUC interface {
	Execute(ctx context.Context, id string) error
}
type uploadMediaUC interface {
	Execute(ctx context.Context, in apppms.UploadMediaInput) (*pmsDomain.Media, error)
}
type listProductMediaUC interface {
	Execute(ctx context.Context, productID string) ([]pmsDomain.Media, error)
}
type deleteMediaUC interface {
	Execute(ctx context.Context, id string) error
}
type updateMediaUC interface {
	Execute(ctx context.Context, in apppms.UpdateMediaInput) (*pmsDomain.Media, error)
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listCategories   listCategoriesUC
	createCategory   createCategoryUC
	updateCategory   updateCategoryUC
	deleteCategory   deleteCategoryUC
	listProducts     listProductsUC
	createProduct    createProductUC
	getProduct       getProductUC
	updateProduct    updateProductUC
	deleteProduct    deleteProductUC
	createVariant    createVariantUC
	updateVariant    updateVariantUC
	deleteVariant    deleteVariantUC
	createSKU        createSKUUC
	updateSKU        updateSKUUC
	deleteSKU        deleteSKUUC
	uploadMedia      uploadMediaUC
	listProductMedia listProductMediaUC
	deleteMedia      deleteMediaUC
	updateMedia      updateMediaUC
}

// NewHandler constructs a Handler with injected use cases (useful for testing).
func NewHandler(
	listCategories listCategoriesUC,
	createCategory createCategoryUC,
	updateCategory updateCategoryUC,
	deleteCategory deleteCategoryUC,
	listProducts listProductsUC,
	createProduct createProductUC,
	getProduct getProductUC,
	updateProduct updateProductUC,
	deleteProduct deleteProductUC,
	createVariant createVariantUC,
	updateVariant updateVariantUC,
	deleteVariant deleteVariantUC,
	createSKU createSKUUC,
	updateSKU updateSKUUC,
	deleteSKU deleteSKUUC,
	uploadMedia uploadMediaUC,
	listProductMedia listProductMediaUC,
	deleteMedia deleteMediaUC,
	updateMedia updateMediaUC,
) *Handler {
	return &Handler{
		listCategories:   listCategories,
		createCategory:   createCategory,
		updateCategory:   updateCategory,
		deleteCategory:   deleteCategory,
		listProducts:     listProducts,
		createProduct:    createProduct,
		getProduct:       getProduct,
		updateProduct:    updateProduct,
		deleteProduct:    deleteProduct,
		createVariant:    createVariant,
		updateVariant:    updateVariant,
		deleteVariant:    deleteVariant,
		createSKU:        createSKU,
		updateSKU:        updateSKU,
		deleteSKU:        deleteSKU,
		uploadMedia:      uploadMedia,
		listProductMedia: listProductMedia,
		deleteMedia:      deleteMedia,
		updateMedia:      updateMedia,
	}
}

var _ genpms.ServerInterface = (*Handler)(nil)

// Register is the production wiring: creates repos + use cases, then
// mounts the generated routes (matching the OpenAPI spec exactly).
// fileStorage is the media storage adapter (local disk or S3/MinIO — see
// infrastructure/storage), selected by the caller based on config.
func Register(rg gin.IRouter, db *gorm.DB, fileStorage pmsDomain.FileStorage) {
	categoryRepo := repositories.NewCategoryRepo(db)
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	mediaRepo := repositories.NewMediaRepo(db)

	h := NewHandler(
		apppms.NewListCategoriesUseCase(db, categoryRepo),
		apppms.NewCreateCategoryUseCase(db, categoryRepo),
		apppms.NewUpdateCategoryUseCase(db, categoryRepo),
		apppms.NewDeleteCategoryUseCase(db, categoryRepo),
		apppms.NewListProductsUseCase(db, productRepo),
		apppms.NewCreateProductUseCase(db, productRepo),
		apppms.NewGetProductUseCase(db, productRepo, variantRepo, skuRepo),
		apppms.NewUpdateProductUseCase(db, productRepo, variantRepo),
		apppms.NewDeleteProductUseCase(db, productRepo),
		apppms.NewCreateVariantUseCase(db, productRepo, variantRepo),
		apppms.NewUpdateVariantUseCase(db, variantRepo, mediaRepo),
		apppms.NewDeleteVariantUseCase(db, variantRepo),
		apppms.NewCreateSKUUseCase(db, variantRepo, skuRepo),
		apppms.NewUpdateSKUUseCase(db, skuRepo),
		apppms.NewDeleteSKUUseCase(db, skuRepo),
		apppms.NewUploadMediaUseCase(db, productRepo, variantRepo, mediaRepo, fileStorage),
		apppms.NewListProductMediaUseCase(db, mediaRepo),
		apppms.NewDeleteMediaUseCase(db, mediaRepo, fileStorage),
		apppms.NewUpdateMediaUseCase(db, mediaRepo),
	)
	genpms.RegisterHandlers(rg, h)
}

// ── Category handlers ─────────────────────────────────────────────────────────

func (h *Handler) ListCategories(c *gin.Context) {
	categories, err := h.listCategories.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genpms.Category, len(categories))
	for i, cat := range categories {
		resp[i] = toGenCategory(cat)
	}
	c.JSON(http.StatusOK, genpms.CategoryListResponse{Data: &resp})
}

func (h *Handler) CreateCategory(c *gin.Context) {
	var req genpms.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	category, err := h.createCategory.Execute(c.Request.Context(), apppms.CreateCategoryInput{
		Name:     req.Name,
		ParentID: req.ParentId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenCategory(*category)
	c.JSON(http.StatusCreated, genpms.CategoryResponse{Data: &resp})
}

func (h *Handler) UpdateCategory(c *gin.Context, id genpms.Id) {
	var req genpms.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	category, err := h.updateCategory.Execute(c.Request.Context(), apppms.UpdateCategoryInput{
		ID:       id,
		Name:     req.Name,
		ParentID: req.ParentId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenCategory(*category)
	c.JSON(http.StatusOK, genpms.CategoryResponse{Data: &resp})
}

func (h *Handler) DeleteCategory(c *gin.Context, id genpms.Id) {
	err := h.deleteCategory.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.EmptyResponse{})
}

// ── Product handlers ─────────────────────────────────────────────────────────

func (h *Handler) ListProducts(c *gin.Context, params genpms.ListProductsParams) {
	in := apppms.ListProductsInput{}
	if params.Page != nil {
		in.Page = *params.Page
	}
	if params.PerPage != nil {
		in.PerPage = *params.PerPage
	}
	if params.Search != nil {
		in.Search = *params.Search
	}
	result, err := h.listProducts.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genpms.Product, len(result.Products))
	for i, p := range result.Products {
		resp[i] = toGenProduct(p)
	}
	total := int(result.Total)
	c.JSON(http.StatusOK, genpms.ProductListResponse{Data: &resp, Total: &total})
}

func (h *Handler) CreateProduct(c *gin.Context) {
	var req genpms.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	product, err := h.createProduct.Execute(c.Request.Context(), apppms.CreateProductInput{
		Name:        req.Name,
		Description: description,
		CategoryID:  req.CategoryId,
		Status:      pmsDomain.ProductStatus(req.Status),
		HasVariants: req.HasVariants != nil && *req.HasVariants,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenProduct(*product)
	c.JSON(http.StatusCreated, genpms.ProductResponse{Data: &resp})
}

func (h *Handler) GetProduct(c *gin.Context, id genpms.Id) {
	detail, err := h.getProduct.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenProductDetail(*detail)
	c.JSON(http.StatusOK, genpms.ProductDetailResponse{Data: &resp})
}

func (h *Handler) UpdateProduct(c *gin.Context, id genpms.Id) {
	var req genpms.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	product, err := h.updateProduct.Execute(c.Request.Context(), apppms.UpdateProductInput{
		ID:          id,
		Name:        req.Name,
		Description: description,
		CategoryID:  req.CategoryId,
		Status:      pmsDomain.ProductStatus(req.Status),
		HasVariants: req.HasVariants != nil && *req.HasVariants,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenProduct(*product)
	c.JSON(http.StatusOK, genpms.ProductResponse{Data: &resp})
}

func (h *Handler) DeleteProduct(c *gin.Context, id genpms.Id) {
	err := h.deleteProduct.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.EmptyResponse{})
}

// ── Variant handlers ─────────────────────────────────────────────────────────

func (h *Handler) CreateVariant(c *gin.Context, id genpms.Id) {
	var req genpms.CreateVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	position := 0
	if req.Position != nil {
		position = *req.Position
	}
	variant, err := h.createVariant.Execute(c.Request.Context(), apppms.CreateVariantInput{
		ProductID: id,
		Name:      req.Name,
		Position:  position,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenVariant(*variant)
	c.JSON(http.StatusCreated, genpms.VariantResponse{Data: &resp})
}

func (h *Handler) UpdateVariant(c *gin.Context, id genpms.Id) {
	var req genpms.UpdateVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	position := 0
	if req.Position != nil {
		position = *req.Position
	}
	variant, err := h.updateVariant.Execute(c.Request.Context(), apppms.UpdateVariantInput{
		ID:              id,
		Name:            req.Name,
		Position:        position,
		FeaturedMediaID: req.FeaturedMediaId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenVariant(*variant)
	c.JSON(http.StatusOK, genpms.VariantResponse{Data: &resp})
}

func (h *Handler) DeleteVariant(c *gin.Context, id genpms.Id) {
	err := h.deleteVariant.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.EmptyResponse{})
}

// ── SKU handlers ──────────────────────────────────────────────────────────────

func (h *Handler) CreateSKU(c *gin.Context, id genpms.Id) {
	var req genpms.CreateSKURequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	barcode := ""
	if req.Barcode != nil {
		barcode = *req.Barcode
	}
	sku, err := h.createSKU.Execute(c.Request.Context(), apppms.CreateSKUInput{
		VariantID:       id,
		SKUCode:         req.SkuCode,
		Barcode:         barcode,
		PriceMinorUnits: req.PriceMinorUnits,
		CurrencyCode:    req.CurrencyCode,
		WeightGrams:     req.WeightGrams,
		LengthMM:        req.LengthMm,
		WidthMM:         req.WidthMm,
		HeightMM:        req.HeightMm,
		IsVirtual:       req.IsVirtual != nil && *req.IsVirtual,
		Status:          pmsDomain.SKUStatus(req.Status),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenSKU(*sku)
	c.JSON(http.StatusCreated, genpms.SKUResponse{Data: &resp})
}

func (h *Handler) UpdateSKU(c *gin.Context, id genpms.Id) {
	var req genpms.UpdateSKURequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	barcode := ""
	if req.Barcode != nil {
		barcode = *req.Barcode
	}
	sku, err := h.updateSKU.Execute(c.Request.Context(), apppms.UpdateSKUInput{
		ID:              id,
		Barcode:         barcode,
		PriceMinorUnits: req.PriceMinorUnits,
		CurrencyCode:    req.CurrencyCode,
		WeightGrams:     req.WeightGrams,
		LengthMM:        req.LengthMm,
		WidthMM:         req.WidthMm,
		HeightMM:        req.HeightMm,
		IsVirtual:       req.IsVirtual != nil && *req.IsVirtual,
		Status:          pmsDomain.SKUStatus(req.Status),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenSKU(*sku)
	c.JSON(http.StatusOK, genpms.SKUResponse{Data: &resp})
}

func (h *Handler) DeleteSKU(c *gin.Context, id genpms.Id) {
	err := h.deleteSKU.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.EmptyResponse{})
}

// ── Media handlers ────────────────────────────────────────────────────────────

func (h *Handler) ListProductMedia(c *gin.Context, id genpms.Id) {
	media, err := h.listProductMedia.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genpms.Media, len(media))
	for i, m := range media {
		resp[i] = toGenMedia(m)
	}
	c.JSON(http.StatusOK, genpms.MediaListResponse{Data: &resp})
}

func (h *Handler) UploadProductMedia(c *gin.Context, id genpms.Id) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "file is required")
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	defer f.Close()

	media, err := h.uploadMedia.Execute(c.Request.Context(), apppms.UploadMediaInput{
		ProductID:   &id,
		Filename:    fileHeader.Filename,
		ContentType: fileHeader.Header.Get("Content-Type"),
		Data:        f,
		AltText:     c.PostForm("alt_text"),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenMedia(*media)
	c.JSON(http.StatusCreated, genpms.MediaResponse{Data: &resp})
}

func (h *Handler) UpdateMedia(c *gin.Context, id genpms.Id) {
	var req genpms.UpdateMediaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	media, err := h.updateMedia.Execute(c.Request.Context(), apppms.UpdateMediaInput{
		ID:      id,
		AltText: req.AltText,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenMedia(*media)
	c.JSON(http.StatusOK, genpms.MediaResponse{Data: &resp})
}

func (h *Handler) DeleteMedia(c *gin.Context, id genpms.Id) {
	err := h.deleteMedia.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.EmptyResponse{})
}

// ── Domain → generated-type mappers ─────────────────────────────────────────

func toGenCategory(c pmsDomain.Category) genpms.Category {
	return genpms.Category{
		Id:        c.ID,
		TenantId:  c.TenantID,
		ParentId:  c.ParentID,
		Name:      c.Name,
		Slug:      c.Slug,
		Position:  c.Position,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func toGenProduct(p pmsDomain.Product) genpms.Product {
	return genpms.Product{
		Id:          p.ID,
		TenantId:    p.TenantID,
		CategoryId:  p.CategoryID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		Status:      genpms.ProductStatus(p.Status),
		HasVariants: p.HasVariants,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func toGenVariant(v pmsDomain.Variant) genpms.Variant {
	return genpms.Variant{
		Id:              v.ID,
		TenantId:        v.TenantID,
		ProductId:       v.ProductID,
		Name:            v.Name,
		Position:        v.Position,
		FeaturedMediaId: v.FeaturedMediaID,
		CreatedAt:       v.CreatedAt,
		UpdatedAt:       v.UpdatedAt,
	}
}

func toGenSKU(s pmsDomain.SKU) genpms.SKU {
	var barcode *string
	if s.Barcode != "" {
		barcode = &s.Barcode
	}
	return genpms.SKU{
		Id:              s.ID,
		TenantId:        s.TenantID,
		VariantId:       s.VariantID,
		SkuCode:         s.SKUCode,
		Barcode:         barcode,
		PriceMinorUnits: s.PriceMinorUnits,
		CurrencyCode:    s.CurrencyCode,
		WeightGrams:     s.WeightGrams,
		LengthMm:        s.LengthMM,
		WidthMm:         s.WidthMM,
		HeightMm:        s.HeightMM,
		IsVirtual:       s.IsVirtual,
		Status:          genpms.SKUStatus(s.Status),
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
}

func toGenMedia(m pmsDomain.Media) genpms.Media {
	return genpms.Media{
		Id:        m.ID,
		TenantId:  m.TenantID,
		ProductId: m.ProductID,
		VariantId: m.VariantID,
		Url:       m.URL,
		Type:      genpms.MediaType(m.Type),
		Position:  m.Position,
		AltText:   m.AltText,
		CreatedAt: m.CreatedAt,
	}
}

func toGenProductDetail(d apppms.ProductDetail) genpms.ProductDetail {
	variants := make([]genpms.VariantWithSKUs, len(d.Variants))
	for i, vd := range d.Variants {
		skus := make([]genpms.SKU, len(vd.SKUs))
		for j, s := range vd.SKUs {
			skus[j] = toGenSKU(s)
		}
		variants[i] = genpms.VariantWithSKUs{
			Id:              vd.Variant.ID,
			TenantId:        vd.Variant.TenantID,
			ProductId:       vd.Variant.ProductID,
			Name:            vd.Variant.Name,
			Position:        vd.Variant.Position,
			FeaturedMediaId: vd.Variant.FeaturedMediaID,
			CreatedAt:       vd.Variant.CreatedAt,
			UpdatedAt:       vd.Variant.UpdatedAt,
			Skus:            &skus,
		}
	}
	return genpms.ProductDetail{
		Product:  toGenProduct(d.Product),
		Variants: variants,
	}
}

// ── Error mapping ────────────────────────────────────────────────────────────

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, pmsDomain.ErrProductNotFound),
		errors.Is(err, pmsDomain.ErrCategoryNotFound),
		errors.Is(err, pmsDomain.ErrVariantNotFound),
		errors.Is(err, pmsDomain.ErrSKUNotFound),
		errors.Is(err, pmsDomain.ErrMediaNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, pmsDomain.ErrDuplicateSlug),
		errors.Is(err, pmsDomain.ErrDuplicateSKUCode):
		middleware.Fail(c, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, pmsDomain.ErrVariantsNotEnabled),
		errors.Is(err, pmsDomain.ErrVirtualSKUCannotHaveDimensions),
		errors.Is(err, pmsDomain.ErrUnsupportedMediaType),
		errors.Is(err, pmsDomain.ErrMediaTargetRequired),
		errors.Is(err, pmsDomain.ErrFeaturedMediaProductMismatch):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		log.Printf("pms: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
