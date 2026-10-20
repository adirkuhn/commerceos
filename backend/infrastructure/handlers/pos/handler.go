package pos

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genpos "github.com/adirkuhn/commerceos/api/pos"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ── Use-case interfaces (defined at point of use — ISP) ───────────────────────

type listProductsUC interface {
	Execute(ctx context.Context, in apppms.ListProductsInput) (*apppms.ListPOSProductCatalogResult, error)
}
type listStockLevelsUC interface {
	Execute(ctx context.Context, in appinventory.ListStockLevelsInput) (*appinventory.ListStockLevelsResult, error)
}
type receivePOSSalesOrderUC interface {
	Execute(ctx context.Context, in appoms.ReceivePOSSalesOrderInput) (*omsDomain.SalesOrder, error)
}

type listWarehousesUC interface {
	Execute(ctx context.Context) ([]invDomain.Warehouse, error)
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listProducts         listProductsUC
	listStockLevels      listStockLevelsUC
	receivePOSSalesOrder receivePOSSalesOrderUC
	listWarehouses       listWarehousesUC
}

func NewHandler(listProducts listProductsUC, listStockLevels listStockLevelsUC, receivePOSSalesOrder receivePOSSalesOrderUC, listWarehouses listWarehousesUC) *Handler {
	return &Handler{listProducts: listProducts, listStockLevels: listStockLevels, receivePOSSalesOrder: receivePOSSalesOrder, listWarehouses: listWarehouses}
}

var _ genpos.ServerInterface = (*Handler)(nil)

// Register is the production wiring. Callers must mount rg on a router
// group using middleware.POSLicense, never middleware.JWT -- see
// api/pos/generate.go's doc comment.
func Register(rg gin.IRouter, db *gorm.DB) {
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	barcodeAliasRepo := repositories.NewBarcodeAliasRepo(db)
	scaleCodeAliasRepo := repositories.NewScaleCodeAliasRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	linkRepo := repositories.NewSalesOrderLineItemStockMovementRepo(db)
	paymentRepo := repositories.NewSalesOrderPaymentRepo(db)
	settingsRepo := repositories.NewTenantSettingsRepo(db)
	warehouseRepo := repositories.NewWarehouseRepo(db)
	mediaRepo := repositories.NewMediaRepo(db)

	allocateStock := appinventory.NewAllocateSaleStockUseCase(db, movementRepo, batchRepo, outboxRepo)
	listProducts := apppms.NewListProductsUseCase(db, productRepo)
	h := NewHandler(
		apppms.NewListPOSProductCatalogUseCase(db, listProducts, variantRepo, skuRepo, mediaRepo, barcodeAliasRepo, scaleCodeAliasRepo),
		appinventory.NewListStockLevelsUseCase(db, movementRepo, batchRepo, settingsRepo),
		appoms.NewReceivePOSSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, linkRepo, paymentRepo, skuRepo, allocateStock, outboxRepo),
		appinventory.NewListWarehousesUseCase(db, warehouseRepo),
	)
	genpos.RegisterHandlers(rg, h)
}

func (h *Handler) ListPOSProducts(c *gin.Context, params genpos.ListPOSProductsParams) {
	page, perPage := 1, 20
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	result, err := h.listProducts.Execute(c.Request.Context(), apppms.ListProductsInput{
		Page: page, PerPage: perPage, UpdatedAfter: params.UpdatedAfter,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	products := make([]genpos.POSProduct, len(result.Products))
	for i, p := range result.Products {
		products[i] = genpos.POSProduct{
			Id: p.ID, Name: p.Name, Slug: p.Slug, Status: string(p.Status),
			HasVariants: p.HasVariants, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
			ImageUrl: p.ImageURL, Skus: toGenPOSProductSKUs(p.SKUs),
		}
	}
	resp := genpos.POSProductList{Products: products, Total: result.Total}
	c.JSON(http.StatusOK, genpos.POSProductListResponse{Data: &resp})
}

func (h *Handler) ListPOSStockLevels(c *gin.Context, params genpos.ListPOSStockLevelsParams) {
	page, perPage := 1, 50
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	result, err := h.listStockLevels.Execute(c.Request.Context(), appinventory.ListStockLevelsInput{
		ChangedSince: params.ChangedSince, Page: page, PerPage: perPage,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	levels := make([]genpos.POSStockLevel, len(result.Levels))
	for i, l := range result.Levels {
		levels[i] = genpos.POSStockLevel{SkuId: l.SKUID, Total: l.Total, Syncable: l.Syncable}
	}
	resp := genpos.POSStockLevelList{Levels: levels, Total: result.Total}
	c.JSON(http.StatusOK, genpos.POSStockLevelListResponse{Data: &resp})
}

func toGenPOSProductSKUs(skus []apppms.POSProductSKU) []genpos.POSProductSKU {
	out := make([]genpos.POSProductSKU, len(skus))
	for i, s := range skus {
		pricingMode := genpos.POSProductSKUPricingMode(s.PricingMode)
		barcodeAliases := make([]genpos.POSBarcodeAliasEntry, len(s.BarcodeAliases))
		for j, a := range s.BarcodeAliases {
			barcodeAliases[j] = genpos.POSBarcodeAliasEntry{Code: a.Code, UnitsPerScan: a.UnitsPerScan}
		}
		scaleCodeAliases := s.ScaleCodeAliases
		out[i] = genpos.POSProductSKU{
			Id: s.ID, VariantId: &s.VariantID, VariantName: &s.VariantName,
			SkuCode: s.SKUCode, Barcode: &s.Barcode,
			PriceMinorUnits: s.PriceMinorUnits, SalePriceMinorUnits: s.SalePriceMinorUnits, SaleEndDate: s.SaleEndDate,
			CurrencyCode: s.CurrencyCode, Status: string(s.Status),
			ScaleCode: s.ScaleCode, PricingMode: &pricingMode, PricePerKiloMinorUnits: s.PricePerKiloMinorUnits,
			BarcodeAliases: &barcodeAliases, ScaleCodeAliases: &scaleCodeAliases,
		}
	}
	return out
}

func (h *Handler) ListPOSWarehouses(c *gin.Context) {
	warehouses, err := h.listWarehouses.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genpos.POSWarehouse, len(warehouses))
	for i, wh := range warehouses {
		out[i] = genpos.POSWarehouse{Id: wh.ID, Name: wh.Name, Active: wh.Active}
	}
	resp := genpos.POSWarehouseList{Warehouses: out}
	c.JSON(http.StatusOK, genpos.POSWarehouseListResponse{Data: &resp})
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func int64OrZero(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func toOMSAddress(a *genpos.POSAddress) omsDomain.Address {
	if a == nil {
		return omsDomain.Address{}
	}
	return omsDomain.Address{
		FirstName: strOrEmpty(a.FirstName), LastName: strOrEmpty(a.LastName), Company: strOrEmpty(a.Company),
		Address1: strOrEmpty(a.Address1), Address2: strOrEmpty(a.Address2), City: strOrEmpty(a.City),
		State: strOrEmpty(a.State), Postcode: strOrEmpty(a.Postcode), Country: strOrEmpty(a.Country),
		Email: strOrEmpty(a.Email), Phone: strOrEmpty(a.Phone),
	}
}

func toGenPOSAddress(a omsDomain.Address) genpos.POSAddress {
	return genpos.POSAddress{
		FirstName: &a.FirstName, LastName: &a.LastName, Company: &a.Company,
		Address1: &a.Address1, Address2: &a.Address2, City: &a.City,
		State: &a.State, Postcode: &a.Postcode, Country: &a.Country,
		Email: &a.Email, Phone: &a.Phone,
	}
}

func (h *Handler) CreatePOSSalesOrder(c *gin.Context) {
	var req genpos.CreatePOSSalesOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	lineItems := make([]appoms.ReceivePOSSalesOrderLineItemInput, len(req.LineItems))
	for i, li := range req.LineItems {
		lineItems[i] = appoms.ReceivePOSSalesOrderLineItemInput{
			SKUID: li.SkuId, Name: strOrEmpty(li.Name), SKUCode: strOrEmpty(li.SkuCode), Quantity: li.Quantity,
			WeightGrams:     li.WeightGrams,
			PriceMinorUnits: int64OrZero(li.PriceMinorUnits), SubtotalMinorUnits: int64OrZero(li.SubtotalMinorUnits),
			SubtotalTaxMinorUnits: int64OrZero(li.SubtotalTaxMinorUnits), TotalMinorUnits: int64OrZero(li.TotalMinorUnits),
			TotalTaxMinorUnits: int64OrZero(li.TotalTaxMinorUnits),
		}
	}
	var status omsDomain.SalesOrderStatus
	if req.Status != nil {
		status = omsDomain.SalesOrderStatus(*req.Status)
	}
	order, err := h.receivePOSSalesOrder.Execute(c.Request.Context(), appoms.ReceivePOSSalesOrderInput{
		ExternalOrderID: req.ExternalOrderId, Number: strOrEmpty(req.Number), OrderKey: strOrEmpty(req.OrderKey),
		Status: status, WarehouseID: req.WarehouseId, CustomerID: req.CustomerId,
		Billing: toOMSAddress(req.Billing), Shipping: toOMSAddress(req.Shipping), CustomerNote: strOrEmpty(req.CustomerNote),
		CurrencyCode: req.CurrencyCode, DiscountTotalMinorUnits: int64OrZero(req.DiscountTotalMinorUnits),
		ShippingTotalMinorUnits: int64OrZero(req.ShippingTotalMinorUnits), TaxTotalMinorUnits: int64OrZero(req.TaxTotalMinorUnits),
		TotalMinorUnits: req.TotalMinorUnits, PaymentMethod: strOrEmpty(req.PaymentMethod),
		PaymentMethodTitle: strOrEmpty(req.PaymentMethodTitle), TransactionID: strOrEmpty(req.TransactionId),
		Payments: toOMSPayments(req.Payments),
		DatePaid: req.DatePaid, DateCompleted: req.DateCompleted,
		OccurredAt: req.OccurredAt, LineItems: lineItems,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpos.POSSalesOrderResponse{Data: toGenPOSSalesOrder(order)})
}

func toGenPOSSalesOrder(order *omsDomain.SalesOrder) *genpos.POSSalesOrder {
	billing := toGenPOSAddress(order.Billing)
	shipping := toGenPOSAddress(order.Shipping)
	status := genpos.POSSalesOrderStatus(order.Status)
	return &genpos.POSSalesOrder{
		Id: order.ID, ExternalOrderId: order.ExternalOrderID, Number: &order.Number, OrderKey: &order.OrderKey,
		Status: status, WarehouseId: order.WarehouseID, CustomerId: order.CustomerID,
		Billing: &billing, Shipping: &shipping, CustomerNote: &order.CustomerNote,
		CurrencyCode: order.CurrencyCode, DiscountTotalMinorUnits: &order.DiscountTotalMinorUnits,
		ShippingTotalMinorUnits: &order.ShippingTotalMinorUnits, TaxTotalMinorUnits: &order.TaxTotalMinorUnits,
		TotalMinorUnits: order.TotalMinorUnits, PaymentMethod: &order.PaymentMethod,
		PaymentMethodTitle: &order.PaymentMethodTitle, TransactionId: &order.TransactionID,
		Payments: toGenPOSPayments(order.Payments),
		DatePaid: order.DatePaid, DateCompleted: order.DateCompleted,
		OccurredAt: order.OccurredAt, CreatedAt: order.CreatedAt,
	}
}

func toOMSPayments(payments *[]genpos.CreatePOSSalesOrderPayment) omsDomain.SalesOrderPayments {
	if payments == nil {
		return nil
	}
	out := make(omsDomain.SalesOrderPayments, len(*payments))
	for i, p := range *payments {
		out[i] = omsDomain.SalesOrderPayment{Method: p.Method, AmountMinorUnits: p.AmountMinorUnits, TransactionID: strOrEmpty(p.TransactionId)}
	}
	return out
}

func toGenPOSPayments(payments omsDomain.SalesOrderPayments) *[]genpos.POSSalesOrderPayment {
	out := make([]genpos.POSSalesOrderPayment, len(payments))
	for i, p := range payments {
		out[i] = genpos.POSSalesOrderPayment{Method: p.Method, AmountMinorUnits: p.AmountMinorUnits, TransactionId: p.TransactionID}
	}
	return &out
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, omsDomain.ErrEmptyLineItems), errors.Is(err, omsDomain.ErrAdhocLineItemRequiresName),
		errors.Is(err, omsDomain.ErrWeighedLineItemQuantityMustBeOne), errors.Is(err, omsDomain.ErrPaymentsTotalMismatch),
		errors.Is(err, pmsDomain.ErrSKUNotFound), errors.Is(err, invDomain.ErrInsufficientStock),
		errors.Is(err, invDomain.ErrInvalidQuantity):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
	default:
		log.Printf("pos: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
