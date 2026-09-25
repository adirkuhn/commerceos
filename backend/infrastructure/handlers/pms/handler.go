package pms

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
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
	Execute(ctx context.Context, in apppms.ListProductsInput) (*apppms.ListPOSProductCatalogResult, error)
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
type duplicateProductUC interface {
	Execute(ctx context.Context, sourceProductID string) (*apppms.ProductDetail, error)
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
type setManualCostPriceUC interface {
	Execute(ctx context.Context, skuID string, costPriceMinorUnits int64) (*pmsDomain.SKU, error)
}
type clearCostPriceOverrideUC interface {
	Execute(ctx context.Context, skuID string) (*pmsDomain.SKU, error)
}
type correctCostUC interface {
	Execute(ctx context.Context, in apppms.CorrectCostInput) (*pmsDomain.SKU, error)
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
type resolveBarcodeUC interface {
	Execute(ctx context.Context, code string) (*apppms.ResolveBarcodeResult, error)
}
type getSKUMarginInsightUC interface {
	Execute(ctx context.Context, skuID string) (*apppms.GetSKUMarginInsightResult, error)
}
type listCostPriceHistoryUC interface {
	Execute(ctx context.Context, skuID string, limit int) ([]pmsDomain.SKUCostPriceHistory, error)
}
type listMarginReportUC interface {
	Execute(ctx context.Context, in apppms.ListMarginReportInput) (*apppms.ListMarginReportResult, error)
}
type listSKUPriceHistoryUC interface {
	Execute(ctx context.Context, skuID string, limit int) ([]pmsDomain.SKUPriceHistory, error)
}
type listProductHistoryUC interface {
	Execute(ctx context.Context, entityType pmsDomain.ProductHistoryEntityType, entityID string) ([]pmsDomain.ProductHistory, error)
}
type listSKUCurrencyPricesUC interface {
	Execute(ctx context.Context, skuID string) ([]pmsDomain.SKUCurrencyPrice, error)
}
type upsertSKUCurrencyPriceUC interface {
	Execute(ctx context.Context, in apppms.UpsertSKUCurrencyPriceInput) (*pmsDomain.SKUCurrencyPrice, error)
}
type deleteSKUCurrencyPriceUC interface {
	Execute(ctx context.Context, skuID, currencyCode string) error
}
type listBarcodeAliasesUC interface {
	Execute(ctx context.Context, skuID string) ([]pmsDomain.BarcodeAlias, error)
}
type createBarcodeAliasUC interface {
	Execute(ctx context.Context, in apppms.CreateBarcodeAliasInput) (*pmsDomain.BarcodeAlias, error)
}
type deleteBarcodeAliasUC interface {
	Execute(ctx context.Context, id string) error
}
type listScaleCodeAliasesUC interface {
	Execute(ctx context.Context, skuID string) ([]pmsDomain.ScaleCodeAlias, error)
}
type createScaleCodeAliasUC interface {
	Execute(ctx context.Context, in apppms.CreateScaleCodeAliasInput) (*pmsDomain.ScaleCodeAlias, error)
}
type deleteScaleCodeAliasUC interface {
	Execute(ctx context.Context, id string) error
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listCategories         listCategoriesUC
	createCategory         createCategoryUC
	updateCategory         updateCategoryUC
	deleteCategory         deleteCategoryUC
	listProducts           listProductsUC
	createProduct          createProductUC
	getProduct             getProductUC
	updateProduct          updateProductUC
	duplicateProduct       duplicateProductUC
	deleteProduct          deleteProductUC
	createVariant          createVariantUC
	updateVariant          updateVariantUC
	deleteVariant          deleteVariantUC
	createSKU              createSKUUC
	updateSKU              updateSKUUC
	deleteSKU              deleteSKUUC
	setManualCostPrice     setManualCostPriceUC
	clearCostPriceOverride clearCostPriceOverrideUC
	correctCost            correctCostUC
	uploadMedia            uploadMediaUC
	listProductMedia       listProductMediaUC
	deleteMedia            deleteMediaUC
	updateMedia            updateMediaUC
	resolveBarcode         resolveBarcodeUC
	getSKUMarginInsight    getSKUMarginInsightUC
	listCostPriceHistory   listCostPriceHistoryUC
	listMarginReport       listMarginReportUC
	listSKUPriceHistory    listSKUPriceHistoryUC
	listProductHistory     listProductHistoryUC
	listSKUCurrencyPrices  listSKUCurrencyPricesUC
	upsertSKUCurrencyPrice upsertSKUCurrencyPriceUC
	deleteSKUCurrencyPrice deleteSKUCurrencyPriceUC
	listBarcodeAliases     listBarcodeAliasesUC
	createBarcodeAlias     createBarcodeAliasUC
	deleteBarcodeAlias     deleteBarcodeAliasUC
	listScaleCodeAliases   listScaleCodeAliasesUC
	createScaleCodeAlias   createScaleCodeAliasUC
	deleteScaleCodeAlias   deleteScaleCodeAliasUC
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
	duplicateProduct duplicateProductUC,
	deleteProduct deleteProductUC,
	createVariant createVariantUC,
	updateVariant updateVariantUC,
	deleteVariant deleteVariantUC,
	createSKU createSKUUC,
	updateSKU updateSKUUC,
	deleteSKU deleteSKUUC,
	setManualCostPrice setManualCostPriceUC,
	clearCostPriceOverride clearCostPriceOverrideUC,
	uploadMedia uploadMediaUC,
	listProductMedia listProductMediaUC,
	deleteMedia deleteMediaUC,
	updateMedia updateMediaUC,
	resolveBarcode resolveBarcodeUC,
	getSKUMarginInsight getSKUMarginInsightUC,
	listCostPriceHistory listCostPriceHistoryUC,
	listMarginReport listMarginReportUC,
	listSKUPriceHistory listSKUPriceHistoryUC,
	listProductHistory listProductHistoryUC,
	listSKUCurrencyPrices listSKUCurrencyPricesUC,
	upsertSKUCurrencyPrice upsertSKUCurrencyPriceUC,
	deleteSKUCurrencyPrice deleteSKUCurrencyPriceUC,
	listBarcodeAliases listBarcodeAliasesUC,
	createBarcodeAlias createBarcodeAliasUC,
	deleteBarcodeAlias deleteBarcodeAliasUC,
	listScaleCodeAliases listScaleCodeAliasesUC,
	createScaleCodeAlias createScaleCodeAliasUC,
	deleteScaleCodeAlias deleteScaleCodeAliasUC,
	correctCost correctCostUC,
) *Handler {
	return &Handler{
		listBarcodeAliases:     listBarcodeAliases,
		createBarcodeAlias:     createBarcodeAlias,
		deleteBarcodeAlias:     deleteBarcodeAlias,
		listScaleCodeAliases:   listScaleCodeAliases,
		createScaleCodeAlias:   createScaleCodeAlias,
		deleteScaleCodeAlias:   deleteScaleCodeAlias,
		getSKUMarginInsight:    getSKUMarginInsight,
		listCostPriceHistory:   listCostPriceHistory,
		listMarginReport:       listMarginReport,
		listSKUPriceHistory:    listSKUPriceHistory,
		listProductHistory:     listProductHistory,
		listSKUCurrencyPrices:  listSKUCurrencyPrices,
		upsertSKUCurrencyPrice: upsertSKUCurrencyPrice,
		deleteSKUCurrencyPrice: deleteSKUCurrencyPrice,
		listCategories:         listCategories,
		createCategory:         createCategory,
		updateCategory:         updateCategory,
		deleteCategory:         deleteCategory,
		listProducts:           listProducts,
		createProduct:          createProduct,
		getProduct:             getProduct,
		updateProduct:          updateProduct,
		duplicateProduct:       duplicateProduct,
		deleteProduct:          deleteProduct,
		createVariant:          createVariant,
		updateVariant:          updateVariant,
		deleteVariant:          deleteVariant,
		createSKU:              createSKU,
		updateSKU:              updateSKU,
		deleteSKU:              deleteSKU,
		setManualCostPrice:     setManualCostPrice,
		clearCostPriceOverride: clearCostPriceOverride,
		uploadMedia:            uploadMedia,
		listProductMedia:       listProductMedia,
		deleteMedia:            deleteMedia,
		updateMedia:            updateMedia,
		resolveBarcode:         resolveBarcode,
		correctCost:            correctCost,
	}
}

var _ genpms.ServerInterface = (*Handler)(nil)

// Register is the production wiring: creates repos + use cases, then
// mounts the generated routes (matching the OpenAPI spec exactly).
// fileStorage is the media storage adapter (local disk or S3/MinIO — see
// infrastructure/storage), selected by the caller based on config.
func Register(rg gin.IRouter, db *gorm.DB, fileStorage pmsDomain.FileStorage, outboxRepo outbox.Repository) {
	categoryRepo := repositories.NewCategoryRepo(db)
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	mediaRepo := repositories.NewMediaRepo(db)
	listingRepo := repositories.NewProductChannelListingRepo(db)
	barcodeAliasRepo := repositories.NewBarcodeAliasRepo(db)
	scaleCodeAliasRepo := repositories.NewScaleCodeAliasRepo(db)
	costPriceHistoryRepo := repositories.NewSKUCostPriceHistoryRepo(db)
	priceHistoryRepo := repositories.NewSKUPriceHistoryRepo(db)
	productHistoryRepo := repositories.NewProductHistoryRepo(db)
	skuCurrencyPriceRepo := repositories.NewSKUCurrencyPriceRepo(db)
	settingsRepo := repositories.NewTenantSettingsRepo(db)
	warehouseRepo := repositories.NewWarehouseRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	discardRepo := repositories.NewDiscardRecordRepo(db)
	batchLister := invapp.NewListBatchesUseCase(db, batchRepo, movementRepo, warehouseRepo)
	discardLister := invapp.NewListDiscardsUseCase(db, discardRepo)

	h := NewHandler(
		apppms.NewListCategoriesUseCase(db, categoryRepo),
		apppms.NewCreateCategoryUseCase(db, categoryRepo),
		apppms.NewUpdateCategoryUseCase(db, categoryRepo, outboxRepo),
		apppms.NewDeleteCategoryUseCase(db, categoryRepo),
		apppms.NewListPOSProductCatalogUseCase(db, apppms.NewListProductsUseCase(db, productRepo), variantRepo, skuRepo, mediaRepo, barcodeAliasRepo, scaleCodeAliasRepo),
		apppms.NewCreateProductUseCase(db, productRepo, outboxRepo),
		apppms.NewGetProductUseCase(db, productRepo, variantRepo, skuRepo),
		apppms.NewUpdateProductUseCase(db, productRepo, variantRepo, listingRepo, outboxRepo, productHistoryRepo),
		apppms.NewDuplicateProductUseCase(db, productRepo, variantRepo, skuRepo),
		apppms.NewDeleteProductUseCase(db, productRepo, listingRepo, outboxRepo),
		apppms.NewCreateVariantUseCase(db, productRepo, variantRepo, outboxRepo),
		apppms.NewUpdateVariantUseCase(db, variantRepo, mediaRepo, productHistoryRepo, outboxRepo),
		apppms.NewDeleteVariantUseCase(db, variantRepo),
		apppms.NewCreateSKUUseCase(db, variantRepo, skuRepo, priceHistoryRepo, outboxRepo),
		apppms.NewUpdateSKUUseCase(db, variantRepo, skuRepo, outboxRepo, priceHistoryRepo),
		apppms.NewDeleteSKUUseCase(db, skuRepo),
		apppms.NewSetManualCostPriceUseCase(db, skuRepo),
		apppms.NewClearCostPriceOverrideUseCase(db, skuRepo),
		apppms.NewUploadMediaUseCase(db, productRepo, variantRepo, mediaRepo, fileStorage),
		apppms.NewListProductMediaUseCase(db, mediaRepo),
		apppms.NewDeleteMediaUseCase(db, mediaRepo, fileStorage),
		apppms.NewUpdateMediaUseCase(db, mediaRepo, outboxRepo),
		apppms.NewResolveBarcodeUseCase(db, skuRepo, barcodeAliasRepo, scaleCodeAliasRepo, variantRepo, productRepo),
		apppms.NewGetSKUMarginInsightUseCase(db, skuRepo, variantRepo, productRepo, categoryRepo, settingsRepo, batchLister, discardLister),
		apppms.NewListCostPriceHistoryUseCase(db, costPriceHistoryRepo),
		apppms.NewListMarginReportUseCase(db, skuRepo, variantRepo, productRepo, categoryRepo, settingsRepo),
		apppms.NewListSKUPriceHistoryUseCase(db, priceHistoryRepo),
		apppms.NewListProductHistoryUseCase(db, productHistoryRepo),
		apppms.NewListSKUCurrencyPricesUseCase(db, skuCurrencyPriceRepo),
		apppms.NewUpsertSKUCurrencyPriceUseCase(db, skuRepo, skuCurrencyPriceRepo),
		apppms.NewDeleteSKUCurrencyPriceUseCase(db, skuCurrencyPriceRepo),
		apppms.NewListBarcodeAliasesUseCase(db, barcodeAliasRepo),
		apppms.NewCreateBarcodeAliasUseCase(db, skuRepo, barcodeAliasRepo),
		apppms.NewDeleteBarcodeAliasUseCase(db, barcodeAliasRepo),
		apppms.NewListScaleCodeAliasesUseCase(db, scaleCodeAliasRepo),
		apppms.NewCreateScaleCodeAliasUseCase(db, skuRepo, scaleCodeAliasRepo),
		apppms.NewDeleteScaleCodeAliasUseCase(db, scaleCodeAliasRepo),
		apppms.NewCorrectCostUseCase(db, skuRepo, repositories.NewSKUCostPriceLedgerRepo(db), batchLister),
	)
	genpms.RegisterHandlers(rg, h)
}
